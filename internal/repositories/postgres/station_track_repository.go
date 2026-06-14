package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"
)

// StationTrackRepository implementa domain.StationTrackRepository sobre PostgreSQL.
type StationTrackRepository struct {
	db *database.Connection
}

// NewStationTrackRepository crea una nueva instancia del repositorio.
func NewStationTrackRepository(db *database.Connection) *StationTrackRepository {
	return &StationTrackRepository{db: db}
}

// Insert guarda una nueva pista detectada.
func (r *StationTrackRepository) Insert(ctx context.Context, track *domain.StationTrack) error {
	query := `
		INSERT INTO station_tracks (
			station_id, raw_title, artist, title, source, played_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.db.DB.QueryRowContext(
		ctx, query,
		track.StationID,
		track.RawTitle,
		nullString(track.Artist),
		nullString(track.Title),
		string(track.Source),
		track.PlayedAt,
	).Scan(&track.ID, &track.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert station track: %w", err)
	}

	return nil
}

// GetLatest obtiene la última pista detectada de una estación ("now playing").
func (r *StationTrackRepository) GetLatest(ctx context.Context, stationID string) (*domain.StationTrack, error) {
	query := `
		SELECT id, station_id, raw_title, artist, title, source, played_at, created_at
		FROM station_tracks
		WHERE station_id = $1
		ORDER BY played_at DESC
		LIMIT 1
	`

	track, err := scanStationTrack(r.db.DB.QueryRowContext(ctx, query, stationID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest station track: %w", err)
	}

	return track, nil
}

// GetRecent obtiene las últimas pistas de una estación.
func (r *StationTrackRepository) GetRecent(ctx context.Context, stationID string, limit int) ([]*domain.StationTrack, error) {
	query := `
		SELECT id, station_id, raw_title, artist, title, source, played_at, created_at
		FROM station_tracks
		WHERE station_id = $1
		ORDER BY played_at DESC
		LIMIT $2
	`

	rows, err := r.db.DB.QueryContext(ctx, query, stationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent station tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*domain.StationTrack
	for rows.Next() {
		track, err := scanStationTrack(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan station track: %w", err)
		}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating station tracks: %w", err)
	}

	return tracks, nil
}

// DeleteOlderThan elimina pistas más antiguas que la duración indicada.
func (r *StationTrackRepository) DeleteOlderThan(ctx context.Context, age time.Duration) (int, error) {
	query := `DELETE FROM station_tracks WHERE played_at < $1`

	cutoff := time.Now().Add(-age)
	result, err := r.db.DB.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old station tracks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// rowScanner abstrae *sql.Row y *sql.Rows para reutilizar el scan.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanStationTrack(row rowScanner) (*domain.StationTrack, error) {
	track := &domain.StationTrack{}
	var artist, title sql.NullString
	var source string

	err := row.Scan(
		&track.ID,
		&track.StationID,
		&track.RawTitle,
		&artist,
		&title,
		&source,
		&track.PlayedAt,
		&track.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	track.Artist = artist.String
	track.Title = title.String
	track.Source = domain.TrackSource(source)

	return track, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
