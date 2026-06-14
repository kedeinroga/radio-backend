package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"radio-backend/internal/config"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/jobs"
	"radio-backend/internal/services"
)

// InitializeJobSystem inicializa el sistema de trabajos en segundo plano
func InitializeJobSystem(
	db *database.Connection,
	streamSessionService *services.StreamSessionService,
	stationService *services.StationService,
	nowPlayingService *services.NowPlayingService,
	nowPlayingCfg config.NowPlayingConfig,
	logger *slog.Logger,
) (*jobs.JobScheduler, error) {

	logger.Info("initializing job system...")

	// Crear scheduler
	scheduler := jobs.NewJobScheduler(logger)

	// Crear cleanup jobs
	cleanupJobs := jobs.NewCleanupJobs(
		streamSessionService,
		nil, // adRepo - usaremos nil por ahora
		logger,
	)

	// Crear analytics jobs
	analyticsJobs := jobs.NewAnalyticsJobs(db, logger)

	// Crear health jobs
	healthJobs := jobs.NewHealthJobs(db, logger)

	// Registrar cleanup jobs
	// Cada 5 minutos: limpiar sesiones abandonadas
	if err := scheduler.AddJob("cleanup_sessions", "0 */5 * * * *", cleanupJobs.CleanAbandonedSessions); err != nil {
		logger.Error("failed to register cleanup_sessions job", "error", err)
	}

	// Cada 1 hora: limpiar token blacklist
	if err := scheduler.AddJob("cleanup_tokens", "0 0 * * * *", cleanupJobs.CleanTokenBlacklist); err != nil {
		logger.Error("failed to register cleanup_tokens job", "error", err)
	}

	// Cada día a las 3 AM: limpiar analytics antiguos
	if err := scheduler.AddJob("cleanup_analytics", "0 0 3 * * *", cleanupJobs.CleanOldAnalytics); err != nil {
		logger.Error("failed to register cleanup_analytics job", "error", err)
	}

	// Cada día a las 2 AM: desactivar anuncios expirados
	if err := scheduler.AddJob("deactivate_ads", "0 0 2 * * *", cleanupJobs.DeactivateExpiredAds); err != nil {
		logger.Error("failed to register deactivate_ads job", "error", err)
	}

	// Registrar analytics jobs
	// Cada 15 minutos: refrescar vistas materializadas
	if err := scheduler.AddJob("refresh_mv", "0 */15 * * * *", analyticsJobs.RefreshMaterializedViews); err != nil {
		logger.Error("failed to register refresh_mv job", "error", err)
	}

	// Cada día a las 1 AM: generar resumen diario
	if err := scheduler.AddJob("daily_summary", "0 0 1 * * *", analyticsJobs.GenerateDailySummary); err != nil {
		logger.Error("failed to register daily_summary job", "error", err)
	}

	// Cada lunes a las 6 AM: generar reporte semanal
	if err := scheduler.AddJob("weekly_report", "0 0 6 * * 1", analyticsJobs.GenerateWeeklyReport); err != nil {
		logger.Error("failed to register weekly_report job", "error", err)
	}

	// Registrar health jobs
	// Cada 1 minuto: verificar salud de base de datos
	if err := scheduler.AddJob("db_health", "0 * * * * *", healthJobs.CheckDatabaseHealth); err != nil {
		logger.Error("failed to register db_health job", "error", err)
	}

	// Cada 5 minutos: recolectar métricas del sistema
	if err := scheduler.AddJob("system_metrics", "0 */5 * * * *", healthJobs.CollectSystemMetrics); err != nil {
		logger.Error("failed to register system_metrics job", "error", err)
	}

	// Cada 5 minutos: verificar salud de APIs
	if err := scheduler.AddJob("api_health", "0 */5 * * * *", healthJobs.CheckAPIHealth); err != nil {
		logger.Error("failed to register api_health job", "error", err)
	}

	// Cada 2 minutos: monitorear streams activos
	if err := scheduler.AddJob("monitor_streams", "0 */2 * * * *", healthJobs.MonitorStreams); err != nil {
		logger.Error("failed to register monitor_streams job", "error", err)
	}

	// Registrar now-playing jobs (captura de metadata ICY)
	if nowPlayingCfg.Enabled && nowPlayingService != nil && stationService != nil {
		nowPlayingJobs := jobs.NewNowPlayingJobs(
			nowPlayingService,
			stationService,
			nowPlayingCfg.TopStations,
			nowPlayingCfg.MaxConcurrency,
			nowPlayingCfg.RetentionDays,
			logger,
		)

		// Sondeo de las top-N estaciones según el intervalo configurado (def. 5 min)
		pollSchedule := fmt.Sprintf("0 */%d * * * *", intervalMinutes(nowPlayingCfg.PollInterval))
		if err := scheduler.AddJob("now_playing_poll", pollSchedule, nowPlayingJobs.PollPopularStations); err != nil {
			logger.Error("failed to register now_playing_poll job", "error", err)
		}

		// Cada día a las 4 AM: retención del historial de pistas
		if err := scheduler.AddJob("now_playing_cleanup", "0 0 4 * * *", nowPlayingJobs.CleanupOldTracks); err != nil {
			logger.Error("failed to register now_playing_cleanup job", "error", err)
		}
	} else {
		logger.Info("now-playing jobs disabled or not configured")
	}

	logger.Info("job system initialized",
		"jobs_registered", len(scheduler.GetJobStats()),
	)

	return scheduler, nil
}

// intervalMinutes convierte una duración a minutos enteros para el cron schedule,
// con un mínimo de 1 minuto.
func intervalMinutes(d time.Duration) int {
	minutes := int(d.Minutes())
	if minutes < 1 {
		return 1
	}
	return minutes
}

// RegisterJobRoutes registra los endpoints de administración de jobs
func RegisterJobRoutes(router *gin.Engine, scheduler *jobs.JobScheduler) {
	api := router.Group("/api/v1/admin/jobs")
	{
		// GET /api/v1/admin/jobs - Listar todos los jobs y sus stats
		api.GET("", func(c *gin.Context) {
			stats := scheduler.GetJobStats()
			c.JSON(200, gin.H{
				"jobs":  stats,
				"count": len(stats),
			})
		})

		// GET /api/v1/admin/jobs/:name - Obtener stats de un job específico
		api.GET("/:name", func(c *gin.Context) {
			name := c.Param("name")
			stat, exists := scheduler.GetJobStat(name)
			if !exists {
				c.JSON(404, gin.H{"error": "job not found"})
				return
			}
			c.JSON(200, stat)
		})
	}
}
