package main

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Storage struct {
	db      *sql.DB
	allTags []string
}

const schema = `
CREATE TABLE IF NOT EXISTS series (
    id TEXT PRIMARY KEY,
    english_name TEXT NOT NULL DEFAULT '',
    japanese_name TEXT NOT NULL DEFAULT '',
    synonyms TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    linked_ids TEXT NOT NULL DEFAULT '',
    thumbnail_path TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL DEFAULT '',
    end_date TEXT NOT NULL DEFAULT '',
    score REAL NOT NULL DEFAULT 0,
    review TEXT NOT NULL DEFAULT '',
    synopsis TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '',
    favorite INTEGER NOT NULL DEFAULT 0,
    owned INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    normalized_text TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
`

const seriesCols = `id, english_name, japanese_name, synonyms, status, linked_ids, thumbnail_path, start_date, end_date, score, review, synopsis, tags, favorite, owned, created_at, updated_at`

func NewStorage(dataRoot string) (*Storage, error) {
	if err := os.MkdirAll(dataRoot, 0755); err != nil {
		return nil, err
	}

	dbPath := dbPathIn(dataRoot)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}

	db.Exec("PRAGMA user_version=1")

	s := &Storage{db: db}
	s.rebuildTags()
	s.populateNormalizedText()
	return s, nil
}

func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Storage) rebuildTags() {
	rows, err := s.db.Query("SELECT DISTINCT tags FROM series WHERE tags != ''")
	if err != nil {
		return
	}
	defer rows.Close()

	tagSet := make(map[string]bool)
	lowerMap := make(map[string]string)
	for rows.Next() {
		var tagsStr string
		if err := rows.Scan(&tagsStr); err != nil {
			continue
		}
		for _, t := range strings.Split(tagsStr, ";") {
			t = strings.TrimSpace(t)
			if t != "" {
				lower := strings.ToLower(t)
				if !tagSet[lower] {
					tagSet[lower] = true
					lowerMap[lower] = t
				}
			}
		}
	}
	s.allTags = make([]string, 0, len(tagSet))
	for _, t := range lowerMap {
		s.allTags = append(s.allTags, t)
	}
	sort.Strings(s.allTags)
}

func buildNormalizedText(a *AnimeSeries) string {
	var parts []string
	parts = append(parts, a.EnglishName, a.JapaneseName)
	parts = append(parts, a.Synonyms...)
	parts = append(parts, a.Tags...)
	if a.Synopsis != "" {
		parts = append(parts, a.Synopsis)
	}
	if a.Review != "" {
		parts = append(parts, a.Review)
	}
	return normalizeSearch(strings.Join(parts, " "))
}

func (s *Storage) populateNormalizedText() {
	rows, err := s.db.Query("SELECT id, english_name, japanese_name, synonyms, review, synopsis, tags FROM series")
	if err != nil {
		return
	}
	defer rows.Close()

	type row struct {
		id, englishName, japaneseName, synonyms, review, synopsis, tags string
	}
	var entries []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.englishName, &r.japaneseName, &r.synonyms, &r.review, &r.synopsis, &r.tags); err != nil {
			continue
		}
		entries = append(entries, r)
	}

	for _, r := range entries {
		a := &AnimeSeries{
			EnglishName:  r.englishName,
			JapaneseName: r.japaneseName,
			Synonyms:     splitField(r.synonyms),
			Review:       r.review,
			Synopsis:     r.synopsis,
			Tags:         splitField(r.tags),
		}
		nt := buildNormalizedText(a)
		s.db.Exec("UPDATE series SET normalized_text = ? WHERE id = ?", nt, r.id)
	}
}

func (s *Storage) GetAllTags() []string {
	return s.allTags
}

func (s *Storage) rowToSeries(row interface{ Scan(dest ...any) error }) (*AnimeSeries, error) {
	var a AnimeSeries
	var synonyms, linkedIDs, tags, createdAt, updatedAt string
	var fav, owned int
	err := row.Scan(&a.ID, &a.EnglishName, &a.JapaneseName, &synonyms, &a.Status, &linkedIDs, &a.ThumbnailPath, &a.StartDate, &a.EndDate, &a.Score, &a.Review, &a.Synopsis, &tags, &fav, &owned, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	a.Synonyms = splitField(synonyms)
	a.LinkedIDs = splitField(linkedIDs)
	a.Tags = splitField(tags)
	a.Favorite = fav == 1
	a.Owned = owned == 1
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &a, nil
}

func (s *Storage) getByID(id string) (*AnimeSeries, bool) {
	row := s.db.QueryRow("SELECT "+seriesCols+" FROM series WHERE id = ?", id)
	a, err := s.rowToSeries(row)
	if err != nil {
		return nil, false
	}
	return a, true
}

func (s *Storage) GetByID(id string) (*AnimeSeries, bool) {
	return s.getByID(id)
}

func (s *Storage) Create(a *AnimeSeries) (*AnimeSeries, error) {
	a.ID = uuid.New().String()
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	if a.Tags == nil {
		a.Tags = []string{}
	}
	if a.Synonyms == nil {
		a.Synonyms = []string{}
	}
	if a.LinkedIDs == nil {
		a.LinkedIDs = []string{}
	}

	fav := boolToInt(a.Favorite)
	nt := buildNormalizedText(a)
	_, err := s.db.Exec(`INSERT INTO series (id, english_name, japanese_name, synonyms, status, linked_ids, thumbnail_path, start_date, end_date, score, review, synopsis, tags, favorite, owned, created_at, updated_at, normalized_text) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.EnglishName, a.JapaneseName, strings.Join(a.Synonyms, ";"), a.Status, strings.Join(a.LinkedIDs, ";"), a.ThumbnailPath, a.StartDate, a.EndDate, a.Score, a.Review, a.Synopsis, strings.Join(a.Tags, ";"), fav, boolToInt(a.Owned), a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339), nt)
	if err != nil {
		return nil, err
	}
	s.rebuildTags()
	return a, nil
}

func (s *Storage) Update(id string, update *AnimeSeries) (*AnimeSeries, error) {
	existing, ok := s.getByID(id)
	if !ok {
		return nil, fmt.Errorf("series not found")
	}

	oldLinked := make(map[string]bool)
	for _, lid := range existing.LinkedIDs {
		oldLinked[lid] = true
	}
	newLinked := make(map[string]bool)
	for _, lid := range update.LinkedIDs {
		newLinked[lid] = true
	}

	changed := make(map[string]*AnimeSeries)
	for lid := range newLinked {
		if !oldLinked[lid] {
			if target, ok := s.getByID(lid); ok {
				if !contains(target.LinkedIDs, id) {
					target.LinkedIDs = append(target.LinkedIDs, id)
					changed[lid] = target
				}
			}
		}
	}
	for lid := range oldLinked {
		if !newLinked[lid] {
			if target, ok := s.getByID(lid); ok {
				target.LinkedIDs = remove(target.LinkedIDs, id)
				changed[lid] = target
			}
		}
	}

	for lid, target := range changed {
		if _, err := s.db.Exec("UPDATE series SET linked_ids = ? WHERE id = ?", strings.Join(target.LinkedIDs, ";"), lid); err != nil {
			return nil, fmt.Errorf("failed to update linked series %s: %w", lid, err)
		}
	}

	existing.EnglishName = update.EnglishName
	existing.JapaneseName = update.JapaneseName
	existing.Synonyms = update.Synonyms
	existing.Status = update.Status
	existing.LinkedIDs = update.LinkedIDs
	existing.ThumbnailPath = update.ThumbnailPath
	existing.StartDate = update.StartDate
	existing.EndDate = update.EndDate
	existing.Score = update.Score
	existing.Review = update.Review
	existing.Synopsis = update.Synopsis
	existing.Tags = update.Tags
	existing.Favorite = update.Favorite
	existing.Owned = update.Owned
	existing.UpdatedAt = time.Now()

	fav := boolToInt(existing.Favorite)
	nt := buildNormalizedText(existing)
	_, err := s.db.Exec(`UPDATE series SET english_name=?, japanese_name=?, synonyms=?, status=?, linked_ids=?, thumbnail_path=?, start_date=?, end_date=?, score=?, review=?, synopsis=?, tags=?, favorite=?, owned=?, updated_at=?, normalized_text=? WHERE id=?`,
		existing.EnglishName, existing.JapaneseName, strings.Join(existing.Synonyms, ";"), existing.Status, strings.Join(existing.LinkedIDs, ";"), existing.ThumbnailPath, existing.StartDate, existing.EndDate, existing.Score, existing.Review, existing.Synopsis, strings.Join(existing.Tags, ";"), fav, boolToInt(existing.Owned), existing.UpdatedAt.Format(time.RFC3339), nt, id)
	if err != nil {
		return nil, err
	}
	s.rebuildTags()
	return existing, nil
}

func (s *Storage) ToggleFavorite(id string) (*AnimeSeries, error) {
	existing, ok := s.getByID(id)
	if !ok {
		return nil, fmt.Errorf("series not found")
	}
	existing.Favorite = !existing.Favorite
	if _, err := s.db.Exec("UPDATE series SET favorite = ? WHERE id = ?", boolToInt(existing.Favorite), id); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Storage) ToggleOwned(id string) (*AnimeSeries, error) {
	existing, ok := s.getByID(id)
	if !ok {
		return nil, fmt.Errorf("series not found")
	}
	existing.Owned = !existing.Owned
	if _, err := s.db.Exec("UPDATE series SET owned = ? WHERE id = ?", boolToInt(existing.Owned), id); err != nil {
		return nil, err
	}
	return existing, nil
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func remove(slice []string, val string) []string {
	var result []string
	for _, s := range slice {
		if s != val {
			result = append(result, s)
		}
	}
	return result
}

func (s *Storage) Delete(id string) error {
	existing, ok := s.getByID(id)
	if !ok {
		return fmt.Errorf("series not found")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, lid := range existing.LinkedIDs {
		if target, ok := s.getByID(lid); ok {
			target.LinkedIDs = remove(target.LinkedIDs, id)
			tx.Exec("UPDATE series SET linked_ids = ? WHERE id = ?", strings.Join(target.LinkedIDs, ";"), lid)
		}
	}

	if _, err := tx.Exec("DELETE FROM series WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Storage) Search(filter SearchFilter) []*AnimeSeries {
	var whereClauses []string
	var args []interface{}

	if filter.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filter.Status)
	}
	for _, tag := range filter.Tags {
		whereClauses = append(whereClauses, "tags LIKE ?")
		args = append(args, "%"+tag+"%")
	}
	if filter.LinkedOnly {
		whereClauses = append(whereClauses, "linked_ids != ''")
	}
	if filter.Query != "" {
		q := "%" + normalizeSearch(filter.Query) + "%"
		whereClauses = append(whereClauses, "normalized_text LIKE ?")
		args = append(args, q)
	}

	query := "SELECT " + seriesCols + " FROM series"
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var orderBy string
	switch filter.SortBy {
	case "englishName":
		orderBy = `LOWER(CASE WHEN english_name LIKE 'The %' THEN SUBSTR(english_name, 5)
			WHEN english_name LIKE 'A %' THEN SUBSTR(english_name, 3)
			WHEN english_name LIKE 'An %' THEN SUBSTR(english_name, 4)
			ELSE english_name END)`
	case "japaneseName":
		orderBy = "LOWER(japanese_name)"
	case "score":
		orderBy = "score"
	case "startDate":
		orderBy = "start_date"
	case "endDate":
		orderBy = "end_date"
	case "status":
		orderBy = "status"
	default:
		orderBy = "updated_at"
	}

	query += " ORDER BY " + orderBy
	if filter.SortDesc {
		query += " DESC"
	} else {
		query += " ASC"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return make([]*AnimeSeries, 0)
	}
	defer rows.Close()

	results := make([]*AnimeSeries, 0)
	for rows.Next() {
		a, err := s.rowToSeries(rows)
		if err != nil {
			continue
		}
		results = append(results, a)
	}
	return results
}

func (s *Storage) GetAll() []*AnimeSeries {
	return s.Search(SearchFilter{})
}

const csvHeader = "englishName,japaneseName,synonyms,status,linkedIds,thumbnailPath,startDate,endDate,score,review,synopsis,tags,favorite,owned"

func (s *Storage) ExportCSV() (string, error) {
	var sb strings.Builder
	sb.WriteString(csvHeader)
	sb.WriteString("\n")
	for _, a := range s.Search(SearchFilter{}) {
		sb.WriteString(csvEscape(a.EnglishName))
		sb.WriteString(",")
		sb.WriteString(csvEscape(a.JapaneseName))
		sb.WriteString(",")
		sb.WriteString(csvEscape(strings.Join(a.Synonyms, ";")))
		sb.WriteString(",")
		sb.WriteString(csvEscape(a.Status))
		sb.WriteString(",")
		sb.WriteString(csvEscape(strings.Join(a.LinkedIDs, ";")))
		sb.WriteString(",")
		sb.WriteString(csvEscape(a.ThumbnailPath))
		sb.WriteString(",")
		sb.WriteString(csvEscape(a.StartDate))
		sb.WriteString(",")
		sb.WriteString(csvEscape(a.EndDate))
		sb.WriteString(",")
		sb.WriteString(fmt.Sprintf("%.1f", a.Score))
		sb.WriteString(",")
		sb.WriteString(csvEscape(a.Review))
		sb.WriteString(",")
		sb.WriteString(csvEscape(a.Synopsis))
		sb.WriteString(",")
		sb.WriteString(csvEscape(strings.Join(a.Tags, ";")))
		sb.WriteString(",")
		if a.Favorite {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
		sb.WriteString(",")
		if a.Owned {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func splitCSVLines(data string) []string {
	var lines []string
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(data); i++ {
		ch := data[i]
		if ch == '"' {
			inQuote = !inQuote
			current.WriteByte(ch)
		} else if (ch == '\n' || ch == '\r') && !inQuote {
			if ch == '\r' && i+1 < len(data) && data[i+1] == '\n' {
				i++
			}
			lines = append(lines, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func (s *Storage) existsByName(name string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM series WHERE LOWER(english_name) = LOWER(?)", name).Scan(&count)
	return count > 0
}

var validStatuses = map[string]bool{
	"to-watch":    true,
	"watched":     true,
	"in-progress": true,
	"abandoned":   true,
}

func isValidDateString(s string) bool {
	if s == "" {
		return true
	}
	// Plan contract: YYYY, YYYY/MM, YYYY/MM/DD (also accept YYYY-MM, YYYY-MM-DD).
	normalized := strings.ReplaceAll(s, "/", "-")
	parts := strings.Split(normalized, "-")
	if len(parts) == 1 && len(parts[0]) == 4 {
		return true
	}
	if len(parts) == 2 && len(parts[0]) == 4 && len(parts[1]) == 2 {
		return true
	}
	if len(parts) == 3 && len(parts[0]) == 4 && len(parts[1]) == 2 && len(parts[2]) == 2 {
		return true
	}
	return false
}

func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inQuote {
			if ch == '"' && i+1 < len(line) && line[i+1] == '"' {
				current.WriteByte('"')
				i++
			} else if ch == '"' {
				inQuote = false
			} else {
				current.WriteByte(ch)
			}
		} else {
			if ch == '"' {
				inQuote = true
			} else if ch == ',' {
				fields = append(fields, current.String())
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		}
	}
	fields = append(fields, current.String())
	return fields
}

func splitField(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ";")
}

func csvColumnIndex(header []string, name string) int {
	for i, h := range header {
		if strings.TrimSpace(h) == name {
			return i
		}
	}
	return -1
}

func fieldAt(fields []string, idx int) string {
	if idx < 0 || idx >= len(fields) {
		return ""
	}
	return fields[idx]
}

// ImportCSV follows the shared column contract (see docs). favorite/owned are
// resolved by header name when present; other columns by header name or index.
func (s *Storage) ImportCSV(data string) (ImportResult, error) {
	lines := splitCSVLines(data)
	if len(lines) < 2 {
		return ImportResult{}, fmt.Errorf("CSV file is empty or has no data rows")
	}

	header := strings.Split(lines[0], ",")
	col := func(name string, fallback int) int {
		if i := csvColumnIndex(header, name); i >= 0 {
			return i
		}
		return fallback
	}
	idxName := col("englishName", 0)
	idxJapanese := col("japaneseName", 1)
	idxSynonyms := col("synonyms", 2)
	idxStatus := col("status", 3)
	idxLinked := col("linkedIds", 4)
	idxThumb := col("thumbnailPath", 5)
	idxStart := col("startDate", 6)
	idxEnd := col("endDate", 7)
	idxScore := col("score", 8)
	idxReview := col("review", 9)
	idxSynopsis := col("synopsis", 10)
	idxTags := col("tags", 11)
	idxFav := csvColumnIndex(header, "favorite")
	idxOwned := csvColumnIndex(header, "owned")

	result := ImportResult{}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := parseCSVLine(line)
		if len(fields) < 12 {
			result.Skipped = append(result.Skipped, fmt.Sprintf("(row too short: %d fields)", len(fields)))
			continue
		}

		name := strings.TrimSpace(fieldAt(fields, idxName))
		if name == "" {
			result.Skipped = append(result.Skipped, "(empty English name)")
			continue
		}

		status := strings.TrimSpace(fieldAt(fields, idxStatus))
		if status != "" && !validStatuses[status] {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (invalid status: %q)", name, status))
			continue
		}

		startDate := strings.TrimSpace(fieldAt(fields, idxStart))
		if !isValidDateString(startDate) {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (invalid startDate: %q)", name, startDate))
			continue
		}
		endDate := strings.TrimSpace(fieldAt(fields, idxEnd))
		if !isValidDateString(endDate) {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s (invalid endDate: %q)", name, endDate))
			continue
		}

		a := &AnimeSeries{
			EnglishName:   name,
			JapaneseName:  strings.TrimSpace(fieldAt(fields, idxJapanese)),
			Synonyms:      splitField(fieldAt(fields, idxSynonyms)),
			Status:        status,
			LinkedIDs:     splitField(fieldAt(fields, idxLinked)),
			ThumbnailPath: strings.TrimSpace(fieldAt(fields, idxThumb)),
			StartDate:     startDate,
			EndDate:       endDate,
			Review:        fieldAt(fields, idxReview),
			Synopsis:      fieldAt(fields, idxSynopsis),
			Tags:          splitField(fieldAt(fields, idxTags)),
		}

		if raw := strings.TrimSpace(fieldAt(fields, idxScore)); raw != "" {
			if _, err := fmt.Sscanf(raw, "%f", &a.Score); err != nil {
				a.Score = 0
			}
		}
		if a.Score < 0 || a.Score > 10 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: score %.1f out of range, clamped to 0", name, a.Score))
			a.Score = 0
		}

		if len(a.LinkedIDs) > 0 {
			var valid []string
			for _, lid := range a.LinkedIDs {
				lid = strings.TrimSpace(lid)
				if len(lid) == 36 && strings.Count(lid, "-") == 4 {
					valid = append(valid, lid)
				}
			}
			if valid == nil {
				valid = []string{}
			}
			if len(valid) != len(a.LinkedIDs) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: linkedIds had invalid entries, cleared", name))
			}
			a.LinkedIDs = valid
		}

		if idxFav >= 0 {
			a.Favorite = strings.TrimSpace(fieldAt(fields, idxFav)) == "true"
		}
		if idxOwned >= 0 {
			a.Owned = strings.TrimSpace(fieldAt(fields, idxOwned)) == "true"
		}

		if s.existsByName(a.EnglishName) {
			result.Skipped = append(result.Skipped, a.EnglishName)
			continue
		}
		created, err := s.Create(a)
		if err != nil {
			continue
		}
		result.Created = append(result.Created, created.ID)
		result.Count++
	}
	return result, nil
}

func (s *Storage) GetSetting(key string) string {
	var v string
	s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	return v
}

func (s *Storage) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Storage) GetTheme() string {
	v := s.GetSetting("theme")
	if v == "" {
		return "dark"
	}
	return v
}

func (s *Storage) SetTheme(theme string) error {
	return s.SetSetting("theme", theme)
}

func (s *Storage) GetAudioFile() string {
	v := s.GetSetting("audio_file")
	if v == "" {
		return "pick.mp3"
	}
	return v
}

func (s *Storage) SetAudioFile(file string) error {
	return s.SetSetting("audio_file", file)
}

func (s *Storage) GetMuteAudio() bool {
	return s.GetSetting("mute_audio") == "1"
}

func (s *Storage) SetMuteAudio(mute bool) error {
	val := "0"
	if mute {
		val = "1"
	}
	return s.SetSetting("mute_audio", val)
}
