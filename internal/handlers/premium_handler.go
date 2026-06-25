package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PremiumHandler maneja endpoints de suscripciones premium
type PremiumHandler struct {
	premiumService *services.PremiumService
	logger         *slog.Logger
}

// NewPremiumHandler crea una nueva instancia del handler
func NewPremiumHandler(
	premiumService *services.PremiumService,
	logger *slog.Logger,
) *PremiumHandler {
	return &PremiumHandler{
		premiumService: premiumService,
		logger:         logger,
	}
}

// CreateCheckoutSession crea una sesión de checkout de Stripe
// @Summary Create checkout session
// @Tags premium
// @Security BearerAuth
// @Param request body CreateCheckoutSessionRequest true "Checkout request"
// @Success 200 {object} CreateCheckoutSessionResponse
// @Failure 400 {object} SimpleErrorResponse
// @Failure 401 {object} SimpleErrorResponse
// @Failure 500 {object} SimpleErrorResponse
// @Router /api/v1/premium/checkout [post]
func (h *PremiumHandler) CreateCheckoutSession(c *gin.Context) {
	// Obtener user_id del contexto (agregado por AuthMiddleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Parse request
	var req CreateCheckoutSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validar plan
	if req.Plan != "monthly" && req.Plan != "yearly" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan must be 'monthly' or 'yearly'"})
		return
	}

	// Crear sesión de checkout
	session, err := h.premiumService.CreateCheckoutSession(userID, req.Plan, req.Email)
	if err != nil {
		h.logger.Error("failed to create checkout session",
			"error", err,
			"user_id", userID,
			"plan", req.Plan,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}

	h.logger.Info("checkout session created",
		"user_id", userID,
		"session_id", session.ID,
		"plan", req.Plan,
	)

	c.JSON(http.StatusOK, CreateCheckoutSessionResponse{
		SessionID:  session.ID,
		SessionURL: session.URL,
	})
}

// GetPremiumStatus obtiene el estado premium del usuario
// @Summary Get premium status
// @Tags premium
// @Security BearerAuth
// @Success 200 {object} PremiumStatusResponse
// @Failure 400 {object} SimpleErrorResponse "Invalid user ID format"
// @Failure 401 {object} SimpleErrorResponse
// @Failure 500 {object} SimpleErrorResponse
// @Router /api/v1/premium/status [get]
func (h *PremiumHandler) GetPremiumStatus(c *gin.Context) {
	// Obtener user_id del contexto
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Buscar perfil del usuario
	profile, err := h.premiumService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user profile",
			"error", err,
			"user_id", userID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, PremiumStatusResponse{
		IsPremium:        profile.IsPremium,
		PremiumExpiresAt: formatTime(profile.PremiumExpiresAt),
		StripeCustomerID: profile.StripeCustomerID,
	})
}

// CancelSubscription cancela la suscripción premium
// @Summary Cancel subscription
// @Tags premium
// @Security BearerAuth
// @Success 200 {object} SuccessMessageResponse "Subscription cancelled"
// @Failure 400 {object} SimpleErrorResponse "Invalid user ID format"
// @Failure 401 {object} SimpleErrorResponse
// @Failure 500 {object} SimpleErrorResponse
// @Router /api/v1/premium/cancel [post]
func (h *PremiumHandler) CancelSubscription(c *gin.Context) {
	// Obtener user_id del contexto
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Desactivar premium
	if err := h.premiumService.DeactivatePremium(c.Request.Context(), userID); err != nil {
		h.logger.Error("failed to cancel subscription",
			"error", err,
			"user_id", userID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel subscription"})
		return
	}

	h.logger.Info("subscription cancelled",
		"user_id", userID,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "subscription cancelled successfully",
		"success": true,
	})
}

// GetCustomerPortalURL obtiene la URL del portal del cliente de Stripe
// @Summary Get customer portal URL
// @Tags premium
// @Security BearerAuth
// @Success 200 {object} CustomerPortalResponse
// @Failure 400 {object} SimpleErrorResponse "Invalid user ID format"
// @Failure 401 {object} SimpleErrorResponse
// @Failure 500 {object} SimpleErrorResponse
// @Router /api/v1/premium/portal [get]
func (h *PremiumHandler) GetCustomerPortalURL(c *gin.Context) {
	// Obtener user_id del contexto
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Obtener URL del portal
	portalURL, err := h.premiumService.GetCustomerPortalURL(userID)
	if err != nil {
		h.logger.Error("failed to get portal URL",
			"error", err,
			"user_id", userID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get portal URL"})
		return
	}

	c.JSON(http.StatusOK, CustomerPortalResponse{
		PortalURL: portalURL,
	})
}

// Request/Response DTOs

type CreateCheckoutSessionRequest struct {
	Plan  string `json:"plan" binding:"required"` // "monthly" or "yearly"
	Email string `json:"email" binding:"required,email"`
}

type CreateCheckoutSessionResponse struct {
	SessionID  string `json:"session_id"`
	SessionURL string `json:"session_url"`
}

type PremiumStatusResponse struct {
	IsPremium        bool    `json:"is_premium"`
	PremiumExpiresAt *string `json:"premium_expires_at,omitempty"`
	StripeCustomerID *string `json:"stripe_customer_id,omitempty"`
}

type CustomerPortalResponse struct {
	PortalURL string `json:"portal_url"`
}

// Helper function to format time pointer to string pointer
func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}
