package services

import (
	"context"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/cache"
	"radio-backend/internal/infrastructure/logger"

	"github.com/google/uuid"
)

// AdvertisementService maneja la lógica de negocio de anuncios
type AdvertisementService struct {
	repo         domain.AdvertisementRepository
	campaignRepo domain.AdCampaignRepository
	adCache      *cache.AdCache
}

// NewAdvertisementService crea una nueva instancia del servicio
func NewAdvertisementService(
	repo domain.AdvertisementRepository,
	campaignRepo domain.AdCampaignRepository,
	adCache *cache.AdCache,
) *AdvertisementService {
	return &AdvertisementService{
		repo:         repo,
		campaignRepo: campaignRepo,
		adCache:      adCache,
	}
}

// CreateAdvertisement crea un nuevo anuncio con validaciones
func (s *AdvertisementService) CreateAdvertisement(ad *domain.Advertisement) error {
	logger.Info("creating new advertisement", "title", ad.Title, "campaign_id", ad.CampaignID)

	// Validar datos
	if err := ad.Validate(); err != nil {
		logger.Error("advertisement validation failed", "error", err)
		return err
	}

	// Verificar que la campaña existe y está activa
	campaign, err := s.campaignRepo.GetByID(ad.CampaignID)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if !campaign.IsActive() {
		return fmt.Errorf("campaign is not active")
	}

	// Generar ID si no existe
	if ad.ID == uuid.Nil {
		ad.ID = uuid.New()
	}

	// Establecer timestamps
	now := time.Now()
	ad.CreatedAt = now
	ad.UpdatedAt = now

	// Inicializar contadores
	ad.ImpressionsCount = 0
	ad.ClicksCount = 0
	ad.SpendCents = 0

	// Crear en BD
	if err := s.repo.Create(ad); err != nil {
		logger.Error("failed to create advertisement", "error", err)
		return fmt.Errorf("failed to create advertisement: %w", err)
	}

	logger.Info("advertisement created successfully", "id", ad.ID, "title", ad.Title)
	return nil
}

// GetAdvertisement obtiene un anuncio por ID
func (s *AdvertisementService) GetAdvertisement(id uuid.UUID) (*domain.Advertisement, error) {
	ad, err := s.repo.GetByID(id)
	if err != nil {
		logger.Error("failed to get advertisement", "id", id, "error", err)
		return nil, fmt.Errorf("advertisement not found: %w", err)
	}
	return ad, nil
}

// GetAdvertisementsByCampaign obtiene todos los anuncios de una campaña
func (s *AdvertisementService) GetAdvertisementsByCampaign(campaignID uuid.UUID) ([]*domain.Advertisement, error) {
	ads, err := s.repo.GetByCampaignID(campaignID)
	if err != nil {
		logger.Error("failed to get advertisements by campaign", "campaign_id", campaignID, "error", err)
		return nil, err
	}
	return ads, nil
}

// UpdateAdvertisement actualiza un anuncio existente
func (s *AdvertisementService) UpdateAdvertisement(ad *domain.Advertisement) error {
	logger.Info("updating advertisement", "id", ad.ID)

	// Validar
	if err := ad.Validate(); err != nil {
		logger.Error("advertisement validation failed", "error", err)
		return err
	}

	// Verificar que existe
	existing, err := s.repo.GetByID(ad.ID)
	if err != nil {
		return fmt.Errorf("advertisement not found: %w", err)
	}

	// No permitir cambiar campaign_id
	ad.CampaignID = existing.CampaignID

	// Actualizar timestamp
	ad.UpdatedAt = time.Now()

	// Actualizar en BD
	if err := s.repo.Update(ad); err != nil {
		logger.Error("failed to update advertisement", "error", err)
		return fmt.Errorf("failed to update advertisement: %w", err)
	}

	logger.Info("advertisement updated successfully", "id", ad.ID)
	return nil
}

// DeleteAdvertisement elimina un anuncio
func (s *AdvertisementService) DeleteAdvertisement(id uuid.UUID) error {
	logger.Info("deleting advertisement", "id", id)

	// Verificar que existe
	_, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("advertisement not found: %w", err)
	}

	if err := s.repo.Delete(id); err != nil {
		logger.Error("failed to delete advertisement", "error", err)
		return fmt.Errorf("failed to delete advertisement: %w", err)
	}

	logger.Info("advertisement deleted successfully", "id", id)
	return nil
}

// GetEligibleAdsForUser obtiene anuncios elegibles para un usuario
// Aplica targeting, frequency capping y filtros de premium
func (s *AdvertisementService) GetEligibleAdsForUser(
	userID uuid.UUID,
	country, genre, language, device string,
	isPremium bool,
) ([]*domain.Advertisement, error) {
	logger.Info("getting eligible ads for user",
		"user_id", userID,
		"country", country,
		"genre", genre,
		"is_premium", isPremium,
	)

	// Los usuarios premium no ven anuncios
	if isPremium {
		logger.Info("user is premium, no ads", "user_id", userID)
		return []*domain.Advertisement{}, nil
	}

	// Verificar frequency capping usando cache
	canShow, err := s.checkFrequencyCapping(userID)
	if err != nil {
		logger.Error("failed to check frequency capping", "error", err)
		// Continuar de todos modos (graceful degradation)
	}
	if !canShow {
		logger.Info("frequency cap exceeded for user", "user_id", userID)
		return []*domain.Advertisement{}, nil
	}

	// Obtener anuncios elegibles con targeting
	ads, err := s.repo.GetEligibleAds(country, genre, language, device)
	if err != nil {
		logger.Error("failed to get eligible ads", "error", err)
		return nil, err
	}

	logger.Info("found eligible ads", "count", len(ads), "user_id", userID)
	return ads, nil
}

// checkFrequencyCapping verifica si el usuario puede ver más anuncios
func (s *AdvertisementService) checkFrequencyCapping(userID uuid.UUID) (bool, error) {
	ctx := context.Background()

	// Obtener contadores del cache
	hourlyCount, err := s.adCache.GetUserAdCountHourly(ctx, userID)
	if err != nil {
		return true, err // Graceful degradation
	}

	dailyCount, err := s.adCache.GetUserAdCountDaily(ctx, userID)
	if err != nil {
		return true, err // Graceful degradation
	}

	// Límites de frecuencia (debería venir de config)
	const (
		maxAdsPerHour = 6
		maxAdsPerDay  = 30
	)

	if hourlyCount >= maxAdsPerHour {
		logger.Warn("hourly frequency cap exceeded", "user_id", userID, "count", hourlyCount)
		return false, nil
	}

	if dailyCount >= maxAdsPerDay {
		logger.Warn("daily frequency cap exceeded", "user_id", userID, "count", dailyCount)
		return false, nil
	}

	return true, nil
}

// RecordImpression registra la impresión de un anuncio
func (s *AdvertisementService) RecordImpression(adID uuid.UUID) error {
	logger.Info("recording impression", "ad_id", adID)

	// Incrementar contador en BD
	if err := s.repo.IncrementImpressions(adID); err != nil {
		logger.Error("failed to increment impressions", "error", err)
		return err
	}

	return nil
}

// RecordClick registra el click de un anuncio
func (s *AdvertisementService) RecordClick(adID uuid.UUID) error {
	logger.Info("recording click", "ad_id", adID)

	// Incrementar contador en BD
	if err := s.repo.IncrementClicks(adID); err != nil {
		logger.Error("failed to increment clicks", "error", err)
		return err
	}

	return nil
}

// RecordSpend registra gasto en un anuncio
func (s *AdvertisementService) RecordSpend(adID uuid.UUID, amountCents int) error {
	logger.Info("recording spend", "ad_id", adID, "amount_cents", amountCents)

	// Verificar que el anuncio existe y está activo
	ad, err := s.repo.GetByID(adID)
	if err != nil {
		return fmt.Errorf("advertisement not found: %w", err)
	}

	if ad.Status != domain.AdStatusActive {
		return domain.ErrAdvertisementNotActive
	}

	// Verificar presupuesto
	if ad.SpendCents+amountCents > ad.TotalBudgetCents {
		return domain.ErrCampaignBudgetExhausted
	}

	// Incrementar gasto
	if err := s.repo.IncrementSpend(adID, amountCents); err != nil {
		logger.Error("failed to increment spend", "error", err)
		return err
	}

	// También incrementar en la campaña
	if err := s.campaignRepo.IncrementSpent(ad.CampaignID, amountCents); err != nil {
		logger.Error("failed to increment campaign spend", "error", err)
		// No retornar error, ya se registró en el ad
	}

	return nil
}

// GetAdvertisementStats obtiene estadísticas de un anuncio
func (s *AdvertisementService) GetAdvertisementStats(id uuid.UUID) (*AdvertisementStats, error) {
	ad, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("advertisement not found: %w", err)
	}

	ctr := 0.0
	if ad.ImpressionsCount > 0 {
		ctr = (float64(ad.ClicksCount) / float64(ad.ImpressionsCount)) * 100
	}

	stats := &AdvertisementStats{
		AdvertisementID:  ad.ID,
		Title:            ad.Title,
		Status:           ad.Status,
		ImpressionsCount: ad.ImpressionsCount,
		ClicksCount:      ad.ClicksCount,
		CTR:              ctr,
		SpendCents:       ad.SpendCents,
		TotalBudgetCents: ad.TotalBudgetCents,
		RemainingCents:   ad.TotalBudgetCents - ad.SpendCents,
		AverageCPM:       s.calculateAverageCPM(ad),
		AverageCPC:       s.calculateAverageCPC(ad),
	}

	return stats, nil
}

func (s *AdvertisementService) calculateAverageCPM(ad *domain.Advertisement) float64 {
	if ad.ImpressionsCount == 0 {
		return 0
	}
	return (float64(ad.SpendCents) / float64(ad.ImpressionsCount)) * 1000
}

func (s *AdvertisementService) calculateAverageCPC(ad *domain.Advertisement) float64 {
	if ad.ClicksCount == 0 {
		return 0
	}
	return float64(ad.SpendCents) / float64(ad.ClicksCount)
}

// AdvertisementStats representa estadísticas de un anuncio
type AdvertisementStats struct {
	AdvertisementID  uuid.UUID       `json:"advertisement_id"`
	Title            string          `json:"title"`
	Status           domain.AdStatus `json:"status"`
	ImpressionsCount int             `json:"impressions_count"`
	ClicksCount      int             `json:"clicks_count"`
	CTR              float64         `json:"ctr"` // Click-Through Rate (%)
	SpendCents       int             `json:"spend_cents"`
	TotalBudgetCents int             `json:"total_budget_cents"`
	RemainingCents   int             `json:"remaining_cents"`
	AverageCPM       float64         `json:"average_cpm"` // Cost Per Mille
	AverageCPC       float64         `json:"average_cpc"` // Cost Per Click
}
