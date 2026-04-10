package domain

import "time"

// SecurityMetrics represents security metrics for a given period
type SecurityMetrics struct {
	TotalLoginsToday    int64          `json:"total_logins_today"`
	TotalLoginsWeek     int64          `json:"total_logins_week"`
	FailedAttemptsToday int64          `json:"failed_attempts_today"`
	FailedAttemptsWeek  int64          `json:"failed_attempts_week"`
	ActiveSessions      int64          `json:"active_sessions"`
	UniqueLocationsWeek int64          `json:"unique_locations_week"`
	Trends              SecurityTrends `json:"trends"`
}

// SecurityTrends represents trends in security metrics
type SecurityTrends struct {
	LoginsTrend         float64 `json:"logins_trend"`          // Percentage change from previous period
	FailedAttemptsTrend float64 `json:"failed_attempts_trend"` // Percentage change from previous period
}

// SecurityLog represents a security event log entry
type SecurityLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	UserID    *string                `json:"user_id,omitempty"`
	Email     *string                `json:"email,omitempty"`
	TokenID   *string                `json:"token_id,omitempty"`
	SessionID *string                `json:"session_id,omitempty"`
	IPAddress *string                `json:"ip_address,omitempty"`
	UserAgent *string                `json:"user_agent,omitempty"`
	Reason    *string                `json:"reason,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityLogFilter represents filters for security logs
type SecurityLogFilter struct {
	Page      int
	Limit     int
	EventType string
	Search    string
	StartDate *time.Time
	EndDate   *time.Time
}

// SecurityLogResult represents paginated security logs
type SecurityLogResult struct {
	Logs  []*SecurityLog `json:"logs"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// SuspiciousSourceStats aggregates request-source anomaly data for a period.
type SuspiciousSourceStats struct {
	Period     string        `json:"period"`
	TotalCount int64         `json:"total_count"`
	BySouce    []SourceCount `json:"by_source"`
	TopIPs     []IPCount     `json:"top_ips"`
	TopPaths   []PathCount   `json:"top_paths"`
}

// SourceCount is a breakdown of suspicious events by classified source.
type SourceCount struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}

// IPCount is a breakdown of suspicious events by originating IP.
type IPCount struct {
	IP       string `json:"ip"`
	Count    int64  `json:"count"`
	LastSeen string `json:"last_seen"`
}

// PathCount is a breakdown of suspicious events by targeted path.
type PathCount struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}

// SecurityRepository defines the interface for security data access
type SecurityRepository interface {
	// GetMetrics retrieves security metrics for a given period
	GetMetrics(period string) (*SecurityMetrics, error)

	// GetLogs retrieves security logs with pagination and filtering
	GetLogs(filter *SecurityLogFilter) (*SecurityLogResult, error)

	// LogSecurityEvent logs a security event
	LogSecurityEvent(event *SecurityEvent) error

	// GetSuspiciousSourceStats returns aggregated stats for suspicious_request_source events.
	GetSuspiciousSourceStats(period string) (*SuspiciousSourceStats, error)
}
