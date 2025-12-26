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
	ID            string               `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name          string               `json:"name" example:"Rock FM 100.1"`
	Slug          string               `json:"slug" example:"rock-fm-100-1"`
	StreamURL     string               `json:"stream_url" example:"https://stream.rockfm.com/live"`
	ImageURL      string               `json:"image_url,omitempty" example:"https://cdn.rockfm.com/logo.png"`
	Tags          []string             `json:"tags" example:"rock,classic rock,80s"`
	Country       string               `json:"country" example:"United States"`
	Votes         int                  `json:"votes" example:"1500"`
	IsPremiumOnly bool                 `json:"is_premium_only" example:"false"`
	SEOMetadata   *domain.SEOMetadata  `json:"seo_metadata,omitempty"`
}

// StationDetailResponse represents the response for station detail endpoint
type StationDetailResponse struct {
	Data        StationDTO           `json:"data"`
	SEOMetadata *domain.SEOMetadata  `json:"seo_metadata,omitempty"`
}

// StationListResponse represents the response for station list endpoints
type StationListResponse struct {
	Data []StationDTO           `json:"data"`
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// GetByID returns a station by ID
// @Summary Obtener detalle de estación
// @Description Obtiene información detallada de una estación por su ID con metadata SEO enriquecida. Puede devolver 503 si el servicio externo está temporalmente no disponible.
// @Tags Stations
// @Accept json
// @Produce json
// @Param id path string true "ID de la estación"
// @Param lang query string false "Código de idioma (es, en, fr, de)" default(es)
// @Success 200 {object} StationDetailResponse "Detalle de la estación con SEO metadata"
// @Failure 404 {object} map[string]interface{} "Estación no encontrada"
// @Failure 403 {object} map[string]interface{} "Acceso denegado - Estación solo para Premium"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Failure 503 {object} map[string]interface{} "Servicio externo temporalmente no disponible"
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
	station, err := h.stationService.GetByID(stationID, userType, lang)
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

	// Verificar si la estación tiene stream válido (404 SEO)
	if station.StreamURL == "" {
		logger.Warn("station has no stream - returning 404 for SEO", "station_id", stationID)
		RespondWithError(c, http.StatusNotFound, "station_unavailable", "Station is currently unavailable")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": gin.H{
			"id":              station.ID,
			"name":            station.Name,
			"slug":            station.Slug, // NUEVO: Slug para SEO
			"stream_url":      station.StreamURL,
			"image_url":       station.ImageURL,
			"tags":            station.Tags,
			"country":         station.Country,
			"votes":           station.Votes,
			"is_premium_only": station.IsPremiumOnly,
		},
		"seo_metadata": station.SEOMetadata, // NUEVO: Metadata SEO
	})
}

// GetPopular returns popular stations
// @Summary Obtener estaciones populares
// @Description Lista de estaciones de radio populares con filtros opcionales y metadata SEO. Puede devolver 503 si el servicio externo está temporalmente no disponible.
// @Tags Stations
// @Accept json
// @Produce json
// @Param limit query int false "Número máximo de estaciones" default(20)
// @Param country query string false "Filtrar por código de país"
// @Param lang query string false "Código de idioma (es, en, fr, de)" default(es)
// @Success 200 {object} StationListResponse "Lista de estaciones populares con SEO metadata"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Failure 503 {object} map[string]interface{} "Servicio externo temporalmente no disponible"
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
	stations, err := h.stationService.ListPopular(limit, country, userType, lang)
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
	stationDTOs := make([]gin.H, 0, len(stations))
	for _, station := range stations {
		stationDTOs = append(stationDTOs, gin.H{
			"id":              station.ID,
			"name":            station.Name,
			"slug":            station.Slug, // NUEVO: Slug para SEO
			"stream_url":      station.StreamURL,
			"image_url":       station.ImageURL,
			"tags":            station.Tags,
			"country":         station.Country,
			"votes":           station.Votes,
			"is_premium_only": station.IsPremiumOnly,
			"seo_metadata":    station.SEOMetadata, // NUEVO: Metadata SEO
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
// @Summary Buscar estaciones
// @Description Busca estaciones de radio por nombre o tags con metadata SEO enriquecida. Puede devolver 503 si el servicio externo está temporalmente no disponible (Circuit Breaker abierto).
// @Tags Stations
// @Accept json
// @Produce json
// @Param q query string true "Término de búsqueda"
// @Param limit query int false "Número máximo de resultados" default(20)
// @Param lang query string false "Código de idioma (es, en, fr, de)" default(es)
// @Success 200 {object} StationListResponse "Resultados de búsqueda con SEO metadata"
// @Failure 400 {object} map[string]interface{} "Parámetro de búsqueda requerido"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Failure 503 {object} map[string]interface{} "Servicio externo temporalmente no disponible"
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
	stations, err := h.stationService.Search(query, limit, userType, lang)
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
	stationDTOs := make([]gin.H, 0, len(stations))
	for _, station := range stations {
		stationDTOs = append(stationDTOs, gin.H{
			"id":              station.ID,
			"name":            station.Name,
			"slug":            station.Slug, // NUEVO: Slug para SEO
			"stream_url":      station.StreamURL,
			"image_url":       station.ImageURL,
			"tags":            station.Tags,
			"country":         station.Country,
			"votes":           station.Votes,
			"is_premium_only": station.IsPremiumOnly,
			"seo_metadata":    station.SEOMetadata, // NUEVO: Metadata SEO
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
