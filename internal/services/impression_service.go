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

// ImpressionService maneja la lógica de negocio de impresiones
type ImpressionService struct {
	repo                domain.AdImpressionRepository
	adRepo              domain.AdvertisementRepository
	userProfileRepo     domain.UserAdProfileRepository
	adCache             *cache.AdCache
	securityService     domain.AdSecurityService
	fraudScoreThreshold float64
}

// NewImpressionService crea una nueva instancia del servicio
func NewImpressionService(
	repo domain.AdImpressionRepository,
	adRepo domain.AdvertisementRepository,
	userProfileRepo domain.UserAdProfileRepository,
	adCache *cache.AdCache,
	securityService domain.AdSecurityService,
	fraudScoreThreshold float64,
) *ImpressionService {
	return &ImpressionService{
		repo:                repo,
		adRepo:              adRepo,
		userProfileRepo:     userProfileRepo,
		adCache:             adCache,
		securityService:     securityService,
		fraudScoreThreshold: fraudScoreThreshold,
	}
}

// RecordImpression registra una impresión con validaciones de seguridad
func (s *ImpressionService) RecordImpression(impression *domain.AdImpression) error {
	logger.Info("recording impression",
		"ad_id", impression.AdvertisementID,
		"user_id", impression.UserID,
		"ip", impression.IPAddress,
	)

	// 1. Validar datos básicos
	if err := s.validateImpressionData(impression); err != nil {
		logger.Error("impression validation failed", "error", err)
		return err
	}

	// 2. Verificar que el anuncio existe y está activo
	ad, err := s.adRepo.GetByID(impression.AdvertisementID)
	if err != nil {
		return domain.ErrAdvertisementNotFound
	}
	if ad.Status != domain.AdStatusActive {
		return domain.ErrAdvertisementNotActive
	}

	// 3. Verificar frequency capping del usuario
	if impression.UserID != nil {
		canShow, err := s.checkUserFrequencyCapping(*impression.UserID)
		if err != nil {
			logger.Error("failed to check frequency capping", "error", err)
			// Continuar con graceful degradation
		}
		if !canShow {
			logger.Warn("frequency cap exceeded", "user_id", *impression.UserID)
			return domain.ErrFrequencyCapExceeded
		}
	}

	// 4. Detectar fraude
	fraudScore, err := s.detectFraud(impression)
	if err != nil {
		logger.Error("fraud detection failed", "error", err)
		// Continuar con graceful degradation
		fraudScore = 0.0
	}

	if fraudScore >= s.fraudScoreThreshold {
		logger.Warn("high fraud score detected",
			"score", fraudScore,
			"threshold", s.fraudScoreThreshold,
			"ip", impression.IPAddress,
		)
		return domain.ErrSuspiciousActivity
	}

	// 5. Generar token de impresión
	token, err := s.securityService.GenerateToken(
		impression.AdvertisementID,
		impression.SessionID,
	)
	if err != nil {
		logger.Error("failed to generate impression token", "error", err)
		return err
	}
	impression.ImpressionToken = token

	// 6. Establecer timestamps
	now := time.Now()
	if impression.ID == uuid.Nil {
		impression.ID = uuid.New()
	}
	impression.CreatedAt = now

	// 7. Guardar en BD
	if err := s.repo.Create(impression); err != nil {
		logger.Error("failed to create impression", "error", err)
		return fmt.Errorf("failed to create impression: %w", err)
	}

	// 8. Incrementar contadores (asíncrono)
	go s.updateCounters(impression)

	logger.Info("impression recorded successfully", "id", impression.ID)
	return nil
}

// validateImpressionData valida los datos básicos de la impresión
func (s *ImpressionService) validateImpressionData(impression *domain.AdImpression) error {
	if impression.AdvertisementID == uuid.Nil {
		return domain.ErrInvalidImpressionData
	}
	if impression.IPAddress == "" {
		return domain.ErrInvalidImpressionData
	}
	if impression.UserAgent == "" {
		return domain.ErrInvalidImpressionData
	}
	if impression.SessionID == "" {
		return domain.ErrInvalidImpressionData
	}
	return nil
}

// checkUserFrequencyCapping verifica los límites de frecuencia del usuario
func (s *ImpressionService) checkUserFrequencyCapping(userID uuid.UUID) (bool, error) {
	ctx := context.Background()

	// Verificar contador horario
	hourlyCount, err := s.adCache.GetUserAdCountHourly(ctx, userID)
	if err == nil && hourlyCount >= 6 {
		return false, nil
	}

	// Verificar contador diario
	dailyCount, err := s.adCache.GetUserAdCountDaily(ctx, userID)
	if err == nil && dailyCount >= 30 {
		return false, nil
	}

	return true, nil
}

// detectFraud analiza la impresión para detectar fraude
func (s *ImpressionService) detectFraud(impression *domain.AdImpression) (float64, error) {
	ctx := context.Background()

	// Contar impresiones recientes desde esta IP
	since := time.Now().Add(-5 * time.Minute)
	ipImpressions, err := s.repo.CountByIPAddress(impression.IPAddress, since)
	if err != nil {
		return 0, err
	}

	// Verificar replay attack (mismo session_id usado múltiples veces)
	recentSessions, err := s.repo.GetRecentBySessionID(impression.SessionID, since)
	if err != nil {
		return 0, err
	}

	// Contar clicks desde esta IP
	ipClicks, err := s.adCache.CountIPClicks(ctx, impression.IPAddress, 5*time.Minute)
	if err != nil {
		// Graceful degradation
		ipClicks = 0
	}

	// Calcular score de fraude (0.0 = sin fraude, 1.0 = fraude total)
	fraudScore := s.calculateFraudScore(ipImpressions, len(recentSessions), int64(ipClicks))

	return fraudScore, nil
}

// calculateFraudScore calcula un score de fraude basado en métricas
func (s *ImpressionService) calculateFraudScore(ipImpressions int64, sessionReplays int, ipClicks int64) float64 {
	score := 0.0

	// Muchas impresiones desde la misma IP (5 min)
	if ipImpressions > 20 {
		score += 0.4
	} else if ipImpressions > 10 {
		score += 0.2
	}

	// Session replay
	if sessionReplays > 1 {
		score += 0.4
	}

	// Muchos clicks desde la misma IP (5 min)
	if ipClicks > 10 {
		score += 0.3
	} else if ipClicks > 5 {
		score += 0.15
	}

	// Normalizar a 0-1
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// updateCounters actualiza contadores en BD y cache (llamado asíncronamente)
func (s *ImpressionService) updateCounters(impression *domain.AdImpression) {
	ctx := context.Background()

	// Incrementar contador en anuncio
	if err := s.adRepo.IncrementImpressions(impression.AdvertisementID); err != nil {
		logger.Error("failed to increment ad impressions", "error", err)
	}

	// Incrementar contadores de usuario en cache
	if impression.UserID != nil {
		if _, err := s.adCache.IncrementUserAdCountHourly(ctx, *impression.UserID); err != nil {
			logger.Error("failed to increment hourly count", "error", err)
		}
		if _, err := s.adCache.IncrementUserAdCountDaily(ctx, *impression.UserID); err != nil {
			logger.Error("failed to increment daily count", "error", err)
		}

		// Actualizar perfil de usuario
		if err := s.userProfileRepo.IncrementAdsShown(*impression.UserID); err != nil {
			logger.Error("failed to increment user ads shown", "error", err)
		}
	}

	// Track en cache para fraud detection
	if err := s.adCache.TrackIPImpression(ctx, impression.IPAddress, impression.AdvertisementID, 1*time.Hour); err != nil {
		logger.Error("failed to track IP impression", "error", err)
	}
}

// ValidateImpressionToken valida un token de impresión
func (s *ImpressionService) ValidateImpressionToken(token string) (*domain.ImpressionToken, error) {
	return s.securityService.ValidateToken(token)
}

// CountViewableImpressions cuenta impresiones viewables (>= 1000ms)
func (s *ImpressionService) CountViewableImpressions(adID uuid.UUID, since time.Time) (int64, error) {
	return s.repo.CountViewableImpressions(adID, since)
}

// GetImpressionsByAdvertisement obtiene impresiones de un anuncio
func (s *ImpressionService) GetImpressionsByAdvertisement(adID uuid.UUID, limit int) ([]*domain.AdImpression, error) {
	return s.repo.GetByAdvertisementID(adID, limit)
}

// GetImpressionsByUser obtiene impresiones de un usuario desde una fecha
func (s *ImpressionService) GetImpressionsByUser(userID uuid.UUID, since time.Time) ([]*domain.AdImpression, error) {
	return s.repo.GetByUserID(userID, since)
}
