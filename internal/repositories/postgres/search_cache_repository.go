package postgres

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/lib/pq"
)

// SearchCacheRepository implements domain.SearchCacheRepository
type SearchCacheRepository struct {
	db *database.Connection
}

// NewSearchCacheRepository creates a new search cache repository
func NewSearchCacheRepository(db *database.Connection) *SearchCacheRepository {
	return &SearchCacheRepository{db: db}
}

// Get retrieves a cached search result
func (r *SearchCacheRepository) Get(queryHash string) (*domain.SearchCacheEntry, error) {
	query := `
		SELECT query_hash, query_params, station_ids, result_count, expires_at, created_at
		FROM station_search_cache
		WHERE query_hash = $1 AND expires_at > NOW()
	`

	var entry domain.SearchCacheEntry
	var queryParamsJSON []byte
	var stationIDs pq.StringArray

	err := r.db.DB.QueryRow(query, queryHash).Scan(
		&entry.QueryHash,
		&queryParamsJSON,
		&stationIDs,
		&entry.ResultCount,
		&entry.ExpiresAt,
		&entry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get search cache: %w", err)
	}

	// Parse query params
	if err := json.Unmarshal(queryParamsJSON, &entry.QueryParams); err != nil {
		return nil, fmt.Errorf("failed to parse query params: %w", err)
	}

	entry.StationIDs = stationIDs

	return &entry, nil
}

// Save saves a search result to cache
func (r *SearchCacheRepository) Save(entry *domain.SearchCacheEntry) error {
	query := `
		INSERT INTO station_search_cache (query_hash, query_params, station_ids, result_count, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (query_hash) DO UPDATE SET
			query_params = EXCLUDED.query_params,
			station_ids = EXCLUDED.station_ids,
			result_count = EXCLUDED.result_count,
			expires_at = EXCLUDED.expires_at,
			created_at = EXCLUDED.created_at
	`

	queryParamsJSON, err := json.Marshal(entry.QueryParams)
	if err != nil {
		return fmt.Errorf("failed to marshal query params: %w", err)
	}

	_, err = r.db.DB.Exec(query,
		entry.QueryHash,
		queryParamsJSON,
		pq.Array(entry.StationIDs),
		entry.ResultCount,
		entry.ExpiresAt,
		entry.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save search cache: %w", err)
	}

	return nil
}

// Invalidate removes a search result from cache
func (r *SearchCacheRepository) Invalidate(queryHash string) error {
	query := `DELETE FROM station_search_cache WHERE query_hash = $1`
	_, err := r.db.DB.Exec(query, queryHash)
	if err != nil {
		return fmt.Errorf("failed to invalidate search cache: %w", err)
	}
	return nil
}

// CleanExpired removes expired search results
func (r *SearchCacheRepository) CleanExpired() error {
	query := `DELETE FROM station_search_cache WHERE expires_at < NOW()`
	_, err := r.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clean expired cache: %w", err)
	}
	return nil
}

// HashQuery creates a hash for query parameters
func HashQuery(params map[string]interface{}) string {
	data, _ := json.Marshal(params)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
