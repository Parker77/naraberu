package main

import "time"

type AnimeSeries struct {
	ID            string    `json:"id"`
	EnglishName   string    `json:"englishName"`
	JapaneseName  string    `json:"japaneseName"`
	Synonyms      []string  `json:"synonyms"`
	Status        string    `json:"status"`
	LinkedIDs     []string  `json:"linkedIds"`
	ThumbnailPath string    `json:"thumbnailPath"`
	StartDate     string    `json:"startDate"`
	EndDate       string    `json:"endDate"`
	Score         float64   `json:"score"`
	Review        string    `json:"review"`
	Synopsis      string    `json:"synopsis"`
	Tags          []string  `json:"tags"`
	Favorite      bool      `json:"favorite"`
	Owned         bool      `json:"owned"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SearchFilter struct {
	Query         string   `json:"query"`
	Status        string   `json:"status"`
	Tags          []string `json:"tags"`
	FavoritesOnly bool     `json:"favoritesOnly"`
	OwnedOnly     bool     `json:"ownedOnly"`
	LinkedOnly    bool     `json:"linkedOnly"`
	SortBy        string   `json:"sortBy"`
	SortDesc      bool     `json:"sortDesc"`
}

type ImportResult struct {
	Count    int      `json:"count"`
	Skipped  []string `json:"skipped"`
	Created  []string `json:"created"`
	Warnings []string `json:"warnings"`
}
