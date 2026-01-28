package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/lib/pq"
)

// StationCacheRepository implements domain.StationCacheRepository
type StationCacheRepository struct {
	db *database.Connection
}

// NewStationCacheRepository creates a new station cache repository
func NewStationCacheRepository(db *database.Connection) *StationCacheRepository {
	return &StationCacheRepository{db: db}
}

// Get retrieves a single station from cache by ID
func (r *StationCacheRepository) Get(ctx context.Context, id string) (*domain.Station, error) {
	query := `
		SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
		       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
		       created_at, updated_at
		FROM stations
		WHERE id = $1 AND is_active = true
	`

	var station domain.Station
	var tags pq.StringArray
	var lastSyncedAt sql.NullTime

	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&station.ID,
		&station.Name,
		&station.StreamURL,
		&station.StreamURLResolved,
		&station.ImageURL,
		&tags,
		&station.Country,
		&station.Votes,
		&station.IsPremiumOnly,
		&station.Source,
		&lastSyncedAt,
		&station.SyncCount,
		&station.IsActive,
		&station.CreatedAt,
		&station.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	station.Tags = tags
	if lastSyncedAt.Valid {
		station.LastSyncedAt = &lastSyncedAt.Time
	}

	return &station, nil
}

// FindByID busca una estación por su ID (implementa domain.StationRepository)
func (r *StationCacheRepository) FindByID(ctx context.Context, id string) (*domain.Station, error) {
	return r.Get(ctx, id)
}

// Save upserts a station to the cache
func (r *StationCacheRepository) Save(ctx context.Context, station *domain.Station) error {
	query := `
		INSERT INTO stations (
			id, name, stream_url, stream_url_resolved, image_url, tags, country,
			votes, is_premium_only, source, last_synced_at, sync_count, is_active,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			stream_url = EXCLUDED.stream_url,
			stream_url_resolved = EXCLUDED.stream_url_resolved,
			image_url = EXCLUDED.image_url,
			tags = EXCLUDED.tags,
			country = EXCLUDED.country,
			votes = EXCLUDED.votes,
			is_premium_only = EXCLUDED.is_premium_only,
			last_synced_at = EXCLUDED.last_synced_at,
			sync_count = stations.sync_count + 1,
			is_active = true,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	if station.CreatedAt.IsZero() {
		station.CreatedAt = now
	}
	station.UpdatedAt = now
	station.LastSyncedAt = &now

	_, err := r.db.DB.ExecContext(ctx, query,
		station.ID,
		station.Name,
		station.StreamURL,
		station.StreamURLResolved,
		station.ImageURL,
		pq.Array(station.Tags),
		station.Country,
		station.Votes,
		station.IsPremiumOnly,
		station.Source,
		station.LastSyncedAt,
		station.SyncCount,
		station.IsActive,
		station.CreatedAt,
		station.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save station: %w", err)
	}

	return nil
}

// GetMany retrieves multiple stations by IDs
func (r *StationCacheRepository) GetMany(ctx context.Context, ids []string) ([]domain.Station, error) {
	if len(ids) == 0 {
		return []domain.Station{}, nil
	}

	query := `
		SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
		       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
		       created_at, updated_at
		FROM stations
		WHERE id = ANY($1) AND is_active = true
		ORDER BY votes DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to get stations: %w", err)
	}
	defer rows.Close()

	return r.scanStations(rows)
}

// SaveMany saves multiple stations in a batch
func (r *StationCacheRepository) SaveMany(ctx context.Context, stations []domain.Station) error {
	for _, station := range stations {
		if err := r.Save(ctx, &station); err != nil {
			return err
		}
	}
	return nil
}

// FindByName searches stations by name using fuzzy matching
func (r *StationCacheRepository) FindByName(ctx context.Context, name string, limit int) ([]domain.Station, error) {
	query := `
		SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
		       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
		       created_at, updated_at
		FROM stations
		WHERE is_active = true
		  AND name ILIKE $1
		ORDER BY votes DESC
		LIMIT $2
	`

	rows, err := r.db.DB.QueryContext(ctx, query, "%"+name+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search stations: %w", err)
	}
	defer rows.Close()

	return r.scanStations(rows)
}

// FindByCountry finds stations by country
func (r *StationCacheRepository) FindByCountry(ctx context.Context, country string, limit int) ([]domain.Station, error) {
	query := `
		SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
		       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
		       created_at, updated_at
		FROM stations
		WHERE is_active = true
		  AND country = $1
		ORDER BY votes DESC
		LIMIT $2
	`

	rows, err := r.db.DB.QueryContext(ctx, query, country, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find stations by country: %w", err)
	}
	defer rows.Close()

	return r.scanStations(rows)
}

// FindPopular finds popular stations from cache
func (r *StationCacheRepository) FindPopular(ctx context.Context, limit int, country string) ([]domain.Station, error) {
	var query string
	var args []interface{}

	if country != "" {
		// With country filter
		query = `
			SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
			       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
			       created_at, updated_at
			FROM stations
			WHERE is_active = true
			  AND country = $1
			ORDER BY votes DESC
			LIMIT $2
		`
		args = []interface{}{country, limit}
	} else {
		// Without country filter
		query = `
			SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
			       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
			       created_at, updated_at
			FROM stations
			WHERE is_active = true
			ORDER BY votes DESC
			LIMIT $1
		`
		args = []interface{}{limit}
	}

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find popular stations: %w", err)
	}
	defer rows.Close()

	return r.scanStations(rows)
}

// Search busca estaciones por nombre o tags (implementa domain.StationRepository)
func (r *StationCacheRepository) Search(ctx context.Context, query string, limit int) ([]domain.Station, error) {
	sqlQuery := `
		SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
		       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
		       created_at, updated_at
		FROM stations
		WHERE is_active = true
		  AND (
			  LOWER(name) LIKE LOWER($1)
			  OR EXISTS (
				  SELECT 1 FROM unnest(tags) AS tag
				  WHERE LOWER(tag) LIKE LOWER($1)
			  )
		  )
		ORDER BY votes DESC
		LIMIT $2
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.DB.QueryContext(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search stations: %w", err)
	}
	defer rows.Close()

	return r.scanStations(rows)
}

// MarkForSync marks a station for synchronization
func (r *StationCacheRepository) MarkForSync(ctx context.Context, id string) error {
	query := `UPDATE stations SET last_synced_at = NULL WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

// GetStaleStations retrieves stations that need synchronization
func (r *StationCacheRepository) GetStaleStations(ctx context.Context, maxAge time.Duration, limit int) ([]domain.Station, error) {
	query := `
		SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country,
		       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
		       created_at, updated_at
		FROM stations
		WHERE is_active = true
		  AND (last_synced_at IS NULL OR last_synced_at < $1)
		ORDER BY last_synced_at ASC NULLS FIRST
		LIMIT $2
	`

	cutoff := time.Now().Add(-maxAge)
	rows, err := r.db.DB.QueryContext(ctx, query, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale stations: %w", err)
	}
	defer rows.Close()

	return r.scanStations(rows)
}

// DeleteInactive deletes inactive stations older than the threshold
func (r *StationCacheRepository) DeleteInactive(ctx context.Context, olderThan time.Duration) error {
	query := `
		DELETE FROM stations
		WHERE is_active = false
		  AND updated_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	_, err := r.db.DB.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete inactive stations: %w", err)
	}

	return nil
}

// scanStations is a helper to scan multiple station rows
func (r *StationCacheRepository) scanStations(rows *sql.Rows) ([]domain.Station, error) {
	stations := make([]domain.Station, 0)

	for rows.Next() {
		var station domain.Station
		var tags pq.StringArray
		var lastSyncedAt sql.NullTime

		err := rows.Scan(
			&station.ID,
			&station.Name,
			&station.StreamURL,
			&station.StreamURLResolved,
			&station.ImageURL,
			&tags,
			&station.Country,
			&station.Votes,
			&station.IsPremiumOnly,
			&station.Source,
			&lastSyncedAt,
			&station.SyncCount,
			&station.IsActive,
			&station.CreatedAt,
			&station.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan station: %w", err)
		}

		station.Tags = tags
		if lastSyncedAt.Valid {
			station.LastSyncedAt = &lastSyncedAt.Time
		}

		stations = append(stations, station)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stations: %w", err)
	}

	return stations, nil
}
