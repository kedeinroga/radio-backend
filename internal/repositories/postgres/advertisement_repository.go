package postgres

import (
	"context"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AdvertisementRepository implementa domain.AdvertisementRepository
type AdvertisementRepository struct {
	db *database.Connection
}

// NewAdvertisementRepository crea una nueva instancia del repositorio
func NewAdvertisementRepository(db *database.Connection) *AdvertisementRepository {
	return &AdvertisementRepository{db: db}
}

// Create crea un nuevo anuncio
func (r *AdvertisementRepository) Create(ctx context.Context, ad *domain.Advertisement) error {
	query := `
		INSERT INTO advertisements (
			id, campaign_id, title, description, advertiser_name,
			ad_format, ad_type, media_url, click_url, companion_banner_url,
			width, height, duration_seconds,
			target_countries, target_genres, target_languages, target_devices,
			start_date, end_date, daily_budget_cents, total_budget_cents,
			cpm_rate_cents, cpc_rate_cents, status, priority,
			max_impressions_per_user, max_impressions_per_day,
			impressions_count, clicks_count, spend_cents,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32
		)
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		ad.ID,
		ad.CampaignID,
		ad.Title,
		ad.Description,
		ad.AdvertiserName,
		ad.AdFormat,
		ad.AdType,
		ad.MediaURL,
		ad.ClickURL,
		ad.CompanionBannerURL,
		ad.Width,
		ad.Height,
		ad.DurationSeconds,
		pq.Array(ad.TargetCountries),
		pq.Array(ad.TargetGenres),
		pq.Array(ad.TargetLanguages),
		pq.Array(ad.TargetDevices),
		ad.StartDate,
		ad.EndDate,
		ad.DailyBudgetCents,
		ad.TotalBudgetCents,
		ad.CPMRateCents,
		ad.CPCRateCents,
		ad.Status,
		ad.Priority,
		ad.MaxImpressionsPerUser,
		ad.MaxImpressionsPerDay,
		ad.ImpressionsCount,
		ad.ClicksCount,
		ad.SpendCents,
		time.Now(),
		time.Now(),
	)

	return err
}

// GetByID obtiene un anuncio por ID
func (r *AdvertisementRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Advertisement, error) {
	query := `
		SELECT
			id, campaign_id, title, description, advertiser_name,
			ad_format, ad_type, media_url, click_url, companion_banner_url,
			width, height, duration_seconds,
			target_countries, target_genres, target_languages, target_devices,
			start_date, end_date, daily_budget_cents, total_budget_cents,
			cpm_rate_cents, cpc_rate_cents, status, priority,
			max_impressions_per_user, max_impressions_per_day,
			impressions_count, clicks_count, spend_cents,
			created_at, updated_at
		FROM advertisements
		WHERE id = $1
	`

	var ad domain.Advertisement
	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&ad.ID,
		&ad.CampaignID,
		&ad.Title,
		&ad.Description,
		&ad.AdvertiserName,
		&ad.AdFormat,
		&ad.AdType,
		&ad.MediaURL,
		&ad.ClickURL,
		&ad.CompanionBannerURL,
		&ad.Width,
		&ad.Height,
		&ad.DurationSeconds,
		pq.Array(&ad.TargetCountries),
		pq.Array(&ad.TargetGenres),
		pq.Array(&ad.TargetLanguages),
		pq.Array(&ad.TargetDevices),
		&ad.StartDate,
		&ad.EndDate,
		&ad.DailyBudgetCents,
		&ad.TotalBudgetCents,
		&ad.CPMRateCents,
		&ad.CPCRateCents,
		&ad.Status,
		&ad.Priority,
		&ad.MaxImpressionsPerUser,
		&ad.MaxImpressionsPerDay,
		&ad.ImpressionsCount,
		&ad.ClicksCount,
		&ad.SpendCents,
		&ad.CreatedAt,
		&ad.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("advertisement not found: %w", err)
	}

	return &ad, nil
}

// GetByCampaignID obtiene todos los anuncios de una campaña
func (r *AdvertisementRepository) GetByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Advertisement, error) {
	query := `
		SELECT
			id, campaign_id, title, description, advertiser_name,
			ad_format, ad_type, media_url, click_url, companion_banner_url,
			width, height, duration_seconds,
			target_countries, target_genres, target_languages, target_devices,
			start_date, end_date, daily_budget_cents, total_budget_cents,
			cpm_rate_cents, cpc_rate_cents, status, priority,
			max_impressions_per_user, max_impressions_per_day,
			impressions_count, clicks_count, spend_cents,
			created_at, updated_at
		FROM advertisements
		WHERE campaign_id = $1
		ORDER BY priority DESC, created_at DESC
	`

	return r.queryAds(ctx, query, campaignID)
}

// Update actualiza un anuncio
func (r *AdvertisementRepository) Update(ctx context.Context, ad *domain.Advertisement) error {
	query := `
		UPDATE advertisements
		SET title = $2,
			description = $3,
			advertiser_name = $4,
			ad_format = $5,
			ad_type = $6,
			media_url = $7,
			click_url = $8,
			companion_banner_url = $9,
			width = $10,
			height = $11,
			duration_seconds = $12,
			target_countries = $13,
			target_genres = $14,
			target_languages = $15,
			target_devices = $16,
			start_date = $17,
			end_date = $18,
			daily_budget_cents = $19,
			total_budget_cents = $20,
			cpm_rate_cents = $21,
			cpc_rate_cents = $22,
			status = $23,
			priority = $24,
			max_impressions_per_user = $25,
			max_impressions_per_day = $26,
			updated_at = $27
		WHERE id = $1
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		ad.ID,
		ad.Title,
		ad.Description,
		ad.AdvertiserName,
		ad.AdFormat,
		ad.AdType,
		ad.MediaURL,
		ad.ClickURL,
		ad.CompanionBannerURL,
		ad.Width,
		ad.Height,
		ad.DurationSeconds,
		pq.Array(ad.TargetCountries),
		pq.Array(ad.TargetGenres),
		pq.Array(ad.TargetLanguages),
		pq.Array(ad.TargetDevices),
		ad.StartDate,
		ad.EndDate,
		ad.DailyBudgetCents,
		ad.TotalBudgetCents,
		ad.CPMRateCents,
		ad.CPCRateCents,
		ad.Status,
		ad.Priority,
		ad.MaxImpressionsPerUser,
		ad.MaxImpressionsPerDay,
		time.Now(),
	)

	return err
}

// Delete elimina un anuncio
func (r *AdvertisementRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM advertisements WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

// GetActiveAds obtiene todos los anuncios activos
func (r *AdvertisementRepository) GetActiveAds(ctx context.Context) ([]*domain.Advertisement, error) {
	query := `
		SELECT
			id, campaign_id, title, description, advertiser_name,
			ad_format, ad_type, media_url, click_url, companion_banner_url,
			width, height, duration_seconds,
			target_countries, target_genres, target_languages, target_devices,
			start_date, end_date, daily_budget_cents, total_budget_cents,
			cpm_rate_cents, cpc_rate_cents, status, priority,
			max_impressions_per_user, max_impressions_per_day,
			impressions_count, clicks_count, spend_cents,
			created_at, updated_at
		FROM advertisements
		WHERE status = 'active'
			AND start_date <= NOW()
			AND end_date >= NOW()
			AND spend_cents < total_budget_cents
		ORDER BY priority DESC
	`

	return r.queryAds(ctx, query)
}

// GetEligibleAds obtiene anuncios elegibles basados en targeting
func (r *AdvertisementRepository) GetEligibleAds(ctx context.Context, country, genre, language, device string) ([]*domain.Advertisement, error) {
	query := `
		SELECT
			id, campaign_id, title, description, advertiser_name,
			ad_format, ad_type, media_url, click_url, companion_banner_url,
			width, height, duration_seconds,
			target_countries, target_genres, target_languages, target_devices,
			start_date, end_date, daily_budget_cents, total_budget_cents,
			cpm_rate_cents, cpc_rate_cents, status, priority,
			max_impressions_per_user, max_impressions_per_day,
			impressions_count, clicks_count, spend_cents,
			created_at, updated_at
		FROM advertisements
		WHERE status = 'active'
			AND start_date <= NOW()
			AND end_date >= NOW()
			AND spend_cents < total_budget_cents
			AND (target_countries IS NULL OR target_countries = '{}' OR $1 = ANY(target_countries))
			AND (target_genres IS NULL OR target_genres = '{}' OR $2 = ANY(target_genres))
			AND (target_languages IS NULL OR target_languages = '{}' OR $3 = ANY(target_languages))
			AND (target_devices IS NULL OR target_devices = '{}' OR $4 = ANY(target_devices))
		ORDER BY priority DESC, RANDOM()
		LIMIT 10
	`

	return r.queryAds(ctx, query, country, genre, language, device)
}

// IncrementImpressions incrementa el contador de impresiones
func (r *AdvertisementRepository) IncrementImpressions(ctx context.Context, adID uuid.UUID) error {
	query := `
		UPDATE advertisements
		SET impressions_count = impressions_count + 1,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.DB.ExecContext(ctx, query, adID)
	return err
}

// IncrementClicks incrementa el contador de clicks
func (r *AdvertisementRepository) IncrementClicks(ctx context.Context, adID uuid.UUID) error {
	query := `
		UPDATE advertisements
		SET clicks_count = clicks_count + 1,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.DB.ExecContext(ctx, query, adID)
	return err
}

// IncrementSpend incrementa el gasto del anuncio
func (r *AdvertisementRepository) IncrementSpend(ctx context.Context, adID uuid.UUID, amountCents int) error {
	query := `
		UPDATE advertisements
		SET spend_cents = spend_cents + $2,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.DB.ExecContext(ctx, query, adID, amountCents)
	return err
}

// queryAds es un helper para ejecutar queries y escanear múltiples ads
func (r *AdvertisementRepository) queryAds(ctx context.Context, query string, args ...interface{}) ([]*domain.Advertisement, error) {
	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ads := make([]*domain.Advertisement, 0)
	for rows.Next() {
		var ad domain.Advertisement
		err := rows.Scan(
			&ad.ID,
			&ad.CampaignID,
			&ad.Title,
			&ad.Description,
			&ad.AdvertiserName,
			&ad.AdFormat,
			&ad.AdType,
			&ad.MediaURL,
			&ad.ClickURL,
			&ad.CompanionBannerURL,
			&ad.Width,
			&ad.Height,
			&ad.DurationSeconds,
			pq.Array(&ad.TargetCountries),
			pq.Array(&ad.TargetGenres),
			pq.Array(&ad.TargetLanguages),
			pq.Array(&ad.TargetDevices),
			&ad.StartDate,
			&ad.EndDate,
			&ad.DailyBudgetCents,
			&ad.TotalBudgetCents,
			&ad.CPMRateCents,
			&ad.CPCRateCents,
			&ad.Status,
			&ad.Priority,
			&ad.MaxImpressionsPerUser,
			&ad.MaxImpressionsPerDay,
			&ad.ImpressionsCount,
			&ad.ClicksCount,
			&ad.SpendCents,
			&ad.CreatedAt,
			&ad.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		ads = append(ads, &ad)
	}

	return ads, nil
}
