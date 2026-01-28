package services

import (
	"context"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/logger"

	"github.com/google/uuid"
)

// Service-level errors
var (
	ErrInvalidCampaignStatus = fmt.Errorf("invalid campaign status for this operation")
)

// AdCampaignService maneja la lógica de negocio de campañas publicitarias
type AdCampaignService struct {
	repo domain.AdCampaignRepository
}

// NewAdCampaignService crea una nueva instancia del servicio
func NewAdCampaignService(repo domain.AdCampaignRepository) *AdCampaignService {
	return &AdCampaignService{
		repo: repo,
	}
}

// CreateCampaign crea una nueva campaña con validaciones
func (s *AdCampaignService) CreateCampaign(ctx context.Context, campaign *domain.AdCampaign) error {
	logger.Info("creating new ad campaign", "name", campaign.Name, "advertiser_id", campaign.AdvertiserID)

	// Validar datos
	if err := campaign.Validate(); err != nil {
		logger.Error("campaign validation failed", "error", err)
		return err
	}

	// Generar ID si no existe
	if campaign.ID == uuid.Nil {
		campaign.ID = uuid.New()
	}

	// Establecer timestamps
	now := time.Now()
	campaign.CreatedAt = now
	campaign.UpdatedAt = now

	// Inicializar contadores
	campaign.SpentCents = 0

	// Estado inicial: draft o active según fechas
	if campaign.StartDate.After(now) {
		campaign.Status = domain.CampaignStatusDraft
	}

	// Crear en BD
	if err := s.repo.Create(ctx, campaign); err != nil {
		logger.Error("failed to create campaign", "error", err)
		return fmt.Errorf("failed to create campaign: %w", err)
	}

	logger.Info("campaign created successfully", "id", campaign.ID, "name", campaign.Name)
	return nil
}

// GetCampaign obtiene una campaña por ID
func (s *AdCampaignService) GetCampaign(ctx context.Context, id uuid.UUID) (*domain.AdCampaign, error) {
	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("failed to get campaign", "id", id, "error", err)
		return nil, fmt.Errorf("campaign not found: %w", err)
	}
	return campaign, nil
}

// GetCampaignsByAdvertiser obtiene todas las campañas de un advertiser
func (s *AdCampaignService) GetCampaignsByAdvertiser(ctx context.Context, advertiserID uuid.UUID) ([]*domain.AdCampaign, error) {
	campaigns, err := s.repo.GetByAdvertiserID(ctx, advertiserID)
	if err != nil {
		logger.Error("failed to get campaigns by advertiser", "advertiser_id", advertiserID, "error", err)
		return nil, err
	}
	return campaigns, nil
}

// UpdateCampaign actualiza una campaña existente
func (s *AdCampaignService) UpdateCampaign(ctx context.Context, campaign *domain.AdCampaign) error {
	logger.Info("updating campaign", "id", campaign.ID)

	// Validar
	if err := campaign.Validate(); err != nil {
		logger.Error("campaign validation failed", "error", err)
		return err
	}

	// Verificar que existe
	existing, err := s.repo.GetByID(ctx, campaign.ID)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	// No permitir cambiar advertiser_id
	campaign.AdvertiserID = existing.AdvertiserID

	// Actualizar timestamp
	campaign.UpdatedAt = time.Now()

	// Actualizar en BD
	if err := s.repo.Update(ctx, campaign); err != nil {
		logger.Error("failed to update campaign", "error", err)
		return fmt.Errorf("failed to update campaign: %w", err)
	}

	logger.Info("campaign updated successfully", "id", campaign.ID)
	return nil
}

// DeleteCampaign elimina una campaña
func (s *AdCampaignService) DeleteCampaign(ctx context.Context, id uuid.UUID) error {
	logger.Info("deleting campaign", "id", id)

	// Verificar que existe
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		logger.Error("failed to delete campaign", "error", err)
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	logger.Info("campaign deleted successfully", "id", id)
	return nil
}

// PauseCampaign pausa una campaña activa
func (s *AdCampaignService) PauseCampaign(ctx context.Context, id uuid.UUID) error {
	logger.Info("pausing campaign", "id", id)

	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if campaign.Status != domain.CampaignStatusActive {
		return ErrInvalidCampaignStatus
	}

	campaign.Status = domain.CampaignStatusPaused
	campaign.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, campaign); err != nil {
		logger.Error("failed to pause campaign", "error", err)
		return err
	}

	logger.Info("campaign paused successfully", "id", id)
	return nil
}

// ResumeCampaign reactiva una campaña pausada
func (s *AdCampaignService) ResumeCampaign(ctx context.Context, id uuid.UUID) error {
	logger.Info("resuming campaign", "id", id)

	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if campaign.Status != domain.CampaignStatusPaused {
		return ErrInvalidCampaignStatus
	}

	// Verificar que aún está en período válido
	now := time.Now()
	if now.Before(campaign.StartDate) || now.After(campaign.EndDate) {
		return domain.ErrInvalidCampaignDates
	}

	campaign.Status = domain.CampaignStatusActive
	campaign.UpdatedAt = now

	if err := s.repo.Update(ctx, campaign); err != nil {
		logger.Error("failed to resume campaign", "error", err)
		return err
	}

	logger.Info("campaign resumed successfully", "id", id)
	return nil
}

// GetActiveCampaigns obtiene todas las campañas activas
func (s *AdCampaignService) GetActiveCampaigns(ctx context.Context) ([]*domain.AdCampaign, error) {
	campaigns, err := s.repo.GetActiveCampaigns(ctx)
	if err != nil {
		logger.Error("failed to get active campaigns", "error", err)
		return nil, err
	}
	return campaigns, nil
}

// RecordSpend registra gasto en una campaña
func (s *AdCampaignService) RecordSpend(ctx context.Context, campaignID uuid.UUID, amountCents int) error {
	logger.Info("recording spend", "campaign_id", campaignID, "amount_cents", amountCents)

	// Verificar que la campaña existe y está activa
	campaign, err := s.repo.GetByID(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if !campaign.IsActive() {
		return fmt.Errorf("campaign is not active")
	}

	// Verificar que tiene presupuesto disponible
	if !campaign.HasBudget() {
		return domain.ErrCampaignBudgetExhausted
	}

	// Incrementar gasto
	if err := s.repo.IncrementSpent(ctx, campaignID, amountCents); err != nil {
		logger.Error("failed to increment spend", "error", err)
		return err
	}

	// Verificar si se agotó el presupuesto
	campaign, _ = s.repo.GetByID(ctx, campaignID)
	if !campaign.HasBudget() {
		logger.Warn("campaign budget exhausted", "campaign_id", campaignID)
		campaign.Status = domain.CampaignStatusExpired
		s.repo.Update(ctx, campaign)
	}

	return nil
}

// ProcessExpiredCampaigns procesa campañas que agotaron presupuesto
// Debe ser llamado por un cron job periódicamente
func (s *AdCampaignService) ProcessExpiredCampaigns(ctx context.Context) (int, error) {
	logger.Info("processing expired campaigns")

	campaigns, err := s.repo.GetExpiredCampaigns(ctx)
	if err != nil {
		logger.Error("failed to get expired campaigns", "error", err)
		return 0, err
	}

	count := 0
	for _, campaign := range campaigns {
		if campaign.Status != domain.CampaignStatusExpired {
			campaign.Status = domain.CampaignStatusExpired
			campaign.UpdatedAt = time.Now()
			if err := s.repo.Update(ctx, campaign); err != nil {
				logger.Error("failed to update expired campaign", "id", campaign.ID, "error", err)
				continue
			}
			count++
		}
	}

	logger.Info("processed expired campaigns", "count", count)
	return count, nil
}

// GetCampaignStats obtiene estadísticas de una campaña
func (s *AdCampaignService) GetCampaignStats(ctx context.Context, id uuid.UUID) (*CampaignStats, error) {
	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("campaign not found: %w", err)
	}

	stats := &CampaignStats{
		CampaignID:        campaign.ID,
		Name:              campaign.Name,
		Status:            campaign.Status,
		TotalBudgetCents:  campaign.TotalBudgetCents,
		SpentCents:        campaign.SpentCents,
		RemainingCents:    campaign.RemainingBudget(),
		BudgetUtilization: campaign.BudgetUtilization(),
		StartDate:         campaign.StartDate,
		EndDate:           campaign.EndDate,
		DaysRemaining:     int(time.Until(campaign.EndDate).Hours() / 24),
		IsActive:          campaign.IsActive(),
	}

	return stats, nil
}

// CampaignStats representa estadísticas de una campaña
type CampaignStats struct {
	CampaignID        uuid.UUID             `json:"campaign_id"`
	Name              string                `json:"name"`
	Status            domain.CampaignStatus `json:"status"`
	TotalBudgetCents  int                   `json:"total_budget_cents"`
	SpentCents        int                   `json:"spent_cents"`
	RemainingCents    int                   `json:"remaining_cents"`
	BudgetUtilization float64               `json:"budget_utilization"`
	StartDate         time.Time             `json:"start_date"`
	EndDate           time.Time             `json:"end_date"`
	DaysRemaining     int                   `json:"days_remaining"`
	IsActive          bool                  `json:"is_active"`
}
