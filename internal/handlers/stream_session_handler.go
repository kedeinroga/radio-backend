package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StreamSessionHandler maneja los endpoints de sesiones de streaming
type StreamSessionHandler struct {
	service *services.StreamSessionService
	logger  *slog.Logger
}

// NewStreamSessionHandler crea una nueva instancia del handler
func NewStreamSessionHandler(
	service *services.StreamSessionService,
	logger *slog.Logger,
) *StreamSessionHandler {
	return &StreamSessionHandler{
		service: service,
		logger:  logger,
	}
}

// StartSessionRequest representa la solicitud para iniciar una sesión
type StartSessionRequest struct {
	StationID string     `json:"station_id" binding:"required"`
	AdID      *uuid.UUID `json:"ad_id"` // Opcional
}

// StartSessionResponse representa la respuesta al iniciar una sesión
type StartSessionResponse struct {
	SessionID uuid.UUID `json:"session_id"`
	StreamURL string    `json:"stream_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// HeartbeatRequest representa la solicitud de heartbeat
type HeartbeatRequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
}

// StopSessionRequest representa la solicitud para detener una sesión
type StopSessionRequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
}

// ActiveSessionsResponse representa la respuesta de sesiones activas
type ActiveSessionsResponse struct {
	Sessions []ActiveSessionInfo `json:"sessions"`
	Count    int                 `json:"count"`
}

// ActiveSessionInfo representa información de una sesión activa
type ActiveSessionInfo struct {
	SessionID         uuid.UUID  `json:"session_id"`
	StationID         string     `json:"station_id"`
	StartedAt         time.Time  `json:"started_at"`
	LastHeartbeat     time.Time  `json:"last_heartbeat"`
	BytesStreamed     int64      `json:"bytes_streamed"`
	ListeningDuration string     `json:"listening_duration"`
	AdID              *uuid.UUID `json:"ad_id,omitempty"`
}

// StartSession maneja POST /api/v1/stream/start
// @Summary Iniciar sesión de streaming
// @Description Crea una nueva sesión de streaming y retorna un token para acceder al proxy
// @Tags Streaming
// @Accept json
// @Produce json
// @Param request body StartSessionRequest true "Datos de la sesión"
// @Success 200 {object} StartSessionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /stream/start [post]
func (h *StreamSessionHandler) StartSession(c *gin.Context) {
	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Obtener user_id del contexto (establecido por AuthMiddleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User ID not found in context")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "INVALID_USER_ID", err.Error())
		return
	}

	// Iniciar sesión
	session, streamURL, err := h.service.StartSession(
		c.Request.Context(),
		userID,
		req.StationID,
		req.AdID,
		c.Request.UserAgent(),
		c.ClientIP(),
	)

	if err != nil {
		h.logger.Error("failed to start session",
			"error", err,
			"user_id", userID,
			"station_id", req.StationID,
		)
		RespondWithError(c, http.StatusInternalServerError, "SESSION_START_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, StartSessionResponse{
		SessionID: session.SessionID,
		StreamURL: streamURL,
		ExpiresAt: session.TokenExpiresAt,
	})
}

// Heartbeat maneja POST /api/v1/stream/heartbeat
// @Summary Enviar heartbeat de sesión
// @Description Actualiza el timestamp de última actividad de una sesión
// @Tags Streaming
// @Accept json
// @Produce json
// @Param request body HeartbeatRequest true "Session ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /stream/heartbeat [post]
func (h *StreamSessionHandler) Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	err := h.service.Heartbeat(c.Request.Context(), req.SessionID)
	if err != nil {
		h.logger.Error("failed to update heartbeat",
			"error", err,
			"session_id", req.SessionID,
		)
		RespondWithError(c, http.StatusInternalServerError, "HEARTBEAT_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Heartbeat updated",
	})
}

// StopSession maneja POST /api/v1/stream/stop
// @Summary Detener sesión de streaming
// @Description Finaliza una sesión de streaming y valida impresiones de anuncios si aplica
// @Tags Streaming
// @Accept json
// @Produce json
// @Param request body StopSessionRequest true "Session ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /stream/stop [post]
func (h *StreamSessionHandler) StopSession(c *gin.Context) {
	var req StopSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	err := h.service.EndSession(c.Request.Context(), req.SessionID)
	if err != nil {
		h.logger.Error("failed to stop session",
			"error", err,
			"session_id", req.SessionID,
		)
		RespondWithError(c, http.StatusInternalServerError, "STOP_SESSION_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session stopped successfully",
	})
}

// GetActiveSessions maneja GET /api/v1/stream/sessions
// @Summary Obtener sesiones activas del usuario
// @Description Retorna todas las sesiones de streaming activas del usuario autenticado
// @Tags Streaming
// @Produce json
// @Success 200 {object} ActiveSessionsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /stream/sessions [get]
func (h *StreamSessionHandler) GetActiveSessions(c *gin.Context) {
	// Obtener user_id del contexto
	userIDStr, exists := c.Get("user_id")
	if !exists {
		RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User ID not found in context")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "INVALID_USER_ID", err.Error())
		return
	}

	sessions, err := h.service.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get active sessions",
			"error", err,
			"user_id", userID,
		)
		RespondWithError(c, http.StatusInternalServerError, "GET_SESSIONS_FAILED", err.Error())
		return
	}

	// Convertir a DTOs
	sessionInfos := make([]ActiveSessionInfo, len(sessions))
	for i, session := range sessions {
		sessionInfos[i] = ActiveSessionInfo{
			SessionID:         session.SessionID,
			StationID:         session.StationID,
			StartedAt:         session.StartedAt,
			LastHeartbeat:     session.LastHeartbeat,
			BytesStreamed:     session.BytesStreamed,
			ListeningDuration: session.ListeningDuration.String(),
			AdID:              session.AdID,
		}
	}

	c.JSON(http.StatusOK, ActiveSessionsResponse{
		Sessions: sessionInfos,
		Count:    len(sessionInfos),
	})
}
