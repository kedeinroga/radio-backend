package handlers

import (
	"net/http"
	"strconv"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// SecurityHandler handles security admin endpoints
type SecurityHandler struct {
	securityService *services.SecurityService
}

// NewSecurityHandler creates a new security handler
func NewSecurityHandler(securityService *services.SecurityService) *SecurityHandler {
	return &SecurityHandler{
		securityService: securityService,
	}
}

// SecurityMetricsResponse represents the response for security metrics
type SecurityMetricsResponse struct {
	TotalLoginsToday    int64                  `json:"total_logins_today" example:"150"`
	TotalLoginsWeek     int64                  `json:"total_logins_week" example:"1240"`
	FailedAttemptsToday int64                  `json:"failed_attempts_today" example:"12"`
	FailedAttemptsWeek  int64                  `json:"failed_attempts_week" example:"85"`
	ActiveSessions      int64                  `json:"active_sessions" example:"342"`
	UniqueLocationsWeek int64                  `json:"unique_locations_week" example:"23"`
	Trends              SecurityTrendsResponse `json:"trends"`
}

// SecurityTrendsResponse represents trends in security metrics
type SecurityTrendsResponse struct {
	LoginsTrend         float64 `json:"logins_trend" example:"12.5"`
	FailedAttemptsTrend float64 `json:"failed_attempts_trend" example:"-8.3"`
}

// SecurityLogResponse represents a security log entry
type SecurityLogResponse struct {
	ID        string                 `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Timestamp string                 `json:"timestamp" example:"2024-01-15T14:30:00Z"`
	EventType string                 `json:"event_type" example:"login_failed"`
	UserID    *string                `json:"user_id,omitempty" example:"user-123"`
	Email     *string                `json:"email,omitempty" example:"user@example.com"`
	TokenID   *string                `json:"token_id,omitempty" example:"token-456"`
	SessionID *string                `json:"session_id,omitempty" example:"session-789"`
	IPAddress *string                `json:"ip_address,omitempty" example:"192.168.1.1"`
	UserAgent *string                `json:"user_agent,omitempty" example:"Mozilla/5.0..."`
	Reason    *string                `json:"reason,omitempty" example:"Invalid credentials"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityLogsResponse represents paginated security logs
type SecurityLogsResponse struct {
	Logs  []SecurityLogResponse `json:"logs"`
	Total int64                 `json:"total" example:"1523"`
	Page  int                   `json:"page" example:"1"`
	Limit int                   `json:"limit" example:"50"`
}

// GetMetrics retrieves security metrics for a given period
// @Summary Get security metrics
// @Description Get security metrics including logins, failed attempts, active sessions, and trends
// @Tags Admin Security
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "Time period (7d or 30d)" default(7d) Enums(7d, 30d)
// @Success 200 {object} SecurityMetricsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/security/metrics [get]
func (h *SecurityHandler) GetMetrics(c *gin.Context) {
	// Check if user is admin (should be done by middleware, but double-check)
	userType, exists := c.Get("user_type")
	if !exists {
		RespondWithError(c, http.StatusForbidden, "ADMIN_ACCESS_REQUIRED", "admin access required")
		return
	}

	// Type assertion and check
	ut, ok := userType.(domain.UserType)
	if !ok || ut.String() != "admin" {
		RespondWithError(c, http.StatusForbidden, "ADMIN_ACCESS_REQUIRED", "admin access required")
		return
	}

	// Get period parameter (default: 7d)
	period := c.DefaultQuery("period", "7d")

	// Validate period
	if period != "7d" && period != "30d" {
		RespondWithError(c, http.StatusBadRequest, "INVALID_PERIOD", "invalid period parameter (allowed: 7d, 30d)")
		return
	}

	// Get metrics from service
	metrics, err := h.securityService.GetMetrics(period)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "METRICS_RETRIEVAL_FAILED", "failed to retrieve security metrics")
		return
	}

	// Convert to response format
	response := SecurityMetricsResponse{
		TotalLoginsToday:    metrics.TotalLoginsToday,
		TotalLoginsWeek:     metrics.TotalLoginsWeek,
		FailedAttemptsToday: metrics.FailedAttemptsToday,
		FailedAttemptsWeek:  metrics.FailedAttemptsWeek,
		ActiveSessions:      metrics.ActiveSessions,
		UniqueLocationsWeek: metrics.UniqueLocationsWeek,
		Trends: SecurityTrendsResponse{
			LoginsTrend:         metrics.Trends.LoginsTrend,
			FailedAttemptsTrend: metrics.Trends.FailedAttemptsTrend,
		},
	}

	RespondWithSuccess(c, http.StatusOK, response)
}

// GetLogs retrieves security logs with pagination and filtering
// @Summary Get security logs
// @Description Get paginated security logs with optional filtering by event type and search query
// @Tags Admin Security
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(50) minimum(1) maximum(100)
// @Param event_type query string false "Filter by event type" Enums(login_success, login_failed, token.issued, token.validated, token.revoked, session.created, session.revoked, session.suspicious, password.reset, password.changed)
// @Param search query string false "Search in event type, IP address, reason, or email"
// @Success 200 {object} SecurityLogsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/security/logs [get]
func (h *SecurityHandler) GetLogs(c *gin.Context) {
	// Check if user is admin (should be done by middleware, but double-check)
	userType, exists := c.Get("user_type")
	if !exists {
		RespondWithError(c, http.StatusForbidden, "ADMIN_ACCESS_REQUIRED", "admin access required")
		return
	}

	// Type assertion and check
	ut, ok := userType.(domain.UserType)
	if !ok || ut.String() != "admin" {
		RespondWithError(c, http.StatusForbidden, "ADMIN_ACCESS_REQUIRED", "admin access required")
		return
	}

	// Parse query parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	eventType := c.Query("event_type")
	search := c.Query("search")

	// Build filter
	filter := &domain.SecurityLogFilter{
		Page:      page,
		Limit:     limit,
		EventType: eventType,
		Search:    search,
	}

	// Optional: Parse date range if provided
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			filter.StartDate = &startDate
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			filter.EndDate = &endDate
		}
	}

	// Get logs from service
	result, err := h.securityService.GetLogs(filter)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "LOGS_RETRIEVAL_FAILED", "failed to retrieve security logs")
		return
	}

	// Convert to response format
	logs := make([]SecurityLogResponse, 0, len(result.Logs))
	for _, log := range result.Logs {
		logs = append(logs, SecurityLogResponse{
			ID:        log.ID,
			Timestamp: log.Timestamp.Format(time.RFC3339),
			EventType: log.EventType,
			UserID:    log.UserID,
			Email:     log.Email,
			TokenID:   log.TokenID,
			SessionID: log.SessionID,
			IPAddress: log.IPAddress,
			UserAgent: log.UserAgent,
			Reason:    log.Reason,
			Metadata:  log.Metadata,
		})
	}

	response := SecurityLogsResponse{
		Logs:  logs,
		Total: result.Total,
		Page:  result.Page,
		Limit: result.Limit,
	}

	RespondWithSuccess(c, http.StatusOK, response)
}
