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

// CampaignPerformanceListResponse wraps the campaign performance list with a count.
type CampaignPerformanceListResponse struct {
	Campaigns []CampaignPerformanceResponse `json:"campaigns"`
	Count     int                           `json:"count" example:"3"`
}

// FraudScoreDistribution buckets fraud scores into low/medium/high ranges.
type FraudScoreDistribution struct {
	Low    int `json:"low" example:"0"`
	Medium int `json:"medium" example:"0"`
	High   int `json:"high" example:"0"`
}

// FraudMetricsResponse represents fraud detection metrics.
type FraudMetricsResponse struct {
	FraudAttemptsCount     int                    `json:"fraud_attempts_count" example:"0"`
	BlockedImpressions     int                    `json:"blocked_impressions" example:"0"`
	BlockedClicks          int                    `json:"blocked_clicks" example:"0"`
	FraudScoreDistribution FraudScoreDistribution `json:"fraud_score_distribution"`
}

// TopAdsResponse represents the top performing ads ranked by a metric.
type TopAdsResponse struct {
	Metric string                   `json:"metric" example:"impressions"`
	TopAds []map[string]interface{} `json:"top_ads"`
	Count  int                      `json:"count" example:"0"`
}

// DashboardOverviewResponse represents the admin dashboard summary metrics.
type DashboardOverviewResponse struct {
	ActiveCampaignsCount int     `json:"active_campaigns_count" example:"5"`
	TotalBudgetCents     int     `json:"total_budget_cents" example:"100000"`
	TotalSpentCents      int     `json:"total_spent_cents" example:"42000"`
	TotalImpressions     int64   `json:"total_impressions" example:"0"`
	TotalClicks          int64   `json:"total_clicks" example:"0"`
	CTR                  float64 `json:"ctr" example:"0"`
	BudgetUtilization    float64 `json:"budget_utilization" example:"42"`
}

// GetRevenueAnalytics retrieves revenue analytics for a date range
// @Summary Get revenue analytics
// @Description Returns aggregated revenue metrics (impressions, clicks, CPM, CPC, CTR) for a given date range. Defaults to the last 30 days.
// @Tags Admin Analytics
// @Produce json
// @Security BearerAuth
// @Param from query string false "Start date (RFC3339)" example("2024-01-01T00:00:00Z")
// @Param to query string false "End date (RFC3339)" example("2024-01-31T23:59:59Z")
// @Success 200 {object} RevenueAnalyticsResponse "Revenue analytics data"
// @Failure 400 {object} ErrorResponse "Invalid date format"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Router /api/v1/admin/analytics/revenue [get]
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
// @Summary Get campaign performance
// @Description Returns performance metrics (impressions, clicks, CTR, spend) for all active campaigns.
// @Tags Admin Analytics
// @Produce json
// @Security BearerAuth
// @Success 200 {object} CampaignPerformanceListResponse "List of campaign performance metrics"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Failed to retrieve campaigns"
// @Router /api/v1/admin/analytics/campaigns [get]
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
// @Summary Get fraud metrics
// @Description Returns fraud detection metrics including blocked impressions, blocked clicks, and fraud score distribution.
// @Tags Admin Analytics
// @Produce json
// @Security BearerAuth
// @Success 200 {object} FraudMetricsResponse "Fraud detection metrics"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Router /api/v1/admin/analytics/fraud [get]
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
// @Summary Get top performing ads
// @Description Returns the top performing advertisements ranked by the specified metric (impressions, clicks, ctr, revenue).
// @Tags Admin Analytics
// @Produce json
// @Security BearerAuth
// @Param metric query string false "Ranking metric: impressions | clicks | ctr | revenue" default(impressions)
// @Success 200 {object} TopAdsResponse "Top ads list with the applied metric"
// @Failure 400 {object} ErrorResponse "Invalid metric value"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Router /api/v1/admin/analytics/top-ads [get]
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
// @Summary Get admin dashboard overview
// @Description Returns a summary of active campaigns, total budget, total spend, impressions, clicks, CTR, and budget utilization percentage.
// @Tags Admin Analytics
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DashboardOverviewResponse "Dashboard overview metrics"
// @Failure 401 {object} SimpleErrorResponse "Unauthorized"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Failed to retrieve dashboard data"
// @Router /api/v1/admin/analytics/dashboard [get]
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
