package handlers

import (
	"net/http"

	"radio-backend/internal/domain"
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

// Analytics endpoints wrap their payload in {"success": true, "data": ...},
// which differs from the bare RespondWithSuccess envelope used elsewhere.

// AnalyticsPopularStationDTO is a single popular-station analytics row.
type AnalyticsPopularStationDTO struct {
	StationID string `json:"station_id" example:"abc123"`
	Name      string `json:"name" example:"Rock FM"`
	Country   string `json:"country" example:"USA"`
	Plays     int    `json:"plays" example:"1520"`
	Favicon   string `json:"favicon" example:"https://cdn.example.com/favicon.png"`
	URL       string `json:"url" example:"https://stream.example.com/live"`
}

// PopularStationsAnalyticsResponse is the envelope for popular-station analytics.
type PopularStationsAnalyticsResponse struct {
	Success bool                         `json:"success" example:"true"`
	Data    []AnalyticsPopularStationDTO `json:"data"`
}

// TrendingSearchDTO is a single trending-search analytics row.
type TrendingSearchDTO struct {
	SearchTerm string  `json:"search_term" example:"rock"`
	Count      int     `json:"count" example:"456"`
	Percentage float64 `json:"percentage" example:"12.5"`
}

// TrendingSearchesResponse is the envelope for trending-search analytics.
type TrendingSearchesResponse struct {
	Success bool                `json:"success" example:"true"`
	Data    []TrendingSearchDTO `json:"data"`
}

// CountData carries a single count value.
type CountData struct {
	Count int64 `json:"count" example:"1234"`
}

// CountResponse is the envelope for active/guest user counts.
type CountResponse struct {
	Success bool      `json:"success" example:"true"`
	Data    CountData `json:"data"`
}

// GuestDetailsResponse is the envelope for guest request details.
type GuestDetailsResponse struct {
	Success bool                 `json:"success" example:"true"`
	Data    []domain.GuestDetail `json:"data"`
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
// @Success 200 {object} PopularStationsAnalyticsResponse "Popular stations statistics"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
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
// @Success 200 {object} TrendingSearchesResponse "Trending searches statistics"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
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
// @Success 200 {object} CountResponse "Active users count"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
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
// @Success 200 {object} GuestDetailsResponse "Guest details list"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
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
// @Success 200 {object} CountResponse "Guest users count"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
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
