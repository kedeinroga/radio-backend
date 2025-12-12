package domain

import "time"

// StationCacheRepository defines the interface for local station cache operations
// Following Dependency Inversion Principle - domain defines interface, infrastructure implements
type StationCacheRepository interface {
	// Single station operations
	Get(id string) (*Station, error)
	Save(station *Station) error

	// Batch operations
	GetMany(ids []string) ([]Station, error)
	SaveMany(stations []Station) error

	// Query operations
	FindByName(name string, limit int) ([]Station, error)
	FindByCountry(country string, limit int) ([]Station, error)
	FindPopular(limit int, country string) ([]Station, error)

	// Cache management
	MarkForSync(id string) error
	GetStaleStations(maxAge time.Duration, limit int) ([]Station, error)
	DeleteInactive(olderThan time.Duration) error
}

// SearchCacheRepository defines the interface for search results caching
type SearchCacheRepository interface {
	Get(queryHash string) (*SearchCacheEntry, error)
	Save(entry *SearchCacheEntry) error
	Invalidate(queryHash string) error
	CleanExpired() error
}

// SearchCacheEntry represents a cached search result
type SearchCacheEntry struct {
	QueryHash   string
	QueryParams map[string]interface{}
	StationIDs  []string
	ResultCount int
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

// IsExpired checks if the cache entry has expired
func (e *SearchCacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}
