package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	app      *application.App
	storage  *Storage
	dataRoot string
}

func NewApp() *App {
	return &App{}
}

func (a *App) setApp(app *application.App) {
	a.app = app
}

func (a *App) dialogApp() *application.App {
	if a.app != nil {
		return a.app
	}
	return application.Get()
}

func (a *App) initStorage(dataRoot string) {
	a.dataRoot = dataRoot
	if err := ensureDataLayout(dataRoot); err != nil {
		return
	}

	a.backupDatabase(dataRoot)
	storage, err := NewStorage(dataRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Naraberu: storage: %v\n", err)
		return
	}
	a.storage = storage
}

func (a *App) backupDatabase(dataRoot string) {
	dbPath := dbPathIn(dataRoot)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return
	}

	db, err := sql.Open("sqlite", dbPath)
	if err == nil {
		db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		db.Close()
	}

	backupDir := backupsDirIn(dataRoot)
	os.MkdirAll(backupDir, 0755)

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("naraberu_%s.db", timestamp))

	src, err := os.Open(dbPath)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return
	}
	defer dst.Close()

	io.Copy(dst, src)

	entries, _ := os.ReadDir(backupDir)
	var backups []os.DirEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "naraberu_") && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, e)
		}
	}
	if len(backups) > 5 {
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].Name() > backups[j].Name()
		})
		for _, b := range backups[5:] {
			os.Remove(filepath.Join(backupDir, b.Name()))
		}
	}
}

func (a *App) thumbDir() string {
	if a.dataRoot != "" {
		return thumbnailsDirIn(a.dataRoot)
	}
	return thumbnailsDirIn(resolveDataRoot())
}

func (a *App) soundsDir() string {
	if a.dataRoot != "" {
		return soundsDirIn(a.dataRoot)
	}
	return soundsDirIn(resolveDataRoot())
}

func (a *App) GetAllSeries() []*AnimeSeries {
	if a.storage == nil {
		return []*AnimeSeries{}
	}
	return a.storage.GetAll()
}

func (a *App) SearchSeries(filter SearchFilter) []*AnimeSeries {
	if a.storage == nil {
		return []*AnimeSeries{}
	}
	series := a.storage.Search(filter)
	if filter.FavoritesOnly {
		filtered := make([]*AnimeSeries, 0, len(series))
		for _, s := range series {
			if s.Favorite {
				filtered = append(filtered, s)
			}
		}
		series = filtered
	}
	if filter.OwnedOnly {
		filtered := make([]*AnimeSeries, 0, len(series))
		for _, s := range series {
			if s.Owned {
				filtered = append(filtered, s)
			}
		}
		series = filtered
	}
	return series
}

func (a *App) GetSeries(id string) (*AnimeSeries, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	s, ok := a.storage.GetByID(id)
	if !ok {
		return nil, fmt.Errorf("series not found")
	}
	return s, nil
}

func (a *App) CreateSeries(s *AnimeSeries) (*AnimeSeries, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.Create(s)
}

func (a *App) UpdateSeries(id string, s *AnimeSeries) (*AnimeSeries, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.Update(id, s)
}

func (a *App) DeleteSeries(id string) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	if s, ok := a.storage.GetByID(id); ok && s.ThumbnailPath != "" {
		if pathInsideDir(a.thumbDir(), s.ThumbnailPath) {
			os.Remove(s.ThumbnailPath)
		}
	}
	return a.storage.Delete(id)
}

func (a *App) GetAllTags() []string {
	if a.storage == nil {
		return []string{}
	}
	return a.storage.GetAllTags()
}

func (a *App) GetTheme() string {
	if a.storage == nil {
		return "dark"
	}
	return a.storage.GetTheme()
}

func (a *App) SetTheme(theme string) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	return a.storage.SetTheme(theme)
}

func (a *App) GetAudioFile() string {
	if a.storage == nil {
		return ""
	}
	return a.storage.GetAudioFile()
}

func (a *App) SetAudioFile(audioFile string) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	return a.storage.SetAudioFile(audioFile)
}

func (a *App) GetMuteAudio() bool {
	if a.storage == nil {
		return false
	}
	return a.storage.GetMuteAudio()
}

func (a *App) SetMuteAudio(mute bool) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	return a.storage.SetMuteAudio(mute)
}

func (a *App) ListAudioFiles() []string {
	names := make([]string, 0, 4)
	seen := map[string]bool{}
	for _, name := range []string{"pick.mp3"} {
		names = append(names, name)
		seen[name] = true
	}
	entries, err := os.ReadDir(a.soundsDir())
	if err != nil {
		return names
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".mp3") && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	return names
}

// ReadAudioFile returns a data URL for a user sound in data/sounds/.
// The bundled default (pick.mp3) is served as a normal asset URL.
func (a *App) ReadAudioFile(name string) (string, error) {
	if name == "" || name == "none" {
		return "", nil
	}
	if name == "pick.mp3" {
		return "pick.mp3", nil
	}
	base := filepath.Base(name)
	if base != name || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid audio file name")
	}
	if !strings.HasSuffix(strings.ToLower(base), ".mp3") {
		return "", fmt.Errorf("unsupported audio format")
	}
	path := filepath.Join(a.soundsDir(), base)
	if !pathInsideDir(a.soundsDir(), path) {
		return "", fmt.Errorf("audio path outside data directory")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (a *App) ToggleFavorite(id string) (*AnimeSeries, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ToggleFavorite(id)
}

func (a *App) ToggleOwned(id string) (*AnimeSeries, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return a.storage.ToggleOwned(id)
}

func (a *App) PickThumbnail() (string, error) {
	filePath, err := a.dialogApp().Dialog.OpenFile().
		SetTitle("Select Thumbnail").
		CanChooseFiles(true).
		AddFilter("Images", "*.jpg;*.jpeg;*.png;*.gif;*.webp").
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", nil
	}
	thumbDir := a.thumbDir()
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", err
	}
	ext := filepath.Ext(filePath)
	destPath := filepath.Join(thumbDir, uuid.New().String()+ext)
	src, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return destPath, nil
}

func (a *App) ReadThumbnail(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if !pathInsideDir(a.thumbDir(), path) {
		return "", fmt.Errorf("thumbnail path outside data directory")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(path)
	mime := "image/jpeg"
	switch ext {
	case ".png":
		mime = "image/png"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (a *App) ExportCSV() (string, error) {
	if a.storage == nil {
		return "", fmt.Errorf("storage not initialized")
	}
	return a.storage.ExportCSV()
}

func (a *App) ImportCSV(data string) (ImportResult, error) {
	if a.storage == nil {
		return ImportResult{}, fmt.Errorf("storage not initialized")
	}
	result, err := a.storage.ImportCSV(data)
	if err != nil {
		return result, err
	}

	for _, id := range result.Created {
		s, ok := a.storage.GetByID(id)
		if !ok || s.ThumbnailPath == "" {
			continue
		}
		if !strings.HasPrefix(s.ThumbnailPath, "http://") && !strings.HasPrefix(s.ThumbnailPath, "https://") {
			continue
		}
		localPath, fetchErr := fetchThumbnail(s.ThumbnailPath, a.thumbDir())
		if fetchErr != nil {
			continue
		}
		s.ThumbnailPath = localPath
		a.storage.Update(id, s)
	}

	return result, nil
}

func fetchThumbnail(url string, thumbDir string) (string, error) {
	lower := strings.ToLower(url)
	validExt := false
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		if strings.HasSuffix(lower, ext) {
			validExt = true
			break
		}
	}
	if !validExt {
		return "", fmt.Errorf("URL does not point to a supported image format")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Naraberu/1.0 (anime tracker)")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "image/") && !strings.Contains(contentType, "octet-stream") {
		return "", fmt.Errorf("not an image: %s", contentType)
	}

	if thumbDir == "" {
		thumbDir = thumbnailsDirIn(resolveDataRoot())
	}
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".jpg"
	}
	destPath := filepath.Join(thumbDir, uuid.New().String()+ext)

	limitedReader := io.LimitReader(resp.Body, 5*1024*1024)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return "", err
	}
	return destPath, nil
}
