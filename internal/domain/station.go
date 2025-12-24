package domain

import "time"

// Station representa una estación de radio
type Station struct {
	ID                string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name              string     `json:"name" example:"Rock FM 100.1"`
	StreamURL         string     `json:"stream_url" example:"https://stream.rockfm.com/live"`
	StreamURLResolved string     `json:"stream_url_resolved,omitempty" example:"https://cdn.rockfm.com/stream.mp3"`
	ImageURL          string     `json:"image_url,omitempty" example:"https://cdn.rockfm.com/logo.png"`
	Tags              []string   `json:"tags" example:"rock,classic rock,80s"`
	Country           string     `json:"country" example:"United States"`
	Votes             int        `json:"votes" example:"1500"`
	IsPremiumOnly     bool       `json:"is_premium_only" example:"false"`

	// SEO fields
	Slug        string       `json:"slug" example:"rock-fm-100-1"`
	SEOMetadata *SEOMetadata `json:"seo_metadata,omitempty"`

	// Cache metadata
	Source       string     `json:"source,omitempty" example:"radio_browser"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	SyncCount    int        `json:"sync_count,omitempty" example:"5"`
	IsActive     bool       `json:"is_active" example:"true"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (s *Station) IsAccessibleBy(userType UserType) bool {
	if s.IsPremiumOnly {
		return userType == UserTypePremium
	}
	return true
}

// NeedsSync checks if station data should be refreshed from external source
func (s *Station) NeedsSync(maxAge time.Duration) bool {
	if s.LastSyncedAt == nil {
		return true
	}
	return time.Since(*s.LastSyncedAt) > maxAge
}

type StationRepository interface {
	FindByID(id string) (*Station, error)
	FindPopular(limit int, country string) ([]Station, error)
	Search(query string, limit int) ([]Station, error)
}
