package handlers

import (
	"net/http"

	"radio-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// GuestRateLimitHandler exposes admin endpoints to inspect and toggle the per-guest-IP rate limiter.
type GuestRateLimitHandler struct {
	limiter *middleware.GuestIPRateLimiter
}

// NewGuestRateLimitHandler creates a new GuestRateLimitHandler.
func NewGuestRateLimitHandler(limiter *middleware.GuestIPRateLimiter) *GuestRateLimitHandler {
	return &GuestRateLimitHandler{limiter: limiter}
}

// GuestRateLimitData holds the guest IP rate limiter state.
type GuestRateLimitData struct {
	Enabled      bool `json:"enabled" example:"false"`
	LimitPerHour int  `json:"limit_per_hour" example:"100"`
}

// GuestRateLimitResponse is the {"success": true, "data": {...}} envelope.
type GuestRateLimitResponse struct {
	Success bool               `json:"success" example:"true"`
	Data    GuestRateLimitData `json:"data"`
}

// GetStatus returns the current state of the guest IP rate limiter.
// @Summary Get guest IP rate limit status
// @Description Returns whether the per-guest-IP rate limiter is currently enabled and the configured limit.
// @Tags Admin Security
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GuestRateLimitResponse "Current status"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Router /api/v1/admin/security/guest-rate-limit [get]
func (h *GuestRateLimitHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":        h.limiter.IsEnabled(),
			"limit_per_hour": middleware.GuestIPRateLimit,
		},
	})
}

// Toggle enables or disables the guest IP rate limiter.
// @Summary Toggle guest IP rate limiter
// @Description Enables the limiter if it is currently disabled, and vice versa. Returns the new state.
// @Tags Admin Security
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GuestRateLimitResponse "New state"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Failed to toggle guest rate limiter"
// @Router /api/v1/admin/security/guest-rate-limit/toggle [post]
func (h *GuestRateLimitHandler) Toggle(c *gin.Context) {
	enabled, err := h.limiter.Toggle()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "toggle_failed", "Failed to toggle guest rate limiter")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":        enabled,
			"limit_per_hour": middleware.GuestIPRateLimit,
		},
	})
}
