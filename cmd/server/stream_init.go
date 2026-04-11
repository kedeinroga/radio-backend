package main

import (
	"log/slog"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/handlers"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/middleware"
	"radio-backend/internal/repositories/postgres"
	"radio-backend/internal/server"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// InitializeStreamingSystem inicializa el sistema de streaming de audio
func InitializeStreamingSystem(
	router *gin.Engine,
	db *database.Connection,
	stationRepo domain.StationRepository,
	adRepo domain.AdvertisementRepository,
	impressionRepo domain.AdImpressionRepository,
	authMiddleware *middleware.AuthMiddleware,
	analyticsService *services.AnalyticsService,
	sharedSecretKey string,
	logger *slog.Logger,
	jwtSecret []byte,
) error {
	logger.Info("initializing streaming system...")

	if len(jwtSecret) == 0 {
		logger.Warn("JWT secret not configured — authenticated stream sessions will fail, guest streaming still works")
	}

	// Crear repositorio de sesiones de streaming
	sessionRepo := postgres.NewStreamSessionRepository(db)
	logger.Info("stream session repository created")

	// Crear StreamSessionService
	sessionService := services.NewStreamSessionService(
		sessionRepo,
		stationRepo,
		adRepo,
		impressionRepo,
		jwtSecret,
		5*time.Minute, // Token duration: 5 minutos
		logger,
	)
	logger.Info("stream session service created")

	// Crear AudioProxyService
	proxyService := services.NewAudioProxyService(
		sessionService,
		logger,
	)
	logger.Info("audio proxy service created")

	// Crear handlers
	sessionHandler := handlers.NewStreamSessionHandler(sessionService, analyticsService, logger)
	proxyHandler := handlers.NewAudioProxyHandler(proxyService, logger)
	logger.Info("stream handlers created")

	// Registrar rutas
	server.RegisterStreamRoutes(
		router,
		sessionHandler,
		proxyHandler,
		authMiddleware,
		sharedSecretKey,
		logger,
	)

	logger.Info("streaming system initialized successfully")
	return nil
}
