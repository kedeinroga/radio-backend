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
	sharedSecretKey string,
	logger *slog.Logger,
) {
	streamGroup := router.Group("/api/v1/stream")
	{
		// Proxy endpoint — sin JWT, validación por token en query param
		streamGroup.GET("/proxy", proxyHandler.ProxyStream)

		// /start: SharedSecret requerido, JWT opcional (soporta guests)
		startGroup := streamGroup.Group("")
		startGroup.Use(middleware.SharedSecretAuth(sharedSecretKey))
		startGroup.Use(authMiddleware.Optional())
		{
			startGroup.POST("/start", sessionHandler.StartSession)
		}

		// Resto de endpoints: JWT requerido
		authenticated := streamGroup.Group("")
		authenticated.Use(authMiddleware.Required())
		{
			authenticated.POST("/heartbeat", sessionHandler.Heartbeat)
			authenticated.POST("/stop", sessionHandler.StopSession)
			authenticated.GET("/sessions", sessionHandler.GetActiveSessions)
			authenticated.GET("/stats", proxyHandler.GetStreamStats)
		}
	}

	logger.Info("stream routes registered",
		"endpoints", []string{
			"GET  /api/v1/stream/proxy       (no auth)",
			"POST /api/v1/stream/start       (shared secret, JWT optional)",
			"POST /api/v1/stream/heartbeat   (JWT required)",
			"POST /api/v1/stream/stop        (JWT required)",
			"GET  /api/v1/stream/sessions    (JWT required)",
			"GET  /api/v1/stream/stats       (JWT required)",
		},
	)
}
