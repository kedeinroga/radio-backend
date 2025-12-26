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
// @Summary Estadísticas de estaciones populares
// @Description Retorna las estaciones más reproducidas en un periodo de tiempo específico. Incluye conteo de reproducciones y duración total de escucha.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Rango de tiempo: hour, day, week, month" default(day) Enums(hour, day, week, month)
// @Param limit query int false "Número máximo de resultados a retornar" default(10) minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{} "Estadísticas de estaciones populares" example({"data":[{"station_id":"uuid-123","play_count":450,"duration_ms":125000}],"meta":{"count":10,"range":"day"}})
// @Failure 401 {object} map[string]interface{} "Token de autenticación inválido o ausente" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Acceso denegado - Solo usuarios Admin" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Error interno del servidor" example({"error":{"code":"fetch_failed","message":"Failed to fetch popular stations"}})
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
// @Summary Búsquedas en tendencia
// @Description Retorna las consultas de búsqueda más frecuentes en un periodo de tiempo específico. Incluye conteo de búsquedas y promedio de resultados.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Rango de tiempo: hour, day, week, month" default(day) Enums(hour, day, week, month)
// @Param limit query int false "Número máximo de resultados a retornar" default(10) minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{} "Estadísticas de búsquedas en tendencia" example({"data":[{"query":"rock","search_count":320,"avg_results":15.5}],"meta":{"count":10,"range":"day"}})
// @Failure 401 {object} map[string]interface{} "Token de autenticación inválido o ausente" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Acceso denegado - Solo usuarios Admin" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Error interno del servidor" example({"error":{"code":"fetch_failed","message":"Failed to fetch trending searches"}})
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
// @Summary Usuarios activos
// @Description Retorna el número de usuarios autenticados activos en las últimas 24 horas. Los usuarios activos son aquellos que han realizado al menos una petición autenticada en el periodo especificado.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Respuesta exitosa" example({"success":true,"data":{"count":1234}})
// @Failure 401 {object} map[string]interface{} "Token de autenticación inválido o ausente" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Acceso denegado - Solo usuarios Admin" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Error interno del servidor" example({"error":{"code":"fetch_failed","message":"Failed to fetch active users count"}})
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

// GetGuestUsers returns guest users count
// @Summary Usuarios guest activos
// @Description Retorna el número de usuarios guest (no autenticados) activos en las últimas 24 horas. Los usuarios guest se identifican por su dirección IP única y representan usuarios que usan la aplicación sin registrarse.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Respuesta exitosa" example({"success":true,"data":{"count":856}})
// @Failure 401 {object} map[string]interface{} "Token de autenticación inválido o ausente" example({"error":{"code":"unauthorized","message":"invalid or expired token"}})
// @Failure 403 {object} map[string]interface{} "Acceso denegado - Solo usuarios Admin" example({"error":{"code":"forbidden","message":"admin access required"}})
// @Failure 500 {object} map[string]interface{} "Error interno del servidor" example({"error":{"code":"fetch_failed","message":"Failed to fetch guest users count"}})
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
