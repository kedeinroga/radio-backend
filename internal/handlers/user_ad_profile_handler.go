package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"radio-backend/internal/domain"
	"radio-backend/internal/services"
)

// User ad-profile endpoints emit flat {"error": "message"} bodies on failure
// (SimpleErrorResponse) and the success envelopes defined below.

// PremiumActivatedResponse is returned after activating premium.
type PremiumActivatedResponse struct {
	Success   bool      `json:"success" example:"true"`
	Message   string    `json:"message" example:"Premium subscription activated"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PremiumDeactivatedResponse is returned after deactivating premium.
type PremiumDeactivatedResponse struct {
	Success bool      `json:"success" example:"true"`
	Message string    `json:"message" example:"Premium subscription deactivated"`
	UserID  uuid.UUID `json:"user_id"`
}

// CanShowAdResponse is the ad-eligibility check result.
type CanShowAdResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	CanShow bool      `json:"can_show" example:"true"`
	Reason  string    `json:"reason" example:"allowed"`
}

// UserAdProfileHandler handles user ad profile management
type UserAdProfileHandler struct {
	profileService *services.UserAdProfileService
	logger         *slog.Logger
}

// NewUserAdProfileHandler creates a new user ad profile handler
func NewUserAdProfileHandler(
	profileService *services.UserAdProfileService,
	logger *slog.Logger,
) *UserAdProfileHandler {
	return &UserAdProfileHandler{
		profileService: profileService,
		logger:         logger,
	}
}

// UpdateProfileRequest represents request to update user preferences
type UpdateProfileRequest struct {
	PreferredGenres []string `json:"preferred_genres,omitempty"`
}

// ActivatePremiumRequest represents request to activate premium subscription
type ActivatePremiumRequest struct {
	StripeCustomerID     string    `json:"stripe_customer_id" binding:"required"`
	StripeSubscriptionID string    `json:"stripe_subscription_id" binding:"required"`
	ExpiresAt            time.Time `json:"expires_at" binding:"required"`
}

// GetUserAdProfile retrieves a user's ad profile
// @Summary Get user advertising profile
// @Description Retrieves the user's advertising profile including premium status, ad interaction statistics, and preferences.
// @Tags User Ad Profile
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} domain.UserAdProfile "User advertising profile"
// @Failure 400 {object} SimpleErrorResponse "Invalid user ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} SimpleErrorResponse "Internal server error"
// @Router /api/v1/users/{user_id}/ad-profile [get]
func (h *UserAdProfileHandler) GetUserAdProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var profile *domain.UserAdProfile
	profile, err = h.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user ad profile", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateUserAdProfile updates user ad preferences
// @Summary Update user ad preferences
// @Description Updates user's advertising preferences such as preferred genres for better ad targeting.
// @Tags User Ad Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param profile body UpdateProfileRequest true "Profile updates"
// @Success 200 {object} domain.UserAdProfile "Updated profile"
// @Failure 400 {object} SimpleErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} SimpleErrorResponse "Internal server error"
// @Router /api/v1/users/{user_id}/ad-profile [put]
func (h *UserAdProfileHandler) UpdateUserAdProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid profile update request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing profile
	profile, err := h.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user ad profile", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	// Update fields
	if req.PreferredGenres != nil {
		profile.PreferredGenres = req.PreferredGenres
	}
	profile.UpdatedAt = time.Now()

	// Save updated profile
	if err := h.profileService.UpdateProfile(c.Request.Context(), profile); err != nil {
		h.logger.Error("Failed to update user ad profile", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ActivatePremium activates premium subscription for a user
// @Summary Activate premium subscription
// @Description Activates premium subscription for a user after successful Stripe payment. Call this endpoint from your Stripe webhook handler.
// @Tags User Ad Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param premium body ActivatePremiumRequest true "Premium subscription details from Stripe"
// @Success 200 {object} PremiumActivatedResponse "Profile with premium activated"
// @Failure 400 {object} SimpleErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} SimpleErrorResponse "Internal server error"
// @Router /api/v1/users/{user_id}/ad-profile/activate-premium [post]
func (h *UserAdProfileHandler) ActivatePremium(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req ActivatePremiumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid premium activation request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.profileService.ActivatePremium(
		c.Request.Context(),
		userID,
		services.PremiumSubscriptionParams{
			StripeCustomerID:     req.StripeCustomerID,
			StripeSubscriptionID: req.StripeSubscriptionID,
			SubscriptionStatus:   "active",
			ExpiresAt:            &req.ExpiresAt,
		},
	); err != nil {
		h.logger.Error("Failed to activate premium", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate premium"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Premium subscription activated",
		"user_id":    userID,
		"expires_at": req.ExpiresAt,
	})
}

// DeactivatePremium deactivates premium subscription for a user
// @Summary Deactivate premium subscription
// @Description Deactivates premium subscription for a user (e.g., after cancellation or expiration). User will start seeing ads again.
// @Tags User Ad Profile
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} PremiumDeactivatedResponse "Premium deactivated"
// @Failure 400 {object} SimpleErrorResponse "Invalid user ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} SimpleErrorResponse "Internal server error"
// @Router /api/v1/users/{user_id}/ad-profile/deactivate-premium [post]
func (h *UserAdProfileHandler) DeactivatePremium(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.profileService.DeactivatePremium(c.Request.Context(), userID); err != nil {
		h.logger.Error("Failed to deactivate premium", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate premium"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium subscription deactivated",
		"user_id": userID,
	})
}

// GetUserAdStats retrieves ad statistics for a user
// @Summary Get user ad interaction statistics
// @Description Retrieves statistics about user's ad interactions including total impressions, clicks, and CTR.
// @Tags User Ad Profile
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} services.UserAdStats "User ad statistics"
// @Failure 400 {object} SimpleErrorResponse "Invalid user ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} SimpleErrorResponse "Internal server error"
// @Router /api/v1/users/{user_id}/ad-profile/stats [get]
func (h *UserAdProfileHandler) GetUserAdStats(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	stats, err := h.profileService.GetProfileStats(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user ad stats", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CanShowAd checks if an ad can be shown to a user (frequency capping)
// @Summary Check if user can see ads
// @Description Checks if a user is eligible to see advertisements based on premium status and frequency capping rules. Returns eligibility status and reason if blocked.
// @Tags User Ad Profile
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} CanShowAdResponse "Ad eligibility result"
// @Failure 400 {object} SimpleErrorResponse "Invalid user ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} SimpleErrorResponse "Internal server error"
// @Router /api/v1/users/{user_id}/ad-profile/can-show-ad [get]
func (h *UserAdProfileHandler) CanShowAd(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	canShow, err := h.profileService.CanShowAd(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to check if ad can be shown", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check ad eligibility"})
		return
	}

	reason := "allowed"
	if !canShow {
		reason = "frequency_cap_exceeded_or_premium"
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"can_show": canShow,
		"reason":   reason,
	})
}
