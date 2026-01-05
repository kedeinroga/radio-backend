package jobs

import (
	"context"
	"log/slog"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/services"
)

// CleanupJobs maneja los trabajos de limpieza del sistema
type CleanupJobs struct {
	sessionService *services.StreamSessionService
	adRepo         domain.AdvertisementRepository
	logger         *slog.Logger
}

// NewCleanupJobs crea una nueva instancia de CleanupJobs
func NewCleanupJobs(
	sessionService *services.StreamSessionService,
	adRepo domain.AdvertisementRepository,
	logger *slog.Logger,
) *CleanupJobs {
	return &CleanupJobs{
		sessionService: sessionService,
		adRepo:         adRepo,
		logger:         logger,
	}
}

// CleanAbandonedSessions limpia sesiones de streaming abandonadas
// Ejecuta cada 5 minutos
func (j *CleanupJobs) CleanAbandonedSessions() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := j.sessionService.CleanupAbandonedSessions(ctx)
	if err != nil {
		j.logger.Error("failed to cleanup abandoned sessions",
			"error", err,
		)
		return
	}

	if count > 0 {
		j.logger.Info("abandoned sessions cleaned",
			"count", count,
		)
	}
}

// CleanTokenBlacklist limpia tokens JWT expirados del blacklist
// Ejecuta cada 1 hora
// TODO: Implementar cuando tengamos TokenBlacklistRepository
func (j *CleanupJobs) CleanTokenBlacklist() {
	j.logger.Debug("token blacklist cleanup skipped - not implemented yet")
}

// CleanOldAnalytics limpia datos analíticos antiguos
// Ejecuta cada día a las 3:00 AM
// TODO: Implementar cuando tengamos método DeleteOlderThan en AnalyticsRepository
func (j *CleanupJobs) CleanOldAnalytics() {
	j.logger.Debug("old analytics cleanup skipped - not implemented yet")
}

// DeactivateExpiredAds desactiva anuncios cuya campaña ha expirado
// Ejecuta cada día a las 2:00 AM
// TODO: Implementar método DeactivateExpired en AdvertisementRepository
func (j *CleanupJobs) DeactivateExpiredAds() {
	j.logger.Debug("expired ads deactivation skipped - not implemented yet")
}
