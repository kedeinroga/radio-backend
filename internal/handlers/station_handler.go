package handlers

import (
	"net/http"
	"strconv"

	"radio-backend/internal/middleware"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// StationHandler handles station endpoints
type StationHandler struct {
	stationService   *services.StationService
	analyticsService *services.AnalyticsService
}

// NewStationHandler creates a new station handler
func NewStationHandler(stationService *services.StationService, analyticsService *services.AnalyticsService) *StationHandler {
	return &StationHandler{
		stationService:   stationService,
		analyticsService: analyticsService,
	}
}

// GetPopular returns popular stations
func (h *StationHandler) GetPopular(c *gin.Context) {
	// Parse query parameters
	limit := parseIntQuery(c, "limit", 20)
	country := c.Query("country")

	// Get user type from context
	userType := middleware.GetUserType(c)

	// Get popular stations
	stations, err := h.stationService.ListPopular(limit, country, userType)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch popular stations")
		return
	}

	// Convert to DTOs
	stationDTOs := make([]gin.H, 0, len(stations))
	for _, station := range stations {
		stationDTOs = append(stationDTOs, gin.H{
			"id":              station.ID,
			"name":            station.Name,
			"stream_url":      station.StreamURL,
			"image_url":       station.ImageURL,
			"tags":            station.Tags,
			"country":         station.Country,
			"votes":           station.Votes,
			"is_premium_only": station.IsPremiumOnly,
		})
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": stationDTOs,
		"meta": gin.H{
			"count":     len(stationDTOs),
			"user_type": userType.String(),
		},
	})
}

// Search searches for stations
func (h *StationHandler) Search(c *gin.Context) {
	// Parse query parameters
	query := c.Query("q")
	if query == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_query", "Query parameter 'q' is required")
		return
	}

	limit := parseIntQuery(c, "limit", 20)

	// Get user type from context
	userType := middleware.GetUserType(c)
	userID := middleware.GetUserID(c)

	// Search stations
	stations, err := h.stationService.Search(query, limit, userType)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "search_failed", "Failed to search stations")
		return
	}

	// Track search analytics asynchronously
	go func() {
		_ = h.analyticsService.TrackSearch(query, len(stations), userID, userType)
	}()

	// Convert to DTOs
	stationDTOs := make([]gin.H, 0, len(stations))
	for _, station := range stations {
		stationDTOs = append(stationDTOs, gin.H{
			"id":              station.ID,
			"name":            station.Name,
			"stream_url":      station.StreamURL,
			"image_url":       station.ImageURL,
			"tags":            station.Tags,
			"country":         station.Country,
			"votes":           station.Votes,
			"is_premium_only": station.IsPremiumOnly,
		})
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": stationDTOs,
		"meta": gin.H{
			"count":     len(stationDTOs),
			"user_type": userType.String(),
		},
	})
}

// parseIntQuery parses an integer query parameter with a default value
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
