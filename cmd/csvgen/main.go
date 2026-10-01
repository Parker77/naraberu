package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AnimeData struct {
	EnglishName  string
	JapaneseName string
	Synonyms     []string
	Synopsis     string
	Thumbnail    string
	Tags         []string
}

type WikiSummary struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumbnail   struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	Extract string `json:"extract"`
}

type WikiSearchResult struct {
	Query struct {
		Search []struct {
			Title string `json:"title"`
		} `json:"search"`
	} `json:"query"`
}

type JikanAnime struct {
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english"`
	TitleJapanese string   `json:"title_japanese"`
	TitleSynonyms []string `json:"title_synonyms"`
	Titles        []struct {
		Type  string `json:"type"`
		Title string `json:"title"`
	} `json:"titles"`
	Synopsis string `json:"synopsis"`
	Images   struct {
		JPG struct {
			ImageURL string `json:"image_url"`
		} `json:"jpg"`
	} `json:"images"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

type JikanResult struct {
	Data []JikanAnime `json:"data"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: csvgen <anime-list.txt> [output.csv]\n")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputPath := "anime-import.csv"
	if len(os.Args) >= 3 {
		outputPath = os.Args[2]
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var results []string
	var skipped []string

	client := &http.Client{Timeout: 15 * time.Second}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fmt.Fprintf(os.Stderr, "[%d/%d] %s... ", i+1, len(lines), line)

		ad, err := fetchAnimeData(client, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAILED (%v)\n", err)
			skipped = append(skipped, line)
			continue
		}

		csvLine := buildCSVLine(line, ad)
		results = append(results, csvLine)
		fmt.Fprintf(os.Stderr, "OK\n")

		time.Sleep(200 * time.Millisecond)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", outputPath, err)
		os.Exit(1)
	}
	defer f.Close()

	f.WriteString("englishName,japaneseName,synonyms,status,linkedIds,thumbnailPath,startDate,endDate,score,review,synopsis,tags,favorite,owned\n")
	for _, r := range results {
		f.WriteString(r + "\n")
	}

	fmt.Fprintf(os.Stderr, "\nDone! %d series written to %s\n", len(results), outputPath)
	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "Skipped %d series: %v\n", len(skipped), skipped)
	}
}

func sanitizeName(name string) string {
	if idx := strings.Index(name, "["); idx != -1 {
		name = strings.TrimSpace(name[:idx])
	}
	if idx := strings.Index(name, "("); idx != -1 {
		name = strings.TrimSpace(name[:idx])
	}
	name = strings.ReplaceAll(name, ":", "")
	name = strings.ReplaceAll(name, "!", "")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	name = strings.ReplaceAll(name, "\u2014", "")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "\u2019", "")
	name = strings.ReplaceAll(name, "\u00d7", "x")
	return strings.TrimSpace(name)
}

func fetchAnimeData(client *http.Client, name string) (*AnimeData, error) {
	ad, err := fetchFromJikan(client, name)
	if err == nil {
		return ad, nil
	}

	ad, err = fetchFromWikipedia(client, name)
	if err == nil {
		return ad, nil
	}

	return nil, fmt.Errorf("not found")
}

func fetchFromJikan(client *http.Client, name string) (*AnimeData, error) {
	apiQuery := url.QueryEscape(name)
	req, err := http.NewRequest("GET", "https://api.jikan.moe/v4/anime?q="+apiQuery+"&limit=5", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Naraberu-csvgen/1.0 (anime tracker)")

	var resp *http.Response
	for retries := 0; retries < 3; retries++ {
		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 429 {
			break
		}
		resp.Body.Close()
		time.Sleep(1 * time.Second)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result JikanResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no results")
	}

	d := matchJikanResult(result.Data, name)
	if d == nil {
		return nil, fmt.Errorf("no match")
	}

	synopsis := cleanSynopsis(d.Synopsis)

	var tags []string
	for _, g := range d.Genres {
		tags = append(tags, g.Name)
	}

	var synonyms []string
	if d.TitleEnglish != "" && d.TitleEnglish != d.Title {
		synonyms = append(synonyms, d.TitleEnglish)
	}
	for _, s := range d.TitleSynonyms {
		if s != d.Title && s != d.TitleEnglish {
			synonyms = append(synonyms, s)
		}
	}

	thumbnail := ""
	if d.Images.JPG.ImageURL != "" {
		thumbnail = d.Images.JPG.ImageURL
	}

	return &AnimeData{
		EnglishName:  name,
		JapaneseName: d.TitleJapanese,
		Synonyms:     synonyms,
		Synopsis:     synopsis,
		Thumbnail:    thumbnail,
		Tags:         tags,
	}, nil
}

func matchJikanResult(data []JikanAnime, name string) *JikanAnime {
	lower := strings.ToLower(name)

	// Exact match on title
	for i := range data {
		if strings.ToLower(data[i].Title) == lower {
			return &data[i]
		}
	}
	// Exact match on English title
	for i := range data {
		if strings.ToLower(data[i].TitleEnglish) == lower {
			return &data[i]
		}
	}
	// Exact match on any title variant
	for i := range data {
		for _, t := range data[i].Titles {
			if strings.ToLower(t.Title) == lower {
				return &data[i]
			}
		}
	}
	// Exact match on synonyms
	for i := range data {
		for _, s := range data[i].TitleSynonyms {
			if strings.ToLower(s) == lower {
				return &data[i]
			}
		}
	}
	// Substring match on title
	for i := range data {
		t := strings.ToLower(data[i].Title)
		if strings.Contains(t, lower) || strings.Contains(lower, t) {
			return &data[i]
		}
	}
	// Substring match on any title variant
	for i := range data {
		for _, t := range data[i].Titles {
			tl := strings.ToLower(t.Title)
			if strings.Contains(tl, lower) || strings.Contains(lower, tl) {
				return &data[i]
			}
		}
	}
	return &data[0]
}

func cleanSynopsis(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "[Written by MAL Rewrite]"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, "(Source:"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	if len(s) > 500 {
		s = s[:500]
		lastSpace := strings.LastIndex(s, " ")
		if lastSpace > 400 {
			s = s[:lastSpace]
		}
		s += "\u2026"
	}
	return s
}

func fetchFromWikipedia(client *http.Client, name string) (*AnimeData, error) {
	sanitized := sanitizeName(name)
	apiName := strings.ReplaceAll(sanitized, " ", "_")

	summary, err := tryWikiSummaryValidated(client, apiName)
	if err != nil {
		origApiName := strings.ReplaceAll(name, " ", "_")
		origApiName = strings.ReplaceAll(origApiName, ":", "")
		origApiName = strings.ReplaceAll(origApiName, "!", "")
		origApiName = strings.ReplaceAll(origApiName, "?", "")
		if origApiName != apiName {
			summary, err = tryWikiSummaryValidated(client, origApiName)
		}
	}

	if err != nil {
		suffixes := []string{"_(anime)", "_(light_novel)", "_(novel_series)", "_(manga)", "_(film)"}
		for _, suffix := range suffixes {
			summary, err = tryWikiSummaryValidated(client, apiName+suffix)
			if err == nil {
				break
			}
		}
	}

	if err != nil {
		searchTitle, searchErr := searchWikipedia(client, name+" anime")
		if searchErr == nil {
			summary, err = tryWikiSummaryValidated(client, strings.ReplaceAll(searchTitle, " ", "_"))
		}
	}

	if err != nil {
		return nil, err
	}

	synopsis := cleanSynopsis(summary.Extract)

	thumbnail := ""
	if summary.Thumbnail.Source != "" {
		thumbnail = summary.Thumbnail.Source
	}

	return &AnimeData{
		EnglishName:  name,
		JapaneseName: "",
		Synonyms:     nil,
		Synopsis:     synopsis,
		Thumbnail:    thumbnail,
		Tags:         nil,
	}, nil
}

func isAnimeDescription(desc string) bool {
	lower := strings.ToLower(desc)
	keywords := []string{"anime", "manga", "novel", "series", "film", "television", "light novel", "ova", "special"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func tryWikiSummaryValidated(client *http.Client, apiName string) (*WikiSummary, error) {
	summary, err := tryWikiSummary(client, apiName)
	if err != nil {
		return nil, err
	}
	if !isAnimeDescription(summary.Description) {
		return nil, fmt.Errorf("not an anime article: %s", summary.Description)
	}
	return summary, nil
}

func searchWikipedia(client *http.Client, query string) (string, error) {
	apiQuery := url.QueryEscape(query)
	req, err := http.NewRequest("GET", "https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch="+apiQuery+"&format=json&srlimit=5", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Naraberu-csvgen/1.0 (anime tracker)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result WikiSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Query.Search) == 0 {
		return "", fmt.Errorf("no results")
	}

	lowerQuery := strings.ToLower(query)
	for _, r := range result.Query.Search {
		title := strings.ToLower(r.Title)
		if strings.Contains(title, "anime") || strings.Contains(title, "manga") || strings.Contains(title, "novel series") {
			return r.Title, nil
		}
		if strings.Contains(lowerQuery, strings.ToLower(r.Title)) || strings.Contains(strings.ToLower(r.Title), lowerQuery) {
			return r.Title, nil
		}
	}

	return result.Query.Search[0].Title, nil
}

func tryWikiSummary(client *http.Client, apiName string) (*WikiSummary, error) {
	req, err := http.NewRequest("GET", "https://en.wikipedia.org/api/rest_v1/page/summary/"+apiName, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Naraberu-csvgen/1.0 (anime tracker)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var summary WikiSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

func buildCSVLine(originalName string, ad *AnimeData) string {
	// Columns: englishName,japaneseName,synonyms,status,linkedIds,thumbnailPath,
	// startDate,endDate,score,review,synopsis,tags,favorite,owned
	return fmt.Sprintf("%s,%s,%s,watched,,%s,,,,,%s,%s,false,false",
		csvEscape(originalName),
		csvEscape(ad.JapaneseName),
		csvEscape(strings.Join(ad.Synonyms, ";")),
		csvEscape(ad.Thumbnail),
		csvEscape(ad.Synopsis),
		csvEscape(strings.Join(ad.Tags, ";")),
	)
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
