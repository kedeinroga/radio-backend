package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"radio-backend/internal/services"
)

// AdminAnalyticsHandler handles admin analytics endpoints
type AdminAnalyticsHandler struct {
	analyticsService *services.AdCampaignService // Reusing campaign service for analytics
	logger           *slog.Logger
}

// NewAdminAnalyticsHandler creates a new admin analytics handler
func NewAdminAnalyticsHandler(
	analyticsService *services.AdCampaignService,
	logger *slog.Logger,
) *AdminAnalyticsHandler {
	return &AdminAnalyticsHandler{
		analyticsService: analyticsService,
		logger:           logger,
	}
}

// RevenueAnalyticsResponse represents revenue analytics data
type RevenueAnalyticsResponse struct {
	TotalRevenueCents int64     `json:"total_revenue_cents"`
	TotalImpressions  int64     `json:"total_impressions"`
	TotalClicks       int64     `json:"total_clicks"`
	AverageCPMCents   float64   `json:"average_cpm_cents"`
	AverageCPCCents   float64   `json:"average_cpc_cents"`
	CTR               float64   `json:"ctr"`
	From              time.Time `json:"from"`
	To                time.Time `json:"to"`
}

// CampaignPerformanceResponse represents campaign performance data
type CampaignPerformanceResponse struct {
	CampaignID       uuid.UUID `json:"campaign_id"`
	CampaignName     string    `json:"campaign_name"`
	SpentCents       int       `json:"spent_cents"`
	TotalImpressions int64     `json:"total_impressions"`
	TotalClicks      int64     `json:"total_clicks"`
	CTR              float64   `json:"ctr"`
	Status           string    `json:"status"`
}

// GetRevenueAnalytics retrieves revenue analytics for a date range
func (h *AdminAnalyticsHandler) GetRevenueAnalytics(c *gin.Context) {
	from := time.Now().Add(-30 * 24 * time.Hour) // Default: last 30 days
	if fromStr := c.Query("from"); fromStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_from_date", "Invalid from format (use RFC3339)")
			return
		}
		from = parsedTime
	}

	to := time.Now()
	if toStr := c.Query("to"); toStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_to_date", "Invalid to format (use RFC3339)")
			return
		}
		to = parsedTime
	}

	// For now, return mock data (would integrate with actual analytics service)
	response := RevenueAnalyticsResponse{
		TotalRevenueCents: 0,
		TotalImpressions:  0,
		TotalClicks:       0,
		AverageCPMCents:   0,
		AverageCPCCents:   0,
		CTR:               0,
		From:              from,
		To:                to,
	}

	h.logger.Info("Retrieved revenue analytics", "from", from, "to", to)
	RespondWithSuccess(c, http.StatusOK, response)
}

// GetCampaignPerformance retrieves performance metrics for all campaigns
func (h *AdminAnalyticsHandler) GetCampaignPerformance(c *gin.Context) {
	// Get all active campaigns
	campaigns, err := h.analyticsService.GetActiveCampaigns(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get active campaigns", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get campaigns")
		return
	}

	// Build performance response for each campaign
	performances := make([]CampaignPerformanceResponse, 0, len(campaigns))
	for _, campaign := range campaigns {
		// Get campaign stats
		stats, err := h.analyticsService.GetCampaignStats(c.Request.Context(), campaign.ID)
		if err != nil {
			h.logger.Warn("Failed to get campaign stats", "error", err, "campaign_id", campaign.ID)
			continue
		}

		// Calculate CTR (from spent budget and typical cost)
		ctr := 0.0 // Would need actual impression/click data

		performances = append(performances, CampaignPerformanceResponse{
			CampaignID:       campaign.ID,
			CampaignName:     campaign.Name,
			SpentCents:       stats.SpentCents,
			TotalImpressions: 0, // Would come from analytics service
			TotalClicks:      0, // Would come from analytics service
			CTR:              ctr,
			Status:           string(campaign.Status),
		})
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"campaigns": performances,
		"count":     len(performances),
	})
}

// GetFraudMetrics retrieves fraud detection metrics
func (h *AdminAnalyticsHandler) GetFraudMetrics(c *gin.Context) {
	// Mock fraud metrics (would integrate with actual fraud detection service)
	response := gin.H{
		"fraud_attempts_count": 0,
		"blocked_impressions":  0,
		"blocked_clicks":       0,
		"fraud_score_distribution": gin.H{
			"low":    0, // 0.0 - 0.3
			"medium": 0, // 0.3 - 0.6
			"high":   0, // 0.6 - 1.0
		},
	}

	h.logger.Info("Retrieved fraud metrics")
	RespondWithSuccess(c, http.StatusOK, response)
}

// GetTopAds retrieves top performing ads
func (h *AdminAnalyticsHandler) GetTopAds(c *gin.Context) {
	metric := c.Query("metric")
	if metric == "" {
		metric = "impressions" // Default metric
	}

	// Validate metric
	validMetrics := map[string]bool{
		"impressions": true,
		"clicks":      true,
		"ctr":         true,
		"revenue":     true,
	}

	if !validMetrics[metric] {
		RespondWithError(c, http.StatusBadRequest, "invalid_metric", "Invalid metric (use: impressions, clicks, ctr, revenue)")
		return
	}

	// Mock top ads (would query actual data)
	topAds := []gin.H{}

	h.logger.Info("Retrieved top ads", "metric", metric)
	RespondWithSuccess(c, http.StatusOK, gin.H{
		"metric":  metric,
		"top_ads": topAds,
		"count":   len(topAds),
	})
}

// GetDashboardOverview retrieves dashboard overview metrics
func (h *AdminAnalyticsHandler) GetDashboardOverview(c *gin.Context) {
	// Get all active campaigns
	activeCampaigns, err := h.analyticsService.GetActiveCampaigns(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get active campaigns", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to get dashboard data")
		return
	}

	// Aggregate metrics
	totalBudgetCents := 0
	totalSpentCents := 0

	for _, campaign := range activeCampaigns {
		totalBudgetCents += campaign.TotalBudgetCents

		stats, err := h.analyticsService.GetCampaignStats(c.Request.Context(), campaign.ID)
		if err != nil {
			h.logger.Warn("Failed to get campaign stats", "error", err, "campaign_id", campaign.ID)
			continue
		}

		totalSpentCents += stats.SpentCents
	}

	overview := gin.H{
		"active_campaigns_count": len(activeCampaigns),
		"total_budget_cents":     totalBudgetCents,
		"total_spent_cents":      totalSpentCents,
		"total_impressions":      0, // Would come from analytics service
		"total_clicks":           0, // Would come from analytics service
		"ctr":                    0.0,
		"budget_utilization":     0.0,
	}

	if totalBudgetCents > 0 {
		overview["budget_utilization"] = (float64(totalSpentCents) / float64(totalBudgetCents)) * 100
	}

	h.logger.Info("Retrieved dashboard overview")
	RespondWithSuccess(c, http.StatusOK, overview)
}
