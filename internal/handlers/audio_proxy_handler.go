package handlers

import (
	"log/slog"
	"net/http"

	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AudioProxyHandler maneja el proxy de streams de audio
type AudioProxyHandler struct {
	service *services.AudioProxyService
	logger  *slog.Logger
}

// NewAudioProxyHandler crea una nueva instancia del handler
func NewAudioProxyHandler(
	service *services.AudioProxyService,
	logger *slog.Logger,
) *AudioProxyHandler {
	return &AudioProxyHandler{
		service: service,
		logger:  logger,
	}
}

// ProxyStream maneja GET /api/v1/stream/proxy?token=JWT
// @Summary Proxy de stream de audio
// @Description Hace proxy del stream de audio de una estación validando el token de sesión
// @Tags Streaming
// @Produce audio/mpeg
// @Param token query string true "Token JWT de la sesión"
// @Success 200 {file} audio/mpeg
// @Failure 401 {object} ErrorResponse "Token inválido o expirado"
// @Failure 404 {object} ErrorResponse "Estación no encontrada"
// @Failure 502 {object} ErrorResponse "Error al conectar con el stream"
// @Router /stream/proxy [get]
func (h *AudioProxyHandler) ProxyStream(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		h.logger.Warn("stream proxy request without token", "ip", c.ClientIP())
		RespondWithError(c, http.StatusUnauthorized, "MISSING_TOKEN", "Stream token is required")
		return
	}

	h.logger.Info("stream proxy request",
		"ip", c.ClientIP(),
		"user_agent", c.Request.UserAgent(),
	)

	// ProxyStream maneja todo: validación, conexión, streaming
	// Los errores ya están manejados dentro del servicio
	err := h.service.ProxyStream(
		c.Request.Context(),
		token,
		c.Writer,
		c.Request,
	)

	if err != nil {
		// Si hay error pero no se envió respuesta aún, enviar error
		// (esto solo ocurre si el error fue antes de comenzar el streaming)
		if !c.Writer.Written() {
			h.logger.Error("stream proxy error", "error", err)
		}
		// Si ya comenzó el streaming, solo logear (no se puede enviar JSON)
	}
}

// GetStreamStats maneja GET /api/v1/stream/stats
// @Summary Obtener estadísticas de streams activos
// @Description Retorna información sobre streams actualmente activos (solo admin)
// @Tags Streaming
// @Produce json
// @Success 200 {object} StreamStatsResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /stream/stats [get]
func (h *AudioProxyHandler) GetStreamStats(c *gin.Context) {
	activeCount := h.service.GetActiveStreamCount()
	streams := h.service.GetActiveStreams()

	c.JSON(http.StatusOK, StreamStatsResponse{
		ActiveStreams: activeCount,
		Streams:       streams,
	})
}

// StreamStatsResponse representa estadísticas de streaming
type StreamStatsResponse struct {
	ActiveStreams int                   `json:"active_streams"`
	Streams       []services.StreamInfo `json:"streams"`
}
