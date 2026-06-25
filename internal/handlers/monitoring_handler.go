package handlers

import (
	"net/http"

	"radio-backend/internal/domain"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// MonitoringHandler handles monitoring endpoints
type MonitoringHandler struct {
	monitoringService *services.MonitoringService
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(monitoringService *services.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoringService,
	}
}

// AlertsResponse is the response for the system alerts endpoint.
type AlertsResponse struct {
	Alerts      []domain.Alert `json:"alerts"`
	AlertCount  int            `json:"alert_count" example:"2"`
	HasCritical bool           `json:"has_critical" example:"false"`
	HasWarning  bool           `json:"has_warning" example:"true"`
}

// GetHealthMetrics returns comprehensive health metrics
// @Summary Get health metrics
// @Description Returns comprehensive system health metrics including database, Redis, partitions, and materialized views
// @Description This endpoint provides detailed health information for monitoring and alerting
// @Tags Monitoring
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.HealthMetrics "Health metrics retrieved successfully"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Failed to get health metrics"
// @Router /admin/monitoring/health [get]
func (h *MonitoringHandler) GetHealthMetrics(c *gin.Context) {
	metrics, err := h.monitoringService.GetHealthMetrics()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "health_check_failed", "Failed to get health metrics")
		return
	}

	RespondWithSuccess(c, http.StatusOK, metrics)
}

// GetAlerts returns active system alerts
// @Summary Get system alerts
// @Description Returns list of active system alerts based on health checks
// @Description Alerts include partition issues, stale materialized views, and other problems
// @Tags Monitoring
// @Produce json
// @Security BearerAuth
// @Success 200 {object} AlertsResponse "Alerts retrieved successfully"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Failed to get alerts"
// @Router /admin/monitoring/alerts [get]
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	alerts, err := h.monitoringService.GetAlerts()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "alerts_check_failed", "Failed to get alerts")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"alerts":       alerts,
		"alert_count":  len(alerts),
		"has_critical": hasAlertLevel(alerts, "critical"),
		"has_warning":  hasAlertLevel(alerts, "warning"),
	})
}

// hasAlertLevel checks if alerts contain a specific level
func hasAlertLevel(alerts []domain.Alert, level string) bool {
	for _, alert := range alerts {
		if alert.Level == level {
			return true
		}
	}
	return false
}
