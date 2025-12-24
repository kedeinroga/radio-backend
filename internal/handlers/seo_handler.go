package handlers

import (
	"net/http"
	"strconv"

	_ "radio-backend/internal/domain" // imported for swagger documentation
	"radio-backend/internal/infrastructure/logger"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// SEOHandler maneja los endpoints públicos de SEO
type SEOHandler struct {
	seoService *services.SEOService
}

// NewSEOHandler crea una nueva instancia del handler SEO
func NewSEOHandler(seoService *services.SEOService) *SEOHandler {
	return &SEOHandler{
		seoService: seoService,
	}
}

// GetSitemapData retorna datos para generar sitemap.xml dinámico
// @Summary Datos para generar sitemap
// @Description Retorna tags y países populares para construir sitemap.xml dinámico. Los datos se cachean por 6 horas.
// @Tags SEO
// @Produce json
// @Success 200 {object} domain.SitemapData "Datos del sitemap"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /seo/sitemap-data [get]
func (h *SEOHandler) GetSitemapData(c *gin.Context) {
	logger.Info("fetching sitemap data")

	data, err := h.seoService.GetSitemapData()
	if err != nil {
		logger.Error("failed to fetch sitemap data", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "seo_error", "Failed to fetch sitemap data")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": data,
	})
}

// GetPopularTags retorna solo los tags más populares
// @Summary Top tags/géneros populares
// @Description Lista de tags con más estaciones activas (útil para sitemap y navegación). Máximo 100 tags.
// @Tags SEO
// @Produce json
// @Param limit query int false "Límite de resultados" default(100) minimum(1) maximum(100)
// @Success 200 {object} object{data=object{tags=[]domain.PopularTag,count=int,total=int}} "Lista de tags populares"
// @Failure 400 {object} map[string]interface{} "Parámetro inválido"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /seo/popular-tags [get]
func (h *SEOHandler) GetPopularTags(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		RespondWithError(c, http.StatusBadRequest, "invalid_limit", "Limit must be a positive integer")
		return
	}

	// Limitar a máximo 100
	if limit > 100 {
		limit = 100
	}

	logger.Info("fetching popular tags", "limit", limit)

	data, err := h.seoService.GetSitemapData()
	if err != nil {
		logger.Error("failed to fetch tags", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "seo_error", "Failed to fetch tags")
		return
	}

	// Devolver solo los tags limitados
	tags := data.PopularTags
	if len(tags) > limit {
		tags = tags[:limit]
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": gin.H{
			"tags":  tags,
			"count": len(tags),
			"total": len(data.PopularTags),
		},
	})
}

// GetPopularCountries retorna solo los países más populares
// @Summary Top países populares
// @Description Lista de países con más estaciones activas (útil para sitemap y navegación). Máximo 50 países.
// @Tags SEO
// @Produce json
// @Param limit query int false "Límite de resultados" default(50) minimum(1) maximum(50)
// @Success 200 {object} object{data=object{countries=[]domain.PopularCountry,count=int,total=int}} "Lista de países populares"
// @Failure 400 {object} map[string]interface{} "Parámetro inválido"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /seo/popular-countries [get]
func (h *SEOHandler) GetPopularCountries(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		RespondWithError(c, http.StatusBadRequest, "invalid_limit", "Limit must be a positive integer")
		return
	}

	// Limitar a máximo 50
	if limit > 50 {
		limit = 50
	}

	logger.Info("fetching popular countries", "limit", limit)

	data, err := h.seoService.GetSitemapData()
	if err != nil {
		logger.Error("failed to fetch countries", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "seo_error", "Failed to fetch countries")
		return
	}

	// Devolver solo los países limitados
	countries := data.PopularCountries
	if len(countries) > limit {
		countries = countries[:limit]
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": gin.H{
			"countries": countries,
			"count":     len(countries),
			"total":     len(data.PopularCountries),
		},
	})
}

// RefreshSEOStats refresca manualmente las estadísticas SEO
// @Summary Refrescar estadísticas SEO (Admin)
// @Description Actualiza las estadísticas de tags y países desde la base de datos. Endpoint administrativo.
// @Tags SEO
// @Produce json
// @Success 200 {object} map[string]interface{} "Estadísticas actualizadas"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /seo/refresh-stats [post]
func (h *SEOHandler) RefreshSEOStats(c *gin.Context) {
	logger.Info("manually refreshing SEO statistics")

	err := h.seoService.RefreshSEOStats()
	if err != nil {
		logger.Error("failed to refresh SEO stats", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "refresh_failed", "Failed to refresh SEO statistics")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "SEO statistics refreshed successfully",
	})
}
