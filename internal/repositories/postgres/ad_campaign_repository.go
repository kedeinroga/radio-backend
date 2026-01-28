package postgres

import (
	"context"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/google/uuid"
)

// AdCampaignRepository implementa domain.AdCampaignRepository
type AdCampaignRepository struct {
	db *database.Connection
}

// NewAdCampaignRepository crea una nueva instancia del repositorio
func NewAdCampaignRepository(db *database.Connection) *AdCampaignRepository {
	return &AdCampaignRepository{db: db}
}

// Create crea una nueva campaña
func (r *AdCampaignRepository) Create(ctx context.Context, campaign *domain.AdCampaign) error {
	query := `
		INSERT INTO ad_campaigns (
			id, advertiser_id, name, description,
			total_budget_cents, daily_budget_cents, spent_cents,
			start_date, end_date, status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		campaign.ID,
		campaign.AdvertiserID,
		campaign.Name,
		campaign.Description,
		campaign.TotalBudgetCents,
		campaign.DailyBudgetCents,
		campaign.SpentCents,
		campaign.StartDate,
		campaign.EndDate,
		campaign.Status,
		time.Now(),
		time.Now(),
	)

	return err
}

// GetByID obtiene una campaña por ID
func (r *AdCampaignRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AdCampaign, error) {
	query := `
		SELECT
			id, advertiser_id, name, description,
			total_budget_cents, daily_budget_cents, spent_cents,
			start_date, end_date, status,
			created_at, updated_at
		FROM ad_campaigns
		WHERE id = $1
	`

	var campaign domain.AdCampaign
	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&campaign.ID,
		&campaign.AdvertiserID,
		&campaign.Name,
		&campaign.Description,
		&campaign.TotalBudgetCents,
		&campaign.DailyBudgetCents,
		&campaign.SpentCents,
		&campaign.StartDate,
		&campaign.EndDate,
		&campaign.Status,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("campaign not found: %w", err)
	}

	return &campaign, nil
}

// GetByAdvertiserID obtiene todas las campañas de un advertiser
func (r *AdCampaignRepository) GetByAdvertiserID(ctx context.Context, advertiserID uuid.UUID) ([]*domain.AdCampaign, error) {
	query := `
		SELECT
			id, advertiser_id, name, description,
			total_budget_cents, daily_budget_cents, spent_cents,
			start_date, end_date, status,
			created_at, updated_at
		FROM ad_campaigns
		WHERE advertiser_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, advertiserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := make([]*domain.AdCampaign, 0)
	for rows.Next() {
		var campaign domain.AdCampaign
		err := rows.Scan(
			&campaign.ID,
			&campaign.AdvertiserID,
			&campaign.Name,
			&campaign.Description,
			&campaign.TotalBudgetCents,
			&campaign.DailyBudgetCents,
			&campaign.SpentCents,
			&campaign.StartDate,
			&campaign.EndDate,
			&campaign.Status,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, &campaign)
	}

	return campaigns, nil
}

// Update actualiza una campaña
func (r *AdCampaignRepository) Update(ctx context.Context, campaign *domain.AdCampaign) error {
	query := `
		UPDATE ad_campaigns
		SET name = $2,
			description = $3,
			total_budget_cents = $4,
			daily_budget_cents = $5,
			spent_cents = $6,
			start_date = $7,
			end_date = $8,
			status = $9,
			updated_at = $10
		WHERE id = $1
	`

	_, err := r.db.DB.ExecContext(ctx, query,
		campaign.ID,
		campaign.Name,
		campaign.Description,
		campaign.TotalBudgetCents,
		campaign.DailyBudgetCents,
		campaign.SpentCents,
		campaign.StartDate,
		campaign.EndDate,
		campaign.Status,
		time.Now(),
	)

	return err
}

// Delete elimina una campaña
func (r *AdCampaignRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ad_campaigns WHERE id = $1`

	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

// GetActiveCampaigns obtiene todas las campañas activas
func (r *AdCampaignRepository) GetActiveCampaigns(ctx context.Context) ([]*domain.AdCampaign, error) {
	query := `
		SELECT
			id, advertiser_id, name, description,
			total_budget_cents, daily_budget_cents, spent_cents,
			start_date, end_date, status,
			created_at, updated_at
		FROM ad_campaigns
		WHERE status = 'active'
			AND start_date <= NOW()
			AND end_date >= NOW()
			AND spent_cents < total_budget_cents
		ORDER BY created_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := make([]*domain.AdCampaign, 0)
	for rows.Next() {
		var campaign domain.AdCampaign
		err := rows.Scan(
			&campaign.ID,
			&campaign.AdvertiserID,
			&campaign.Name,
			&campaign.Description,
			&campaign.TotalBudgetCents,
			&campaign.DailyBudgetCents,
			&campaign.SpentCents,
			&campaign.StartDate,
			&campaign.EndDate,
			&campaign.Status,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, &campaign)
	}

	return campaigns, nil
}

// IncrementSpent incrementa el gasto de una campaña
func (r *AdCampaignRepository) IncrementSpent(ctx context.Context, campaignID uuid.UUID, amountCents int) error {
	query := `
		UPDATE ad_campaigns
		SET spent_cents = spent_cents + $2,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.DB.ExecContext(ctx, query, campaignID, amountCents)
	return err
}

// GetExpiredCampaigns obtiene campañas que deberían expirar
func (r *AdCampaignRepository) GetExpiredCampaigns(ctx context.Context) ([]*domain.AdCampaign, error) {
	query := `
		SELECT
			id, advertiser_id, name, description,
			total_budget_cents, daily_budget_cents, spent_cents,
			start_date, end_date, status,
			created_at, updated_at
		FROM ad_campaigns
		WHERE status = 'active'
			AND (end_date < NOW() OR spent_cents >= total_budget_cents)
	`

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := make([]*domain.AdCampaign, 0)
	for rows.Next() {
		var campaign domain.AdCampaign
		err := rows.Scan(
			&campaign.ID,
			&campaign.AdvertiserID,
			&campaign.Name,
			&campaign.Description,
			&campaign.TotalBudgetCents,
			&campaign.DailyBudgetCents,
			&campaign.SpentCents,
			&campaign.StartDate,
			&campaign.EndDate,
			&campaign.Status,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, &campaign)
	}

	return campaigns, nil
}
