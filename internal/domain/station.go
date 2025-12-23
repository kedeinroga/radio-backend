package domain

import "time"

type Station struct {
	ID                string
	Name              string
	StreamURL         string
	StreamURLResolved string
	ImageURL          string
	Tags              []string
	Country           string
	Votes             int
	IsPremiumOnly     bool

	// Cache metadata
	Source       string
	LastSyncedAt *time.Time
	SyncCount    int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
