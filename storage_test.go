package main

import (
	"strings"
	"testing"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndScoreRoundTrip(t *testing.T) {
	s := newTestStorage(t)
	a, err := s.Create(&AnimeSeries{EnglishName: "Test Show", Score: 8.5, Status: "watched"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Score != 8.5 {
		t.Fatalf("score=%v want 8.5", a.Score)
	}
	got, ok := s.GetByID(a.ID)
	if !ok {
		t.Fatal("missing series")
	}
	if got.Score != 8.5 || got.EnglishName != "Test Show" {
		t.Fatalf("got %+v", got)
	}
}

func TestImportClampsOutOfRangeScore(t *testing.T) {
	s := newTestStorage(t)
	csv := csvHeader + "\n" + `Over,,,,,,,,99,,,,,,`
	res, err := s.ImportCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 {
		t.Fatalf("count=%d %v", res.Count, res.Skipped)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected score clamp warning")
	}
	got := s.GetAll()
	if len(got) != 1 || got[0].Score != 0 {
		t.Fatalf("score should clamp to 0, got %+v", got)
	}
}

func TestDateAcceptsSlashAndHyphen(t *testing.T) {
	s := newTestStorage(t)
	csv := csvHeader + "\n" + `SlashDate,,,,,,,,0,,,,,,`
	csv += "\n" + strings.Join([]string{
		"SlashDate2", "", "", "", "", "", "2020/01", "2020/02/03", "1", "", "", "", "false", "false",
	}, ",")
	res, err := s.ImportCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 2 {
		t.Fatalf("count=%d skipped=%v", res.Count, res.Skipped)
	}
}

func TestToggleFavorite(t *testing.T) {
	s := newTestStorage(t)
	a, _ := s.Create(&AnimeSeries{EnglishName: "Fav"})
	if a.Favorite {
		t.Fatal("expected not favorite")
	}
	a, err := s.ToggleFavorite(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Favorite {
		t.Fatal("expected favorite")
	}
	a, _ = s.ToggleFavorite(a.ID)
	if a.Favorite {
		t.Fatal("expected unfavorite")
	}
}

func TestExportImportCSVContract(t *testing.T) {
	s := newTestStorage(t)
	in := &AnimeSeries{
		EnglishName:  "CSV Show",
		JapaneseName: "日本語",
		Synonyms:     []string{"Alt"},
		Status:       "to-watch",
		Score:        7.5,
		Review:       "line1\nline2",
		Synopsis:     "A, quoted",
		Tags:         []string{"drama", "action"},
		Favorite:     true,
		Owned:        true,
	}
	if _, err := s.Create(in); err != nil {
		t.Fatal(err)
	}
	csv, err := s.ExportCSV()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(csv, csvHeader+"\n") {
		t.Fatalf("bad header: %q", strings.Split(csv, "\n")[0])
	}

	s2 := newTestStorage(t)
	res, err := s2.ImportCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 {
		t.Fatalf("import count=%d skipped=%v warnings=%v", res.Count, res.Skipped, res.Warnings)
	}
	list := s2.GetAll()
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	got := list[0]
	if got.Score != 7.5 {
		t.Fatalf("score=%v", got.Score)
	}
	if !got.Favorite || !got.Owned {
		t.Fatalf("favorite/owned lost: %+v", got)
	}
	if got.Review != "line1\nline2" {
		t.Fatalf("review=%q", got.Review)
	}
}

func TestImportValidation(t *testing.T) {
	s := newTestStorage(t)
	csv := strings.Join([]string{
		csvHeader,
		`,,,,,,,,,,,,`,                 // empty name
		`Bad,,,,,,-not-a-date,,,,,,`,    // invalid date
		`NoStatus,,,flying,,,,,,,,,`,    // invalid status
		`Short,row`,                     // short row
		`Good Name,,,,, ,,,, ,,,,false,false`,
	}, "\n")
	res, err := s.ImportCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 {
		t.Fatalf("count=%d created=%v skipped=%v", res.Count, res.Created, res.Skipped)
	}
	if len(res.Skipped) < 3 {
		t.Fatalf("expected skips, got %v", res.Skipped)
	}
}

func TestSettings(t *testing.T) {
	s := newTestStorage(t)
	if s.GetTheme() != "dark" {
		t.Fatal("default theme")
	}
	if err := s.SetTheme("blue-dark"); err != nil {
		t.Fatal(err)
	}
	if s.GetTheme() != "blue-dark" {
		t.Fatal("theme not saved")
	}
	if err := s.SetAudioFile("pick.mp3"); err != nil {
		t.Fatal(err)
	}
	if s.GetAudioFile() != "pick.mp3" {
		t.Fatal("audio not saved")
	}
	if err := s.SetMuteAudio(true); err != nil {
		t.Fatal(err)
	}
	if !s.GetMuteAudio() {
		t.Fatal("mute not saved")
	}
}

func TestBidirectionalLinks(t *testing.T) {
	s := newTestStorage(t)
	a, _ := s.Create(&AnimeSeries{EnglishName: "A"})
	b, _ := s.Create(&AnimeSeries{EnglishName: "B"})
	a.LinkedIDs = []string{b.ID}
	if _, err := s.Update(a.ID, a); err != nil {
		t.Fatal(err)
	}
	gotB, _ := s.GetByID(b.ID)
	if !contains(gotB.LinkedIDs, a.ID) {
		t.Fatalf("B should link back to A: %v", gotB.LinkedIDs)
	}
}

func TestReadAudioFileRejectsPaths(t *testing.T) {
	a := &App{dataRoot: t.TempDir()}
	if err := ensureDataLayout(a.dataRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ReadAudioFile("../secret.mp3"); err == nil {
		t.Fatal("expected reject parent path")
	}
	if _, err := a.ReadAudioFile("sub/x.mp3"); err == nil {
		t.Fatal("expected reject nested path")
	}
	src, err := a.ReadAudioFile("pick.mp3")
	if err != nil || src != "pick.mp3" {
		t.Fatalf("pick.mp3 -> %q %v", src, err)
	}
}
