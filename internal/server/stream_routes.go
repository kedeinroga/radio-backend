package server

import (
	"log/slog"

	"radio-backend/internal/handlers"
	"radio-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterStreamRoutes registra las rutas de streaming
func RegisterStreamRoutes(
	router *gin.Engine,
	sessionHandler *handlers.StreamSessionHandler,
	proxyHandler *handlers.AudioProxyHandler,
	authMiddleware *middleware.AuthMiddleware,
	logger *slog.Logger,
) {
	streamGroup := router.Group("/api/v1/stream")
	{
		// Proxy endpoint - sin autenticación JWT, validación por token en query
		streamGroup.GET("/proxy", proxyHandler.ProxyStream)

		// Endpoints autenticados
		authenticated := streamGroup.Group("")
		authenticated.Use(authMiddleware.Required())
		{
			// Gestión de sesiones
			authenticated.POST("/start", sessionHandler.StartSession)
			authenticated.POST("/heartbeat", sessionHandler.Heartbeat)
			authenticated.POST("/stop", sessionHandler.StopSession)
			authenticated.GET("/sessions", sessionHandler.GetActiveSessions)

			// Estadísticas (solo para debugging/admin)
			authenticated.GET("/stats", proxyHandler.GetStreamStats)
		}
	}

	logger.Info("stream routes registered",
		"endpoints", []string{
			"GET /api/v1/stream/proxy",
			"POST /api/v1/stream/start",
			"POST /api/v1/stream/heartbeat",
			"POST /api/v1/stream/stop",
			"GET /api/v1/stream/sessions",
			"GET /api/v1/stream/stats",
		},
	)
}
