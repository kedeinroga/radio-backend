package postgres

import (
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/google/uuid"
)

// AdClickRepository implementa domain.AdClickRepository
type AdClickRepository struct {
	db *database.Connection
}

// NewAdClickRepository crea una nueva instancia del repositorio
func NewAdClickRepository(db *database.Connection) *AdClickRepository {
	return &AdClickRepository{db: db}
}

// Create crea un nuevo click
func (r *AdClickRepository) Create(click *domain.AdClick) error {
	query := `
		INSERT INTO ad_clicks (
			id, impression_id, advertisement_id, user_id,
			ip_address, user_agent, referrer, click_position,
			converted, conversion_value_cents,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.db.DB.Exec(query,
		click.ID,
		click.ImpressionID,
		click.AdvertisementID,
		click.UserID,
		click.IPAddress,
		click.UserAgent,
		click.Referrer,
		click.ClickPosition,
		click.Converted,
		click.ConversionValueCents,
		time.Now(),
	)

	return err
}

// GetByID obtiene un click por ID
func (r *AdClickRepository) GetByID(id uuid.UUID) (*domain.AdClick, error) {
	query := `
		SELECT
			id, impression_id, advertisement_id, user_id,
			ip_address, user_agent, referrer, click_position,
			converted, conversion_value_cents,
			created_at
		FROM ad_clicks
		WHERE id = $1
	`

	var click domain.AdClick
	err := r.db.DB.QueryRow(query, id).Scan(
		&click.ID,
		&click.ImpressionID,
		&click.AdvertisementID,
		&click.UserID,
		&click.IPAddress,
		&click.UserAgent,
		&click.Referrer,
		&click.ClickPosition,
		&click.Converted,
		&click.ConversionValueCents,
		&click.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("click not found: %w", err)
	}

	return &click, nil
}

// GetByAdvertisementID obtiene clicks de un anuncio
func (r *AdClickRepository) GetByAdvertisementID(adID uuid.UUID, limit int) ([]*domain.AdClick, error) {
	query := `
		SELECT
			id, impression_id, advertisement_id, user_id,
			ip_address, user_agent, referrer, click_position,
			converted, conversion_value_cents,
			created_at
		FROM ad_clicks
		WHERE advertisement_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	return r.queryClicks(query, adID, limit)
}

// GetByImpressionID obtiene el click asociado a una impresión
func (r *AdClickRepository) GetByImpressionID(impressionID uuid.UUID) (*domain.AdClick, error) {
	query := `
		SELECT
			id, impression_id, advertisement_id, user_id,
			ip_address, user_agent, referrer, click_position,
			converted, conversion_value_cents,
			created_at
		FROM ad_clicks
		WHERE impression_id = $1
	`

	var click domain.AdClick
	err := r.db.DB.QueryRow(query, impressionID).Scan(
		&click.ID,
		&click.ImpressionID,
		&click.AdvertisementID,
		&click.UserID,
		&click.IPAddress,
		&click.UserAgent,
		&click.Referrer,
		&click.ClickPosition,
		&click.Converted,
		&click.ConversionValueCents,
		&click.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("click not found: %w", err)
	}

	return &click, nil
}

// CountByAdvertisementID cuenta clicks de un anuncio
func (r *AdClickRepository) CountByAdvertisementID(adID uuid.UUID, since time.Time) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM ad_clicks
		WHERE advertisement_id = $1
			AND created_at >= $2
	`

	var count int64
	err := r.db.DB.QueryRow(query, adID, since).Scan(&count)
	return count, err
}

// CountByIPAddress cuenta clicks de una IP
func (r *AdClickRepository) CountByIPAddress(ipAddress string, since time.Time) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM ad_clicks
		WHERE ip_address = $1
			AND created_at >= $2
	`

	var count int64
	err := r.db.DB.QueryRow(query, ipAddress, since).Scan(&count)
	return count, err
}

// UpdateConversion actualiza la información de conversión de un click
func (r *AdClickRepository) UpdateConversion(clickID uuid.UUID, valueCents int) error {
	query := `
		UPDATE ad_clicks
		SET converted = true,
			conversion_value_cents = $2
		WHERE id = $1
	`

	_, err := r.db.DB.Exec(query, clickID, valueCents)
	return err
}

// GetRecentByIPAddress obtiene clicks recientes de una IP
func (r *AdClickRepository) GetRecentByIPAddress(ipAddress string, since time.Time) ([]*domain.AdClick, error) {
	query := `
		SELECT
			id, impression_id, advertisement_id, user_id,
			ip_address, user_agent, referrer, click_position,
			converted, conversion_value_cents,
			created_at
		FROM ad_clicks
		WHERE ip_address = $1
			AND created_at >= $2
		ORDER BY created_at DESC
	`

	return r.queryClicks(query, ipAddress, since)
}

// queryClicks es un helper para ejecutar queries y escanear múltiples clicks
func (r *AdClickRepository) queryClicks(query string, args ...interface{}) ([]*domain.AdClick, error) {
	rows, err := r.db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clicks := make([]*domain.AdClick, 0)
	for rows.Next() {
		var click domain.AdClick
		err := rows.Scan(
			&click.ID,
			&click.ImpressionID,
			&click.AdvertisementID,
			&click.UserID,
			&click.IPAddress,
			&click.UserAgent,
			&click.Referrer,
			&click.ClickPosition,
			&click.Converted,
			&click.ConversionValueCents,
			&click.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		clicks = append(clicks, &click)
	}

	return clicks, nil
}
