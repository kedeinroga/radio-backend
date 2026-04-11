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
// @Summary Popular stations statistics
// @Description Returns the most played stations in a specific time period ordered by number of plays. Includes complete information for each station.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Time range: hour, day, week, month" default(day) Enums(hour, day, week, month)
// @Param limit query int false "Maximum number of results to return" default(10) minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{} "Popular stations statistics" example({"success":true,"data":[{"station_id":"abc123","name":"Rock FM","country":"USA","plays":1520,"favicon":"https://...","url":"https://..."}]})
// @Failure 401 {object} map[string]interface{} "Invalid or missing authentication token" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Access denied - Admin users only" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Internal server error" example({"error":{"code":"fetch_failed","message":"Failed to fetch popular stations"}})
// @Router /analytics/stations/popular [get]
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
			"station_id": stat.StationID,
			"name":       stat.Name,
			"country":    stat.Country,
			"plays":      stat.PlayCount,
			"favicon":    stat.Favicon,
			"url":        stat.URL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    statsDTOs,
	})
}

// GetTrendingSearches returns trending searches analytics
// @Summary Trending searches
// @Description Returns the most frequent search terms in a specific time period ordered by frequency. Includes absolute count and percentage of total.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Time range: hour, day, week, month" default(day) Enums(hour, day, week, month)
// @Param limit query int false "Maximum number of results to return" default(10) minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{} "Trending searches statistics" example({"success":true,"data":[{"search_term":"rock","count":456,"percentage":12.5}]})
// @Failure 401 {object} map[string]interface{} "Invalid or missing authentication token" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Access denied - Admin users only" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Internal server error" example({"error":{"code":"fetch_failed","message":"Failed to fetch trending searches"}})
// @Router /analytics/searches/trending [get]
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
		percentage := 0.0
		if stat.TotalCount > 0 {
			percentage = (float64(stat.SearchCount) / float64(stat.TotalCount)) * 100
		}

		statsDTOs = append(statsDTOs, gin.H{
			"search_term": stat.Query,
			"count":       stat.SearchCount,
			"percentage":  percentage,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    statsDTOs,
	})
}

// GetActiveUsers returns active users count
// @Summary Active users
// @Description Returns the number of authenticated active users in the last 24 hours. Active users are those who have made at least one authenticated request in the specified period.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Successful response" example({"success":true,"data":{"count":1234}})
// @Failure 401 {object} map[string]interface{} "Invalid or missing authentication token" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Access denied - Admin users only" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Internal server error" example({"error":{"code":"fetch_failed","message":"Failed to fetch active users count"}})
// @Router /analytics/users/active [get]
func (h *AnalyticsHandler) GetActiveUsers(c *gin.Context) {
	count, err := h.analyticsService.GetActiveUsersCount()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch active users count")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"count": count,
		},
	})
}

// GetGuestDetails returns detailed request info per guest IP
// @Summary Guest users details
// @Description Returns request details for each guest (unauthenticated) IP address in the given time range, ordered by total requests descending. Useful for understanding guest behavior patterns.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Time range: hour, day, week, month" default(day) Enums(hour, day, week, month)
// @Param limit query int false "Maximum number of IPs to return" default(50) minimum(1) maximum(500)
// @Success 200 {object} map[string]interface{} "Guest details list" example({"success":true,"data":[{"ip_address":"1.2.3.4","total_requests":42,"unique_endpoints":7,"user_agent":"Mozilla/5.0...","first_seen":"2026-04-11T00:00:00Z","last_seen":"2026-04-11T12:00:00Z"}]})
// @Failure 401 {object} map[string]interface{} "Invalid or missing authentication token"
// @Failure 403 {object} map[string]interface{} "Access denied - Admin users only"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /analytics/users/guest/details [get]
func (h *AnalyticsHandler) GetGuestDetails(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "day")
	limit := parseIntQuery(c, "limit", 50)

	details, err := h.analyticsService.GetGuestDetails(timeRange, limit)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch guest details")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    details,
	})
}

// GetGuestUsers returns guest users count
// @Summary Active guest users
// @Description Returns the number of guest (unauthenticated) active users in the last 24 hours. Guest users are identified by their unique IP address and represent users using the application without registering.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Successful response" example({"success":true,"data":{"count":856}})
// @Failure 401 {object} map[string]interface{} "Invalid or missing authentication token" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Access denied - Admin users only" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Internal server error" example({"error":{"code":"fetch_failed","message":"Failed to fetch guest users count"}})
// @Router /analytics/users/guest [get]
func (h *AnalyticsHandler) GetGuestUsers(c *gin.Context) {
	count, err := h.analyticsService.GetGuestUsersCount()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch guest users count")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"count": count,
		},
	})
}
