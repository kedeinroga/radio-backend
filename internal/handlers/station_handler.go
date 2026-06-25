package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/logger"
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

// StationDTO represents a station in API responses
type StationDTO struct {
	ID            string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name          string              `json:"name" example:"Rock FM 100.1"`
	Slug          string              `json:"slug" example:"rock-fm-100-1"`
	StreamURL     string              `json:"stream_url" example:"https://stream.rockfm.com/live"`
	ImageURL      string              `json:"image_url,omitempty" example:"https://cdn.rockfm.com/logo.png"`
	Tags          []string            `json:"tags" example:"rock,classic rock,80s"`
	Country       string              `json:"country" example:"United States"`
	Votes         int                 `json:"votes" example:"1500"`
	IsPremiumOnly bool                `json:"is_premium_only" example:"false"`
	SEOMetadata   *domain.SEOMetadata `json:"seo_metadata,omitempty"`
}

// StationDetailResponse represents the response for station detail endpoint
type StationDetailResponse struct {
	Data        StationDTO          `json:"data"`
	SEOMetadata *domain.SEOMetadata `json:"seo_metadata,omitempty"`
}

// StationListResponse represents the response for station list endpoints
type StationListResponse struct {
	Data []StationDTO           `json:"data"`
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// GetByID returns a station by ID
// @Summary Get station detail
// @Description Gets detailed information for a station by its ID with enriched SEO metadata. May return 503 if external service is temporarily unavailable.
// @Tags Stations
// @Accept json
// @Produce json
// @Security SharedSecret
// @Param X-Rradio-Secret header string true "Shared API secret for bot protection"
// @Param id path string true "Station ID"
// @Param lang query string false "Language code (es, en, fr, de)" default(es)
// @Success 200 {object} StationDetailResponse "Station detail with SEO metadata"
// @Failure 400 {object} ErrorResponse "Station ID is required"
// @Failure 401 {object} ErrorResponse "Unauthorized – missing or invalid X-Rradio-Secret"
// @Failure 403 {object} ErrorResponse "Access denied - Premium only station"
// @Failure 404 {object} ErrorResponse "Station not found or has no stream"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "External service temporarily unavailable"
// @Router /stations/{id} [get]
func (h *StationHandler) GetByID(c *gin.Context) {
	stationID := c.Param("id")
	if stationID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Station ID is required")
		return
	}

	// Get user type from context
	userType := middleware.GetUserType(c)

	// Get language from context (detectado por middleware)
	lang := middleware.GetLanguage(c)

	// Get station by ID
	station, err := h.stationService.GetByID(c.Request.Context(), stationID, userType, lang)
	if err != nil {
		if err == domain.ErrStationNotFound {
			RespondWithError(c, http.StatusNotFound, "station_not_found", "Station not found")
			return
		}
		if err == domain.ErrUnauthorized {
			RespondWithError(c, http.StatusForbidden, "premium_only", "This station is only available for premium users")
			return
		}

		// Check if it's a circuit breaker error
		if strings.Contains(err.Error(), "temporarily unavailable") {
			RespondWithError(c, http.StatusServiceUnavailable, "api_unavailable",
				"The radio station service is temporarily unavailable. Please try again in a few moments.")
			return
		}

		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch station")
		return
	}

	// Check if station has a valid stream (404 for SEO)
	if station.StreamURL == "" {
		logger.Warn("station has no stream - returning 404 for SEO", "station_id", stationID)
		RespondWithError(c, http.StatusNotFound, "station_unavailable", "Station is currently unavailable")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data":         h.toStationDTO(*station),
		"seo_metadata": station.SEOMetadata,
	})
}

// GetPopular returns popular stations
// @Summary Get popular stations
// @Description List of popular radio stations with optional filters and SEO metadata. May return 503 if external service is temporarily unavailable.
// @Tags Stations
// @Accept json
// @Produce json
// @Security SharedSecret
// @Param X-Rradio-Secret header string true "Shared API secret for bot protection"
// @Param limit query int false "Maximum number of stations" default(20)
// @Param country query string false "Filter by country code"
// @Param lang query string false "Language code (es, en, fr, de)" default(es)
// @Success 200 {object} StationListResponse "List of popular stations with SEO metadata"
// @Failure 401 {object} ErrorResponse "Unauthorized – missing or invalid X-Rradio-Secret"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "External service temporarily unavailable"
// @Router /stations/popular [get]
func (h *StationHandler) GetPopular(c *gin.Context) {
	// Parse query parameters
	limit := parseIntQuery(c, "limit", 20)
	country := c.Query("country")

	// Get user type from context
	userType := middleware.GetUserType(c)

	// Get language from context
	lang := middleware.GetLanguage(c)

	// Get popular stations
	stations, err := h.stationService.ListPopular(c.Request.Context(), limit, country, userType, lang)
	if err != nil {
		// Check if it's a circuit breaker error
		if strings.Contains(err.Error(), "temporarily unavailable") {
			RespondWithError(c, http.StatusServiceUnavailable, "api_unavailable",
				"The radio station service is temporarily unavailable. Please try again in a few moments.")
			return
		}

		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch popular stations")
		return
	}

	// Convert to DTOs with SEO metadata
	stationDTOs := h.toStationDTOs(stations)

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": stationDTOs,
		"meta": gin.H{
			"count":     len(stationDTOs),
			"user_type": userType.String(),
		},
	})
}

// Search searches for stations
// @Summary Search stations
// @Description Searches for radio stations by name or tags with enriched SEO metadata. May return 503 if external service is temporarily unavailable (Circuit Breaker open).
// @Tags Stations
// @Accept json
// @Produce json
// @Security SharedSecret
// @Param X-Rradio-Secret header string true "Shared API secret for bot protection"
// @Param q query string true "Search term"
// @Param limit query int false "Maximum number of results" default(20)
// @Param lang query string false "Language code (es, en, fr, de)" default(es)
// @Success 200 {object} StationListResponse "Search results with SEO metadata"
// @Failure 400 {object} ErrorResponse "Search parameter required"
// @Failure 401 {object} ErrorResponse "Unauthorized – missing or invalid X-Rradio-Secret"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "External service temporarily unavailable"
// @Router /stations/search [get]
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

	// Get language from context
	lang := middleware.GetLanguage(c)

	// Search stations
	stations, err := h.stationService.Search(c.Request.Context(), query, limit, userType, lang)
	if err != nil {
		logger.Error("search failed", "query", query, "limit", limit, "user_type", userType, "error", err)

		// Check if it's a circuit breaker error (API temporarily unavailable)
		if strings.Contains(err.Error(), "temporarily unavailable") {
			RespondWithError(c, http.StatusServiceUnavailable, "api_unavailable",
				"The radio station service is temporarily unavailable. Please try again in a few moments.")
			return
		}

		RespondWithError(c, http.StatusInternalServerError, "search_failed", "Failed to search stations")
		return
	}

	// Track search analytics asynchronously
	go func() {
		_ = h.analyticsService.TrackSearch(query, len(stations), userID, userType)
	}()

	// Convert to DTOs with SEO metadata
	stationDTOs := h.toStationDTOs(stations)

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": stationDTOs,
		"meta": gin.H{
			"count":     len(stationDTOs),
			"user_type": userType.String(),
		},
	})
}

// toStationDTO converts a domain Station to StationDTO
func (h *StationHandler) toStationDTO(station domain.Station) StationDTO {
	return StationDTO{
		ID:            station.ID,
		Name:          station.Name,
		Slug:          station.Slug,
		StreamURL:     station.StreamURL,
		ImageURL:      station.ImageURL,
		Tags:          station.Tags,
		Country:       station.Country,
		Votes:         station.Votes,
		IsPremiumOnly: station.IsPremiumOnly,
		SEOMetadata:   station.SEOMetadata,
	}
}

// toStationDTOs converts a slice of domain Stations to a slice of StationDTOs
func (h *StationHandler) toStationDTOs(stations []domain.Station) []StationDTO {
	dtos := make([]StationDTO, len(stations))
	for i, station := range stations {
		dtos[i] = h.toStationDTO(station)
	}
	return dtos
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
