package handlers

import (
	"net/http"
	"strconv"

	"radio-backend/internal/domain"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// MaintenanceHandler handles maintenance endpoints (admin only)
type MaintenanceHandler struct {
	maintenanceService *services.MaintenanceService
}

// NewMaintenanceHandler creates a new maintenance handler
func NewMaintenanceHandler(maintenanceService *services.MaintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{maintenanceService: maintenanceService}
}

// RecommendationsResponse is the response for maintenance recommendations.
type RecommendationsResponse struct {
	Recommendations []domain.MaintenanceRecommendation `json:"recommendations"`
	Total           int                                `json:"total" example:"3"`
}

// RefreshViewsResponse is the response for the refresh-views operation.
type RefreshViewsResponse struct {
	Results []domain.RefreshResult `json:"results"`
	Type    string                 `json:"type" example:"all"`
}

// RefreshStatisticsResponse is the response for refresh statistics.
type RefreshStatisticsResponse struct {
	Statistics []domain.RefreshStatistics `json:"statistics"`
	DaysBack   int                        `json:"days_back" example:"7"`
}

// CleanupPartitionsResponse is the response for partition cleanup.
type CleanupPartitionsResponse struct {
	Results         []domain.PartitionCleanupResult `json:"results"`
	TotalDropped    int                             `json:"total_dropped" example:"3"`
	RetentionMonths int                             `json:"retention_months" example:"12"`
}

// CheckPartitionsResponse is the response for the future-partition check.
type CheckPartitionsResponse struct {
	Results     []domain.PartitionCheckResult `json:"results"`
	HasMissing  bool                          `json:"has_missing" example:"false"`
	MonthsAhead int                           `json:"months_ahead" example:"3"`
}

// PartitionStatusResponse is the response for partition status.
type PartitionStatusResponse struct {
	Partitions  []domain.PartitionStatusResult `json:"partitions"`
	Total       int                            `json:"total" example:"18"`
	TotalRows   int64                          `json:"total_rows" example:"154200"`
	TotalSizeMB float64                        `json:"total_size_mb" example:"42.5"`
}

// GetRecommendations returns maintenance recommendations
// @Summary Get maintenance recommendations
// @Description Returns recommendations for database maintenance operations that should be executed. Shows priority levels and reasons.
// @Tags Maintenance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} RecommendationsResponse "Maintenance recommendations"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/maintenance/recommendations [get]
func (h *MaintenanceHandler) GetRecommendations(c *gin.Context) {
	recommendations, err := h.maintenanceService.GetRecommendations()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "maintenance_failed", "Failed to get maintenance recommendations")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"recommendations": recommendations,
		"total":           len(recommendations),
	})
}

// RefreshViews refreshes materialized views
// @Summary Refresh materialized views
// @Description Refreshes materialized views used for SEO and analytics. Can refresh all views or specific types (seo/analytics).
// @Tags Maintenance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type query string false "Type of views to refresh: all, seo, analytics" default(all) Enums(all, seo, analytics)
// @Success 200 {object} RefreshViewsResponse "Refresh results"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/maintenance/refresh-views [post]
func (h *MaintenanceHandler) RefreshViews(c *gin.Context) {
	viewType := c.DefaultQuery("type", "all")

	var results interface{}
	var err error

	switch viewType {
	case "seo":
		results, err = h.maintenanceService.RefreshSEOViews()
	case "analytics":
		results, err = h.maintenanceService.RefreshAnalyticsViews()
	default:
		results, err = h.maintenanceService.RefreshAllViews()
	}

	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "refresh_failed", "Failed to refresh views")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"results": results,
		"type":    viewType,
	})
}

// GetRefreshStatistics returns refresh statistics
// @Summary Get refresh statistics
// @Description Returns statistics about materialized view refreshes including duration, success rate, and last refresh time.
// @Tags Maintenance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param days query int false "Number of days to look back" default(7) minimum(1) maximum(90)
// @Success 200 {object} RefreshStatisticsResponse "Refresh statistics"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/maintenance/refresh-stats [get]
func (h *MaintenanceHandler) GetRefreshStatistics(c *gin.Context) {
	daysBack := parseIntQueryParam(c, "days", 7)

	stats, err := h.maintenanceService.GetRefreshStatistics(daysBack)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "stats_failed", "Failed to get refresh statistics")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"statistics": stats,
		"days_back":  daysBack,
	})
}

// CleanupPartitions cleans up old partitions
// @Summary Cleanup old partitions
// @Description Removes partitions older than specified retention period to free up disk space.
// @Tags Maintenance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param retention_months query int false "Number of months to retain" default(12) minimum(1) maximum(36)
// @Success 200 {object} CleanupPartitionsResponse "Cleanup results"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/maintenance/cleanup-partitions [post]
func (h *MaintenanceHandler) CleanupPartitions(c *gin.Context) {
	retentionMonths := parseIntQueryParam(c, "retention_months", 12)

	results, err := h.maintenanceService.CleanupOldPartitions(retentionMonths)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "cleanup_failed", "Failed to cleanup partitions")
		return
	}

	// Contar cuántas se eliminaron
	droppedCount := 0
	for _, r := range results {
		if r.Dropped {
			droppedCount++
		}
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"results":          results,
		"total_dropped":    droppedCount,
		"retention_months": retentionMonths,
	})
}

// CheckPartitions checks for missing future partitions
// @Summary Check future partitions
// @Description Verifies that partitions exist for future months to prevent insertion failures.
// @Tags Maintenance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param months_ahead query int false "Number of months ahead to check" default(3) minimum(1) maximum(12)
// @Success 200 {object} CheckPartitionsResponse "Check results"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/maintenance/check-partitions [get]
func (h *MaintenanceHandler) CheckPartitions(c *gin.Context) {
	monthsAhead := parseIntQueryParam(c, "months_ahead", 3)

	results, err := h.maintenanceService.CheckFuturePartitions(monthsAhead)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "check_failed", "Failed to check partitions")
		return
	}

	// Verificar si hay particiones faltantes
	hasMissing := false
	for _, r := range results {
		if !r.PartitionsExist {
			hasMissing = true
			break
		}
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"results":      results,
		"has_missing":  hasMissing,
		"months_ahead": monthsAhead,
	})
}

// GetPartitionStatus returns partition status
// @Summary Get partition status
// @Description Returns detailed status of all partitions including size, row count, and date ranges.
// @Tags Maintenance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PartitionStatusResponse "Partition status"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/maintenance/partition-status [get]
func (h *MaintenanceHandler) GetPartitionStatus(c *gin.Context) {
	results, err := h.maintenanceService.GetPartitionStatus()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "status_failed", "Failed to get partition status")
		return
	}

	// Calcular totales
	var totalRows int64
	var totalSizeMB float64
	for _, r := range results {
		totalRows += r.RowCount
		totalSizeMB += r.TotalSizeMB
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"partitions":    results,
		"total":         len(results),
		"total_rows":    totalRows,
		"total_size_mb": totalSizeMB,
	})
}

// PerformFullMaintenance performs full maintenance
// @Summary Perform full maintenance
// @Description Executes a complete maintenance routine including view refresh, partition check, and status report.
// @Tags Maintenance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Full maintenance results (dynamic map)"
// @Failure 401 {object} SimpleErrorResponse "Invalid or missing authentication token"
// @Failure 403 {object} SimpleErrorResponse "Access denied - Admin users only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/maintenance/full [post]
func (h *MaintenanceHandler) PerformFullMaintenance(c *gin.Context) {
	result, err := h.maintenanceService.PerformFullMaintenance()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "maintenance_failed", "Failed to perform full maintenance")
		return
	}

	RespondWithSuccess(c, http.StatusOK, result)
}

// parseIntQueryParam helper to parse integer query parameters
func parseIntQueryParam(c *gin.Context, key string, defaultValue int) int {
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
