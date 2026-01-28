package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CampaignStatus representa el estado de una campaña publicitaria
type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusExpired   CampaignStatus = "expired"
)

// AdCampaign representa una campaña publicitaria
type AdCampaign struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	AdvertiserID     uuid.UUID      `json:"advertiser_id" db:"advertiser_id"`
	Name             string         `json:"name" db:"name"`
	Description      *string        `json:"description,omitempty" db:"description"`
	TotalBudgetCents int            `json:"total_budget_cents" db:"total_budget_cents"`
	DailyBudgetCents *int           `json:"daily_budget_cents,omitempty" db:"daily_budget_cents"`
	SpentCents       int            `json:"spent_cents" db:"spent_cents"`
	StartDate        time.Time      `json:"start_date" db:"start_date"`
	EndDate          time.Time      `json:"end_date" db:"end_date"`
	Status           CampaignStatus `json:"status" db:"status"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

// Validate valida los datos de la campaña
func (c *AdCampaign) Validate() error {
	if c.Name == "" {
		return ErrInvalidCampaignName
	}
	if c.TotalBudgetCents <= 0 {
		return ErrInvalidBudget
	}
	if c.DailyBudgetCents != nil && *c.DailyBudgetCents <= 0 {
		return ErrInvalidBudget
	}
	if c.EndDate.Before(c.StartDate) || c.EndDate.Equal(c.StartDate) {
		return ErrInvalidCampaignDates
	}
	return nil
}

// IsActive verifica si la campaña está activa actualmente
func (c *AdCampaign) IsActive() bool {
	now := time.Now()
	return c.Status == CampaignStatusActive &&
		now.After(c.StartDate) &&
		now.Before(c.EndDate)
}

// HasBudget verifica si la campaña tiene presupuesto disponible
func (c *AdCampaign) HasBudget() bool {
	return c.SpentCents < c.TotalBudgetCents
}

// RemainingBudget retorna el presupuesto restante en centavos
func (c *AdCampaign) RemainingBudget() int {
	remaining := c.TotalBudgetCents - c.SpentCents
	if remaining < 0 {
		return 0
	}
	return remaining
}

// BudgetUtilization retorna el porcentaje de presupuesto utilizado (0-100)
func (c *AdCampaign) BudgetUtilization() float64 {
	if c.TotalBudgetCents == 0 {
		return 0
	}
	return (float64(c.SpentCents) / float64(c.TotalBudgetCents)) * 100
}

// CanActivate verifica si la campaña puede ser activada
func (c *AdCampaign) CanActivate() bool {
	now := time.Now()
	return c.Status == CampaignStatusDraft &&
		c.HasBudget() &&
		now.Before(c.EndDate)
}

// ShouldExpire verifica si la campaña debería expirar
func (c *AdCampaign) ShouldExpire() bool {
	return time.Now().After(c.EndDate) && c.Status == CampaignStatusActive
}

// AdCampaignRepository define la interfaz para persistencia de campañas
type AdCampaignRepository interface {
	Create(ctx context.Context, campaign *AdCampaign) error
	GetByID(ctx context.Context, id uuid.UUID) (*AdCampaign, error)
	GetByAdvertiserID(ctx context.Context, advertiserID uuid.UUID) ([]*AdCampaign, error)
	Update(ctx context.Context, campaign *AdCampaign) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveCampaigns(ctx context.Context) ([]*AdCampaign, error)
	IncrementSpent(ctx context.Context, campaignID uuid.UUID, amountCents int) error
	GetExpiredCampaigns(ctx context.Context) ([]*AdCampaign, error)
}
