package domain

import (
	"testing"
)

func TestSEOMetadata(t *testing.T) {
	metadata := &SEOMetadata{
		Title:        "Test Radio Station",
		Description:  "A test radio station",
		CanonicalURL: "https://example.com/radio/test-station",
		Keywords:     []string{"rock", "music"},
		ImageURL:     "https://example.com/image.png",
		LastModified: "2025-12-23T10:00:00Z",
	}

	if metadata.Title == "" {
		t.Error("Title should not be empty")
	}

	if len(metadata.Keywords) != 2 {
		t.Errorf("Expected 2 keywords, got %d", len(metadata.Keywords))
	}
}

func TestPopularTag(t *testing.T) {
	tag := PopularTag{
		Name:         "Rock",
		Slug:         "rock",
		StationCount: 100,
		ActiveCount:  80,
	}

	if tag.ActiveCount > tag.StationCount {
		t.Error("ActiveCount should not be greater than StationCount")
	}

	if tag.Slug != "rock" {
		t.Errorf("Expected slug 'rock', got '%s'", tag.Slug)
	}
}

func TestPopularCountry(t *testing.T) {
	country := PopularCountry{
		Code:         "US",
		Name:         "United States",
		Slug:         "united-states",
		StationCount: 500,
	}

	if len(country.Code) != 2 {
		t.Errorf("Country code should be 2 characters, got %d", len(country.Code))
	}

	if country.StationCount < 0 {
		t.Error("StationCount should not be negative")
	}
}

func TestSitemapData(t *testing.T) {
	data := &SitemapData{
		PopularTags: []PopularTag{
			{Name: "Rock", Slug: "rock", StationCount: 100, ActiveCount: 80},
			{Name: "Pop", Slug: "pop", StationCount: 90, ActiveCount: 70},
		},
		PopularCountries: []PopularCountry{
			{Code: "US", Name: "United States", Slug: "united-states", StationCount: 500},
		},
		TotalStations: 1000,
		LastUpdated:   "2025-12-23T10:00:00Z",
	}

	if len(data.PopularTags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(data.PopularTags))
	}

	if len(data.PopularCountries) != 1 {
		t.Errorf("Expected 1 country, got %d", len(data.PopularCountries))
	}

	if data.TotalStations <= 0 {
		t.Error("TotalStations should be greater than 0")
	}
}
