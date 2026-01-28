package domain

import (
	"context"
	"time"
)

// StationCacheRepository defines the interface for local station cache operations
// Following Dependency Inversion Principle - domain defines interface, infrastructure implements
type StationCacheRepository interface {
	// Single station operations
	Get(ctx context.Context, id string) (*Station, error)
	Save(ctx context.Context, station *Station) error

	// Batch operations
	GetMany(ctx context.Context, ids []string) ([]Station, error)
	SaveMany(ctx context.Context, stations []Station) error

	// Query operations
	FindByName(ctx context.Context, name string, limit int) ([]Station, error)
	FindByCountry(ctx context.Context, country string, limit int) ([]Station, error)
	FindPopular(ctx context.Context, limit int, country string) ([]Station, error)

	// Cache management
	MarkForSync(ctx context.Context, id string) error
	GetStaleStations(ctx context.Context, maxAge time.Duration, limit int) ([]Station, error)
	DeleteInactive(ctx context.Context, olderThan time.Duration) error
}

// SearchCacheRepository defines the interface for search results caching
type SearchCacheRepository interface {
	Get(ctx context.Context, queryHash string) (*SearchCacheEntry, error)
	Save(ctx context.Context, entry *SearchCacheEntry) error
	Invalidate(ctx context.Context, queryHash string) error
	CleanExpired(ctx context.Context) error
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
