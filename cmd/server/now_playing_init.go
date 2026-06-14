package main

import (
	"log/slog"
	"time"

	"radio-backend/internal/handlers"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/repositories/postgres"
	"radio-backend/internal/server"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// InitializeNowPlayingSystem inicializa el sistema de "sonando ahora":
// repositorio, servicio, handler y rutas. Retorna el NowPlayingService para
// que el job de sondeo pueda capturar metadata periódicamente.
func InitializeNowPlayingSystem(
	router *gin.Engine,
	db *database.Connection,
	sharedSecretKey string,
	fetchTimeout time.Duration,
	logger *slog.Logger,
) *services.NowPlayingService {
	logger.Info("initializing now-playing system...")

	trackRepo := postgres.NewStationTrackRepository(db)
	nowPlayingService := services.NewNowPlayingService(trackRepo, fetchTimeout, logger)
	nowPlayingHandler := handlers.NewNowPlayingHandler(nowPlayingService)

	server.RegisterNowPlayingRoutes(router, nowPlayingHandler, sharedSecretKey, logger)

	logger.Info("now-playing system initialized successfully")
	return nowPlayingService
}
