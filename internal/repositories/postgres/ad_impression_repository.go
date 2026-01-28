package postgres

import (
	"context"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/google/uuid"
)

// AdImpressionRepository implementa domain.AdImpressionRepository
type AdImpressionRepository struct {
	db *database.Connection
}

// NewAdImpressionRepository crea una nueva instancia del repositorio
func NewAdImpressionRepository(db *database.Connection) *AdImpressionRepository {
	return &AdImpressionRepository{db: db}
}

// Create crea una nueva impresión
func (r *AdImpressionRepository) Create(ctx context.Context, impression *domain.AdImpression) error {
	query := `
		INSERT INTO ad_impressions (
			id, advertisement_id, user_id, session_id, station_id,
			ip_address, user_agent, country_code, city, device_type,
			viewable, impression_duration_ms, impression_token,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		impression.ID,
		impression.AdvertisementID,
		impression.UserID,
		impression.SessionID,
		impression.StationID,
		impression.IPAddress,
		impression.UserAgent,
		impression.CountryCode,
		impression.City,
		impression.DeviceType,
		impression.Viewable,
		impression.ImpressionDurationMs,
		impression.ImpressionToken,
		time.Now(),
	)

	return err
}

// GetByID obtiene una impresión por ID
func (r *AdImpressionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AdImpression, error) {
	query := `
		SELECT
			id, advertisement_id, user_id, session_id, station_id,
			ip_address, user_agent, country_code, city, device_type,
			viewable, impression_duration_ms, impression_token,
			created_at
		FROM ad_impressions
		WHERE id = $1
	`

	var impression domain.AdImpression
	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&impression.ID,
		&impression.AdvertisementID,
		&impression.UserID,
		&impression.SessionID,
		&impression.StationID,
		&impression.IPAddress,
		&impression.UserAgent,
		&impression.CountryCode,
		&impression.City,
		&impression.DeviceType,
		&impression.Viewable,
		&impression.ImpressionDurationMs,
		&impression.ImpressionToken,
		&impression.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("impression not found: %w", err)
	}

	return &impression, nil
}

// GetByAdvertisementID obtiene impresiones de un anuncio
func (r *AdImpressionRepository) GetByAdvertisementID(ctx context.Context, adID uuid.UUID, limit int) ([]*domain.AdImpression, error) {
	query := `
		SELECT
			id, advertisement_id, user_id, session_id, station_id,
			ip_address, user_agent, country_code, city, device_type,
			viewable, impression_duration_ms, impression_token,
			created_at
		FROM ad_impressions
		WHERE advertisement_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	return r.queryImpressions(ctx, query, adID, limit)
}

// GetByUserID obtiene impresiones de un usuario desde una fecha
func (r *AdImpressionRepository) GetByUserID(ctx context.Context, userID uuid.UUID, since time.Time) ([]*domain.AdImpression, error) {
	query := `
		SELECT
			id, advertisement_id, user_id, session_id, station_id,
			ip_address, user_agent, country_code, city, device_type,
			viewable, impression_duration_ms, impression_token,
			created_at
		FROM ad_impressions
		WHERE user_id = $1
			AND created_at >= $2
		ORDER BY created_at DESC
	`

	return r.queryImpressions(ctx, query, userID, since)
}

// CountByAdvertisementID cuenta impresiones de un anuncio
func (r *AdImpressionRepository) CountByAdvertisementID(ctx context.Context, adID uuid.UUID, since time.Time) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM ad_impressions
		WHERE advertisement_id = $1
			AND created_at >= $2
	`

	var count int64
	err := r.db.DB.QueryRowContext(ctx, query, adID, since).Scan(&count)
	return count, err
}

// CountByIPAddress cuenta impresiones de una IP
func (r *AdImpressionRepository) CountByIPAddress(ctx context.Context, ipAddress string, since time.Time) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM ad_impressions
		WHERE ip_address = $1
			AND created_at >= $2
	`

	var count int64
	err := r.db.DB.QueryRowContext(ctx, query, ipAddress, since).Scan(&count)
	return count, err
}

// CountViewableImpressions cuenta impresiones viewable de un anuncio
func (r *AdImpressionRepository) CountViewableImpressions(ctx context.Context, adID uuid.UUID, since time.Time) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM ad_impressions
		WHERE advertisement_id = $1
			AND created_at >= $2
			AND viewable = true
			AND impression_duration_ms >= 1000
	`

	var count int64
	err := r.db.DB.QueryRowContext(ctx, query, adID, since).Scan(&count)
	return count, err
}

// GetRecentBySessionID obtiene impresiones recientes de una sesión
func (r *AdImpressionRepository) GetRecentBySessionID(ctx context.Context, sessionID string, since time.Time) ([]*domain.AdImpression, error) {
	query := `
		SELECT
			id, advertisement_id, user_id, session_id, station_id,
			ip_address, user_agent, country_code, city, device_type,
			viewable, impression_duration_ms, impression_token,
			created_at
		FROM ad_impressions
		WHERE session_id = $1
			AND created_at >= $2
		ORDER BY created_at DESC
	`

	return r.queryImpressions(ctx, query, sessionID, since)
}

// queryImpressions es un helper para ejecutar queries y escanear múltiples impresiones
func (r *AdImpressionRepository) queryImpressions(ctx context.Context, query string, args ...interface{}) ([]*domain.AdImpression, error) {
	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	impressions := make([]*domain.AdImpression, 0)
	for rows.Next() {
		var impression domain.AdImpression
		err := rows.Scan(
			&impression.ID,
			&impression.AdvertisementID,
			&impression.UserID,
			&impression.SessionID,
			&impression.StationID,
			&impression.IPAddress,
			&impression.UserAgent,
			&impression.CountryCode,
			&impression.City,
			&impression.DeviceType,
			&impression.Viewable,
			&impression.ImpressionDurationMs,
			&impression.ImpressionToken,
			&impression.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		impressions = append(impressions, &impression)
	}

	return impressions, nil
}
