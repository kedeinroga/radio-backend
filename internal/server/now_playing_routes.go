package server

import (
	"log/slog"

	"radio-backend/internal/handlers"
	"radio-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterNowPlayingRoutes registra las rutas de "sonando ahora" / "sonó recientemente".
// Comparten el mismo grupo y protección (X-Rradio-Secret) que el resto de endpoints de estación.
func RegisterNowPlayingRoutes(
	router *gin.Engine,
	nowPlayingHandler *handlers.NowPlayingHandler,
	sharedSecretKey string,
	logger *slog.Logger,
) {
	group := router.Group("/api/v1/stations")
	group.Use(middleware.SharedSecretAuth(sharedSecretKey))
	{
		group.GET("/:id/now-playing", nowPlayingHandler.GetNowPlaying)
		group.GET("/:id/recent-tracks", nowPlayingHandler.GetRecentTracks)
	}

	logger.Info("now-playing routes registered",
		"endpoints", []string{
			"GET /api/v1/stations/:id/now-playing   (shared secret)",
			"GET /api/v1/stations/:id/recent-tracks (shared secret)",
		},
	)
}
