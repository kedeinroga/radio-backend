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
// @Description Retorna las estaciones más reproducidas en un periodo de tiempo específico ordenadas por número de reproducciones. Incluye información completa de cada estación.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Rango de tiempo: hour, day, week, month" default(day) Enums(hour, day, week, month)
// @Param limit query int false "Número máximo de resultados a retornar" default(10) minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{} "Estadísticas de estaciones populares" example({"success":true,"data":[{"station_id":"abc123","name":"Rock FM","country":"USA","plays":1520,"favicon":"https://...","url":"https://..."}]})
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
// @Summary Búsquedas en tendencia
// @Description Retorna los términos de búsqueda más frecuentes en un periodo de tiempo específico ordenadas por frecuencia. Incluye contador absoluto y porcentaje del total.
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Rango de tiempo: hour, day, week, month" default(day) Enums(hour, day, week, month)
// @Param limit query int false "Número máximo de resultados a retornar" default(10) minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{} "Estadísticas de búsquedas en tendencia" example({"success":true,"data":[{"search_term":"rock","count":456,"percentage":12.5}]})
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
