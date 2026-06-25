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

// AdvertisementHandler handles HTTP requests for advertisements
type AdvertisementHandler struct {
	adService *services.AdvertisementService
	logger    *slog.Logger
}

// NewAdvertisementHandler creates a new advertisement handler
func NewAdvertisementHandler(
	adService *services.AdvertisementService,
	logger *slog.Logger,
) *AdvertisementHandler {
	return &AdvertisementHandler{
		adService: adService,
		logger:    logger,
	}
}

// EligibleAdsResponse is the {"ads": [...], "count": n} envelope returned by GetEligibleAds.
type EligibleAdsResponse struct {
	Ads   []domain.Advertisement `json:"ads"`
	Count int                    `json:"count" example:"3"`
}

// CreateAdvertisementRequest represents the request body for creating an advertisement
type CreateAdvertisementRequest struct {
	CampaignID            string    `json:"campaign_id" binding:"required,uuid"`
	Title                 string    `json:"title" binding:"required,min=3,max=200"`
	Description           *string   `json:"description,omitempty"`
	AdvertiserName        string    `json:"advertiser_name" binding:"required,min=2,max=100"`
	MediaURL              string    `json:"media_url" binding:"required,url"`
	AdFormat              string    `json:"ad_format" binding:"required,oneof=banner interstitial audio native"`
	AdType                string    `json:"ad_type" binding:"required,oneof=image video audio html"`
	ClickURL              string    `json:"click_url" binding:"required,url"`
	TotalBudgetCents      int       `json:"total_budget_cents" binding:"required,min=1"`
	CPMRateCents          *int      `json:"cpm_rate_cents,omitempty"`
	CPCRateCents          *int      `json:"cpc_rate_cents,omitempty"`
	StartDate             time.Time `json:"start_date" binding:"required"`
	EndDate               time.Time `json:"end_date" binding:"required"`
	TargetCountries       []string  `json:"target_countries,omitempty"`
	TargetGenres          []string  `json:"target_genres,omitempty"`
	TargetLanguages       []string  `json:"target_languages,omitempty"`
	TargetDevices         []string  `json:"target_devices,omitempty"`
	MaxImpressionsPerUser *int      `json:"max_impressions_per_user,omitempty"`
	MaxImpressionsPerDay  *int      `json:"max_impressions_per_day,omitempty"`
}

// UpdateAdvertisementRequest represents the request body for updating an advertisement
type UpdateAdvertisementRequest struct {
	Title                 *string  `json:"title,omitempty" binding:"omitempty,min=3,max=200"`
	Description           *string  `json:"description,omitempty"`
	MediaURL              *string  `json:"media_url,omitempty" binding:"omitempty,url"`
	ClickURL              *string  `json:"click_url,omitempty" binding:"omitempty,url"`
	TotalBudgetCents      *int     `json:"total_budget_cents,omitempty" binding:"omitempty,min=1"`
	CPMRateCents          *int     `json:"cpm_rate_cents,omitempty"`
	CPCRateCents          *int     `json:"cpc_rate_cents,omitempty"`
	TargetCountries       []string `json:"target_countries,omitempty"`
	TargetGenres          []string `json:"target_genres,omitempty"`
	TargetLanguages       []string `json:"target_languages,omitempty"`
	TargetDevices         []string `json:"target_devices,omitempty"`
	MaxImpressionsPerUser *int     `json:"max_impressions_per_user,omitempty"`
	MaxImpressionsPerDay  *int     `json:"max_impressions_per_day,omitempty"`
}

// CreateAdvertisement creates a new advertisement
// @Summary Create a new advertisement
// @Description Creates a new advertisement within a campaign. Supports multiple ad formats (banner, interstitial, audio, native) and ad types (image, video, audio, html). Includes targeting options for countries, genres, languages, and devices. Pricing can be CPM (cost per mille impressions) or CPC (cost per click).
// @Tags Advertisements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param advertisement body CreateAdvertisementRequest true "Advertisement data"
// @Success 201 {object} domain.Advertisement "Advertisement created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or campaign_id"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/ads [post]
func (h *AdvertisementHandler) CreateAdvertisement(c *gin.Context) {
	var req CreateAdvertisementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid advertisement creation request", "error", err)
		RespondWithError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	campaignID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_campaign_id", "Invalid campaign_id")
		return
	}

	ad := &domain.Advertisement{
		ID:                    uuid.New(),
		CampaignID:            campaignID,
		Title:                 req.Title,
		Description:           req.Description,
		AdvertiserName:        req.AdvertiserName,
		MediaURL:              req.MediaURL,
		AdFormat:              domain.AdFormat(req.AdFormat),
		AdType:                domain.AdType(req.AdType),
		ClickURL:              req.ClickURL,
		TotalBudgetCents:      req.TotalBudgetCents,
		CPMRateCents:          req.CPMRateCents,
		CPCRateCents:          req.CPCRateCents,
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		TargetCountries:       req.TargetCountries,
		TargetGenres:          req.TargetGenres,
		TargetLanguages:       req.TargetLanguages,
		TargetDevices:         req.TargetDevices,
		MaxImpressionsPerUser: req.MaxImpressionsPerUser,
		MaxImpressionsPerDay:  req.MaxImpressionsPerDay,
		Status:                domain.AdStatusActive,
		Priority:              1,
		ImpressionsCount:      0,
		ClicksCount:           0,
		SpendCents:            0,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := h.adService.CreateAdvertisement(c.Request.Context(), ad); err != nil {
		h.logger.Error("Failed to create advertisement", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "create_failed", "Failed to create advertisement")
		return
	}

	RespondWithSuccess(c, http.StatusCreated, ad)
}

// GetAdvertisement retrieves an advertisement by ID
// @Summary Get advertisement details
// @Description Retrieves detailed information about a specific advertisement including current statistics (impressions, clicks, spend, CTR).
// @Tags Advertisements
// @Produce json
// @Security BearerAuth
// @Param id path string true "Advertisement ID (UUID)"
// @Success 200 {object} domain.Advertisement "Advertisement details"
// @Failure 400 {object} ErrorResponse "Invalid advertisement ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 404 {object} ErrorResponse "Advertisement not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/ads/{id} [get]
func (h *AdvertisementHandler) GetAdvertisement(c *gin.Context) {
	adID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	ad, err := h.adService.GetAdvertisement(c.Request.Context(), adID)
	if err != nil {
		h.logger.Error("Failed to get advertisement", "error", err, "ad_id", adID)
		RespondWithError(c, http.StatusNotFound, "advertisement_not_found", "Advertisement not found")
		return
	}

	RespondWithSuccess(c, http.StatusOK, ad)
}

// GetAdvertisementsByCampaign retrieves all advertisements for a campaign
// @Summary List advertisements by campaign
// @Description Retrieves all advertisements belonging to a specific campaign. Useful for campaign managers to see all their ads in one place.
// @Tags Advertisements
// @Produce json
// @Security BearerAuth
// @Param campaign_id path string true "Campaign ID (UUID)"
// @Success 200 {array} domain.Advertisement "List of advertisements"
// @Failure 400 {object} ErrorResponse "Invalid campaign ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/{campaign_id}/ads [get]
func (h *AdvertisementHandler) GetAdvertisementsByCampaign(c *gin.Context) {
	campaignID, ok := GetUUIDParam(c, "campaign_id")
	if !ok {
		return
	}

	ads, err := h.adService.GetAdvertisementsByCampaign(c.Request.Context(), campaignID)
	if err != nil {
		h.logger.Error("Failed to get advertisements", "error", err, "campaign_id", campaignID)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get advertisements")
		return
	}

	RespondWithSuccess(c, http.StatusOK, ads)
}

// UpdateAdvertisement updates an advertisement
// @Summary Update advertisement
// @Description Updates an existing advertisement. All fields are optional - only provided fields will be updated. Cannot update if campaign is paused or budget is exhausted.
// @Tags Advertisements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Advertisement ID (UUID)"
// @Param advertisement body UpdateAdvertisementRequest true "Fields to update"
// @Success 200 {object} domain.Advertisement "Updated advertisement"
// @Failure 400 {object} ErrorResponse "Invalid ID format or request body"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 404 {object} ErrorResponse "Advertisement not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/ads/{id} [put]
func (h *AdvertisementHandler) UpdateAdvertisement(c *gin.Context) {
	adID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	var req UpdateAdvertisementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid advertisement update request", "error", err)
		RespondWithError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Get existing advertisement
	ad, err := h.adService.GetAdvertisement(c.Request.Context(), adID)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "advertisement_not_found", "Advertisement not found")
		return
	}

	// Apply updates
	if req.Title != nil {
		ad.Title = *req.Title
	}
	if req.Description != nil {
		ad.Description = req.Description
	}
	if req.MediaURL != nil {
		ad.MediaURL = *req.MediaURL
	}
	if req.ClickURL != nil {
		ad.ClickURL = *req.ClickURL
	}
	if req.TotalBudgetCents != nil {
		ad.TotalBudgetCents = *req.TotalBudgetCents
	}
	if req.CPMRateCents != nil {
		ad.CPMRateCents = req.CPMRateCents
	}
	if req.CPCRateCents != nil {
		ad.CPCRateCents = req.CPCRateCents
	}
	if req.TargetCountries != nil {
		ad.TargetCountries = req.TargetCountries
	}
	if req.TargetGenres != nil {
		ad.TargetGenres = req.TargetGenres
	}
	if req.TargetLanguages != nil {
		ad.TargetLanguages = req.TargetLanguages
	}
	if req.TargetDevices != nil {
		ad.TargetDevices = req.TargetDevices
	}
	if req.MaxImpressionsPerUser != nil {
		ad.MaxImpressionsPerUser = req.MaxImpressionsPerUser
	}
	if req.MaxImpressionsPerDay != nil {
		ad.MaxImpressionsPerDay = req.MaxImpressionsPerDay
	}

	if err := h.adService.UpdateAdvertisement(c.Request.Context(), ad); err != nil {
		h.logger.Error("Failed to update advertisement", "error", err, "ad_id", adID)
		RespondWithError(c, http.StatusInternalServerError, "update_failed", "Failed to update advertisement")
		return
	}

	RespondWithSuccess(c, http.StatusOK, ad)
}

// DeleteAdvertisement deletes an advertisement
// @Summary Delete advertisement
// @Description Permanently deletes an advertisement. This action cannot be undone. All associated statistics will be preserved but the ad will no longer be served.
// @Tags Advertisements
// @Security BearerAuth
// @Param id path string true "Advertisement ID (UUID)"
// @Success 204 "Advertisement deleted successfully (no content)"
// @Failure 400 {object} ErrorResponse "Invalid advertisement ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/ads/{id} [delete]
func (h *AdvertisementHandler) DeleteAdvertisement(c *gin.Context) {
	adID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.adService.DeleteAdvertisement(c.Request.Context(), adID); err != nil {
		h.logger.Error("Failed to delete advertisement", "error", err, "ad_id", adID)
		RespondWithError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete advertisement")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetEligibleAds retrieves ads eligible for a user based on targeting criteria (KEY ENDPOINT)
// @Summary Get eligible advertisements for user
// @Description **KEY ENDPOINT FOR FRONTEND**: Returns advertisements that match the user's profile and targeting criteria. Considers geographic location, language, device type, genre preferences, frequency capping, and premium status. Premium users receive empty array. Uses smart caching and fraud detection.
// @Tags Advertisements
// @Produce json
// @Param user_id query string false "User ID (UUID) - optional for guest users"
// @Param country query string false "User's country code (e.g., 'US', 'MX', 'ES')"
// @Param genre query string false "Current station genre/tag (e.g., 'rock', 'jazz')"
// @Param language query string false "User's preferred language code (e.g., 'en', 'es', 'fr')"
// @Param device query string false "Device type: 'mobile', 'tablet', 'desktop', 'smart_speaker'"
// @Param is_premium query boolean false "Whether user has premium subscription (default: false)"
// @Param limit query integer false "Maximum number of ads to return (default: 5, max: 10)"
// @Success 200 {object} EligibleAdsResponse "Eligible advertisements (empty for premium users)"
// @Failure 400 {object} ErrorResponse "Invalid query parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/ads/eligible [get]
func (h *AdvertisementHandler) GetEligibleAds(c *gin.Context) {
	// Extract user ID from context or query (optional)
	var userID *uuid.UUID
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_user_id", "Invalid user_id")
			return
		}
		userID = &parsedID
	}

	// Extract targeting criteria from query parameters
	country := c.Query("country")
	genre := c.Query("genre")
	language := c.Query("language")
	device := c.Query("device")

	// Check if user is premium
	isPremium := c.Query("is_premium") == "true"

	h.logger.Info("Getting eligible ads",
		"user_id", userID,
		"country", country,
		"genre", genre,
		"language", language,
		"device", device,
		"is_premium", isPremium,
	)

	// Get eligible ads - use uuid.Nil if userID is nil
	var userIDValue uuid.UUID
	if userID != nil {
		userIDValue = *userID
	}

	ads, err := h.adService.GetEligibleAdsForUser(c.Request.Context(), userIDValue, country, genre, language, device, isPremium)
	if err != nil {
		h.logger.Error("Failed to get eligible ads", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get eligible ads")
		return
	}

	// Return empty array if no ads (premium users or frequency cap exceeded)
	if ads == nil {
		ads = []*domain.Advertisement{}
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"ads":   ads,
		"count": len(ads),
	})
}

// GetAdvertisementStats retrieves statistics for an advertisement
// @Summary Get advertisement statistics
// @Description Retrieves comprehensive performance statistics for a specific advertisement including impressions, clicks, CTR (click-through rate), total spend, and remaining budget. Useful for campaign performance monitoring.
// @Tags Advertisements
// @Produce json
// @Security BearerAuth
// @Param id path string true "Advertisement ID (UUID)"
// @Success 200 {object} services.AdvertisementStats "Advertisement statistics"
// @Failure 400 {object} ErrorResponse "Invalid advertisement ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/ads/{id}/stats [get]
func (h *AdvertisementHandler) GetAdvertisementStats(c *gin.Context) {
	adID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	stats, err := h.adService.GetAdvertisementStats(c.Request.Context(), adID)
	if err != nil {
		h.logger.Error("Failed to get advertisement stats", "error", err, "ad_id", adID)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get stats")
		return
	}

	RespondWithSuccess(c, http.StatusOK, stats)
}
