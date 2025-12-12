package handlers

import (
	"net/http"

	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AnalyticsHandler handles analytics endpoints
type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

// GetPopularStations returns popular stations analytics
func (h *AnalyticsHandler) GetPopularStations(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "day")
	limit := parseIntQuery(c, "limit", 10)

	stats, err := h.analyticsService.GetPopularStations(timeRange, limit)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch popular stations")
		return
	}

	// Convert to DTOs
	statsDTOs := make([]gin.H, 0, len(stats))
	for _, stat := range stats {
		statsDTOs = append(statsDTOs, gin.H{
			"station_id":  stat.StationID,
			"play_count":  stat.PlayCount,
			"duration_ms": stat.Duration.Milliseconds(),
		})
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": statsDTOs,
		"meta": gin.H{
			"count": len(statsDTOs),
			"range": timeRange,
		},
	})
}

// GetTrendingSearches returns trending searches analytics
func (h *AnalyticsHandler) GetTrendingSearches(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "day")
	limit := parseIntQuery(c, "limit", 10)

	stats, err := h.analyticsService.GetTrendingSearches(timeRange, limit)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch trending searches")
		return
	}

	// Convert to DTOs
	statsDTOs := make([]gin.H, 0, len(stats))
	for _, stat := range stats {
		statsDTOs = append(statsDTOs, gin.H{
			"query":        stat.Query,
			"search_count": stat.SearchCount,
			"avg_results":  stat.AvgResults,
		})
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": statsDTOs,
		"meta": gin.H{
			"count": len(statsDTOs),
			"range": timeRange,
		},
	})
}

// GetActiveUsers returns active users count
func (h *AnalyticsHandler) GetActiveUsers(c *gin.Context) {
	count, err := h.analyticsService.GetActiveUsersCount()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch active users count")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": gin.H{
			"active_users": count,
		},
	})
}
