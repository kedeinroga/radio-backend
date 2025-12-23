package handlers

import (
	"net/http"
	"strconv"

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

// GetByID returns a station by ID
// @Summary Obtener detalle de estación
// @Description Obtiene información detallada de una estación por su ID
// @Tags Stations
// @Accept json
// @Produce json
// @Param id path string true "ID de la estación"
// @Success 200 {object} map[string]interface{} "Detalle de la estación"
// @Failure 404 {object} map[string]interface{} "Estación no encontrada"
// @Failure 403 {object} map[string]interface{} "Acceso denegado - Estación solo para Premium"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /stations/{id} [get]
func (h *StationHandler) GetByID(c *gin.Context) {
	stationID := c.Param("id")
	if stationID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Station ID is required")
		return
	}

	// Get user type from context
	userType := middleware.GetUserType(c)

	// Get station by ID
	station, err := h.stationService.GetByID(stationID, userType)
	if err != nil {
		if err == domain.ErrStationNotFound {
			RespondWithError(c, http.StatusNotFound, "station_not_found", "Station not found")
			return
		}
		if err == domain.ErrUnauthorized {
			RespondWithError(c, http.StatusForbidden, "premium_only", "This station is only available for premium users")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch station")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": gin.H{
			"id":              station.ID,
			"name":            station.Name,
			"stream_url":      station.StreamURL,
			"image_url":       station.ImageURL,
			"tags":            station.Tags,
			"country":         station.Country,
			"votes":           station.Votes,
			"is_premium_only": station.IsPremiumOnly,
		},
	})
}

// GetPopular returns popular stations
// @Summary Obtener estaciones populares
// @Description Lista de estaciones de radio populares con filtros opcionales
// @Tags Stations
// @Accept json
// @Produce json
// @Param limit query int false "Número máximo de estaciones" default(20)
// @Param country query string false "Filtrar por código de país"
// @Success 200 {object} map[string]interface{} "Lista de estaciones populares"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /stations/popular [get]
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
// @Summary Buscar estaciones
// @Description Busca estaciones de radio por nombre o tags
// @Tags Stations
// @Accept json
// @Produce json
// @Param q query string true "Término de búsqueda"
// @Param limit query int false "Número máximo de resultados" default(20)
// @Success 200 {object} map[string]interface{} "Resultados de búsqueda"
// @Failure 400 {object} map[string]interface{} "Parámetro de búsqueda requerido"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
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

	// Search stations
	stations, err := h.stationService.Search(query, limit, userType)
	if err != nil {
		logger.Error("search failed", "query", query, "limit", limit, "user_type", userType, "error", err)
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
