package handlers

import (
	"net/http"
	"strconv"

	_ "radio-backend/internal/domain" // imported for swagger documentation
	"radio-backend/internal/infrastructure/logger"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// SEOHandler handles public SEO endpoints
type SEOHandler struct {
	seoService *services.SEOService
}

// NewSEOHandler creates a new SEO handler instance
func NewSEOHandler(seoService *services.SEOService) *SEOHandler {
	return &SEOHandler{
		seoService: seoService,
	}
}

// GetSitemapData returns data for generating dynamic sitemap.xml
// @Summary Data for generating sitemap
// @Description Returns popular tags and countries to build dynamic sitemap.xml. Data is cached for 6 hours.
// @Tags SEO
// @Produce json
// @Success 200 {object} domain.SitemapData "Sitemap data"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /seo/sitemap-data [get]
func (h *SEOHandler) GetSitemapData(c *gin.Context) {
	logger.Info("fetching sitemap data")

	data, err := h.seoService.GetSitemapData(c.Request.Context())
	if err != nil {
		logger.Error("failed to fetch sitemap data", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "seo_error", "Failed to fetch sitemap data")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": data,
	})
}

// GetPopularTags returns only the most popular tags
// @Summary Top popular tags/genres
// @Description List of tags with most active stations (useful for sitemap and navigation). Maximum 100 tags.
// @Tags SEO
// @Produce json
// @Param limit query int false "Results limit" default(100) minimum(1) maximum(100)
// @Success 200 {object} object{data=object{tags=[]domain.PopularTag,count=int,total=int}} "List of popular tags"
// @Failure 400 {object} map[string]interface{} "Invalid parameter"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /seo/popular-tags [get]
func (h *SEOHandler) GetPopularTags(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		RespondWithError(c, http.StatusBadRequest, "invalid_limit", "Limit must be a positive integer")
		return
	}

	// Limit to maximum 100
	if limit > 100 {
		limit = 100
	}

	logger.Info("fetching popular tags", "limit", limit)

	data, err := h.seoService.GetSitemapData(c.Request.Context())
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

// GetPopularCountries returns only the most popular countries
// @Summary Top popular countries
// @Description List of countries with most active stations (useful for sitemap and navigation). Maximum 50 countries.
// @Tags SEO
// @Produce json
// @Param limit query int false "Results limit" default(50) minimum(1) maximum(50)
// @Success 200 {object} object{data=object{countries=[]domain.PopularCountry,count=int,total=int}} "List of popular countries"
// @Failure 400 {object} map[string]interface{} "Invalid parameter"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /seo/popular-countries [get]
func (h *SEOHandler) GetPopularCountries(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		RespondWithError(c, http.StatusBadRequest, "invalid_limit", "Limit must be a positive integer")
		return
	}

	// Limit to maximum 50
	if limit > 50 {
		limit = 50
	}

	logger.Info("fetching popular countries", "limit", limit)

	data, err := h.seoService.GetSitemapData(c.Request.Context())
	if err != nil {
		logger.Error("failed to fetch countries", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "seo_error", "Failed to fetch countries")
		return
	}

	// Return only limited countries
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

// RefreshSEOStats manually refreshes SEO statistics
// @Summary Refresh SEO statistics (Admin)
// @Description Updates tag and country statistics from the database. Administrative endpoint that requires admin authentication.
// @Tags SEO
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Statistics updated"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied - Admin only"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/seo/refresh-stats [post]
func (h *SEOHandler) RefreshSEOStats(c *gin.Context) {
	logger.Info("manually refreshing SEO statistics")

	err := h.seoService.RefreshSEOStats(c.Request.Context())
	if err != nil {
		logger.Error("failed to refresh SEO stats", "error", err)
		RespondWithError(c, http.StatusInternalServerError, "refresh_failed", "Failed to refresh SEO statistics")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "SEO statistics refreshed successfully",
	})
}
