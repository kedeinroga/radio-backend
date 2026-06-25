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

// AdCampaignHandler handles HTTP requests for ad campaigns
type AdCampaignHandler struct {
	campaignService *services.AdCampaignService
	logger          *slog.Logger
}

// NewAdCampaignHandler creates a new ad campaign handler
func NewAdCampaignHandler(
	campaignService *services.AdCampaignService,
	logger *slog.Logger,
) *AdCampaignHandler {
	return &AdCampaignHandler{
		campaignService: campaignService,
		logger:          logger,
	}
}

// CreateCampaignRequest represents the request body for creating a campaign
type CreateCampaignRequest struct {
	Name             string    `json:"name" binding:"required,min=3,max=200"`
	AdvertiserID     string    `json:"advertiser_id" binding:"required,uuid"`
	TotalBudgetCents int       `json:"total_budget_cents" binding:"required,min=1000"`
	StartDate        time.Time `json:"start_date" binding:"required"`
	EndDate          time.Time `json:"end_date" binding:"required"`
	Description      *string   `json:"description,omitempty"`
}

// UpdateCampaignRequest represents the request body for updating a campaign
type UpdateCampaignRequest struct {
	Name             *string    `json:"name,omitempty" binding:"omitempty,min=3,max=200"`
	TotalBudgetCents *int       `json:"total_budget_cents,omitempty" binding:"omitempty,min=1000"`
	StartDate        *time.Time `json:"start_date,omitempty"`
	EndDate          *time.Time `json:"end_date,omitempty"`
	Description      *string    `json:"description,omitempty"`
}

// CreateCampaign creates a new ad campaign
// @Summary Create advertising campaign
// @Description Creates a new advertising campaign for an advertiser. Campaign starts in 'draft' status and must be explicitly activated. Budget is specified in cents (USD). Validates that end_date is after start_date and budget is at least $10.00.
// @Tags Ad Campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param campaign body CreateCampaignRequest true "Campaign data"
// @Success 201 {object} domain.AdCampaign "Campaign created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or advertiser_id"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns [post]
func (h *AdCampaignHandler) CreateCampaign(c *gin.Context) {
	var req CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid campaign creation request", "error", err)
		RespondWithError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	advertiserID, err := uuid.Parse(req.AdvertiserID)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_advertiser_id", "Invalid advertiser_id")
		return
	}

	campaign := &domain.AdCampaign{
		ID:               uuid.New(),
		AdvertiserID:     advertiserID,
		Name:             req.Name,
		Description:      req.Description,
		TotalBudgetCents: req.TotalBudgetCents,
		SpentCents:       0,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		Status:           domain.CampaignStatusDraft,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.campaignService.CreateCampaign(c.Request.Context(), campaign); err != nil {
		h.logger.Error("Failed to create campaign", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "create_failed", "Failed to create campaign")
		return
	}

	RespondWithSuccess(c, http.StatusCreated, campaign)
}

// GetCampaign retrieves a campaign by ID
// @Summary Get campaign details
// @Description Retrieves detailed information about a specific campaign including budget utilization, spend tracking, and current status.
// @Tags Ad Campaigns
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID (UUID)"
// @Success 200 {object} domain.AdCampaign "Campaign details"
// @Failure 400 {object} ErrorResponse "Invalid campaign ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 404 {object} ErrorResponse "Campaign not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/{id} [get]
func (h *AdCampaignHandler) GetCampaign(c *gin.Context) {
	campaignID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	campaign, err := h.campaignService.GetCampaign(c.Request.Context(), campaignID)
	if err != nil {
		h.logger.Error("Failed to get campaign", "error", err, "campaign_id", campaignID)
		RespondWithError(c, http.StatusNotFound, "campaign_not_found", "Campaign not found")
		return
	}

	RespondWithSuccess(c, http.StatusOK, campaign)
}

// GetCampaignsByAdvertiser retrieves all campaigns for an advertiser
// @Summary List campaigns by advertiser
// @Description Retrieves all advertising campaigns belonging to a specific advertiser. Useful for advertiser dashboard to see all their campaigns.
// @Tags Ad Campaigns
// @Produce json
// @Security BearerAuth
// @Param advertiser_id path string true "Advertiser ID (UUID)"
// @Success 200 {array} domain.AdCampaign "List of campaigns"
// @Failure 400 {object} ErrorResponse "Invalid advertiser ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/advertisers/{advertiser_id}/campaigns [get]
func (h *AdCampaignHandler) GetCampaignsByAdvertiser(c *gin.Context) {
	advertiserID, ok := GetUUIDParam(c, "advertiser_id")
	if !ok {
		return
	}

	campaigns, err := h.campaignService.GetCampaignsByAdvertiser(c.Request.Context(), advertiserID)
	if err != nil {
		h.logger.Error("Failed to get campaigns", "error", err, "advertiser_id", advertiserID)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get campaigns")
		return
	}

	RespondWithSuccess(c, http.StatusOK, campaigns)
}

// UpdateCampaign updates a campaign
// @Summary Update campaign
// @Description Updates an existing campaign. All fields are optional - only provided fields will be updated. Budget can only be increased, not decreased. Cannot update if campaign is completed.
// @Tags Ad Campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID (UUID)"
// @Param campaign body UpdateCampaignRequest true "Fields to update"
// @Success 200 {object} domain.AdCampaign "Updated campaign"
// @Failure 400 {object} ErrorResponse "Invalid ID format or request body"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 404 {object} ErrorResponse "Campaign not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/{id} [put]
func (h *AdCampaignHandler) UpdateCampaign(c *gin.Context) {
	campaignID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	var req UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid campaign update request", "error", err)
		RespondWithError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Get existing campaign
	campaign, err := h.campaignService.GetCampaign(c.Request.Context(), campaignID)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "campaign_not_found", "Campaign not found")
		return
	}

	// Apply updates
	if req.Name != nil {
		campaign.Name = *req.Name
	}
	if req.TotalBudgetCents != nil {
		campaign.TotalBudgetCents = *req.TotalBudgetCents
	}
	if req.StartDate != nil {
		campaign.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		campaign.EndDate = *req.EndDate
	}
	if req.Description != nil {
		campaign.Description = req.Description
	}

	if err := h.campaignService.UpdateCampaign(c.Request.Context(), campaign); err != nil {
		h.logger.Error("Failed to update campaign", "error", err, "campaign_id", campaignID)
		RespondWithError(c, http.StatusInternalServerError, "update_failed", "Failed to update campaign")
		return
	}

	RespondWithSuccess(c, http.StatusOK, campaign)
}

// DeleteCampaign deletes a campaign
// @Summary Delete campaign
// @Description Permanently deletes a campaign and all its associated advertisements. This action cannot be undone. All statistics will be preserved but ads will no longer be served.
// @Tags Ad Campaigns
// @Security BearerAuth
// @Param id path string true "Campaign ID (UUID)"
// @Success 204 "Campaign deleted successfully (no content)"
// @Failure 400 {object} ErrorResponse "Invalid campaign ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/{id} [delete]
func (h *AdCampaignHandler) DeleteCampaign(c *gin.Context) {
	campaignID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.campaignService.DeleteCampaign(c.Request.Context(), campaignID); err != nil {
		h.logger.Error("Failed to delete campaign", "error", err, "campaign_id", campaignID)
		RespondWithError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete campaign")
		return
	}

	c.Status(http.StatusNoContent)
}

// PauseCampaign pauses an active campaign
// @Summary Pause campaign
// @Description Pauses an active advertising campaign. All ads in the campaign will stop being served immediately. Campaign can be resumed later.
// @Tags Ad Campaigns
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID (UUID)"
// @Success 200 {object} domain.AdCampaign "Campaign paused successfully"
// @Failure 400 {object} ErrorResponse "Invalid campaign ID or campaign cannot be paused"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/{id}/pause [post]
func (h *AdCampaignHandler) PauseCampaign(c *gin.Context) {
	campaignID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.campaignService.PauseCampaign(c.Request.Context(), campaignID); err != nil {
		h.logger.Error("Failed to pause campaign", "error", err, "campaign_id", campaignID)
		RespondWithError(c, http.StatusBadRequest, "pause_failed", err.Error())
		return
	}

	campaign, _ := h.campaignService.GetCampaign(c.Request.Context(), campaignID)
	RespondWithSuccess(c, http.StatusOK, campaign)
}

// ResumeCampaign resumes a paused campaign
// @Summary Resume campaign
// @Description Resumes a paused advertising campaign. All ads in the campaign will start being served again if they meet targeting criteria and budget is available.
// @Tags Ad Campaigns
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID (UUID)"
// @Success 200 {object} domain.AdCampaign "Campaign resumed successfully"
// @Failure 400 {object} ErrorResponse "Invalid campaign ID or campaign cannot be resumed"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/{id}/resume [post]
func (h *AdCampaignHandler) ResumeCampaign(c *gin.Context) {
	campaignID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.campaignService.ResumeCampaign(c.Request.Context(), campaignID); err != nil {
		h.logger.Error("Failed to resume campaign", "error", err, "campaign_id", campaignID)
		RespondWithError(c, http.StatusBadRequest, "resume_failed", err.Error())
		return
	}

	campaign, _ := h.campaignService.GetCampaign(c.Request.Context(), campaignID)
	RespondWithSuccess(c, http.StatusOK, campaign)
}

// GetCampaignStats retrieves campaign statistics
// @Summary Get campaign statistics
// @Description Retrieves comprehensive performance statistics for a campaign including total impressions, clicks, average CTR, total spend, number of active ads, and top performing ad.
// @Tags Ad Campaigns
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID (UUID)"
// @Success 200 {object} services.CampaignStats "Campaign statistics"
// @Failure 400 {object} ErrorResponse "Invalid campaign ID format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/{id}/stats [get]
func (h *AdCampaignHandler) GetCampaignStats(c *gin.Context) {
	campaignID, ok := GetUUIDParam(c, "id")
	if !ok {
		return
	}

	stats, err := h.campaignService.GetCampaignStats(c.Request.Context(), campaignID)
	if err != nil {
		h.logger.Error("Failed to get campaign stats", "error", err, "campaign_id", campaignID)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get stats")
		return
	}

	RespondWithSuccess(c, http.StatusOK, stats)
}

// GetActiveCampaigns retrieves all active campaigns
// @Summary List active campaigns
// @Description Retrieves all campaigns with 'active' status that are currently running and serving ads.
// @Tags Ad Campaigns
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.AdCampaign "List of active campaigns"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized - JWT token required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/campaigns/active [get]
func (h *AdCampaignHandler) GetActiveCampaigns(c *gin.Context) {
	campaigns, err := h.campaignService.GetActiveCampaigns(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get active campaigns", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get campaigns")
		return
	}

	RespondWithSuccess(c, http.StatusOK, campaigns)
}
