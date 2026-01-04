package domain

import "time"

// HealthMetrics represents system health metrics
type HealthMetrics struct {
	Timestamp            time.Time              `json:"timestamp"`
	Status               string                 `json:"status"` // healthy, degraded, unhealthy
	Database             DatabaseHealth         `json:"database"`
	Redis                RedisHealth            `json:"redis"`
	ExternalAPI          ExternalAPIHealth      `json:"external_api"`
	Partitions           PartitionHealth        `json:"partitions"`
	MaterializedViews    MaterializedViewHealth `json:"materialized_views"`
	Alerts               []Alert                `json:"alerts,omitempty"`
	PerformanceMetrics   PerformanceMetrics     `json:"performance_metrics"`
}

// DatabaseHealth represents database health status
type DatabaseHealth struct {
	Status      string  `json:"status"` // up, down
	Ping        bool    `json:"ping"`
	Connections int     `json:"connections"`
	MaxConns    int     `json:"max_connections"`
	UsagePercent float64 `json:"usage_percent"`
}

// RedisHealth represents Redis health status
type RedisHealth struct {
	Status       string  `json:"status"` // up, down
	Ping         bool    `json:"ping"`
	Memory       int64   `json:"memory_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

// ExternalAPIHealth represents external API health
type ExternalAPIHealth struct {
	Status          string        `json:"status"` // up, down, degraded
	CircuitBreaker  string        `json:"circuit_breaker"` // closed, open, half-open
	LastError       string        `json:"last_error,omitempty"`
	ErrorCount      int           `json:"error_count"`
	AvgResponseTime time.Duration `json:"avg_response_time_ms"`
}

// PartitionHealth represents partition table health
type PartitionHealth struct {
	Status              string `json:"status"` // healthy, warning, critical
	TotalPartitions     int    `json:"total_partitions"`
	MissingFuture       int    `json:"missing_future_partitions"`
	OldPartitionsCount  int    `json:"old_partitions_count"`
	LastCheck           time.Time `json:"last_check"`
}

// MaterializedViewHealth represents materialized view health
type MaterializedViewHealth struct {
	Status              string        `json:"status"` // healthy, stale, critical
	TotalViews          int           `json:"total_views"`
	StaleViews          []string      `json:"stale_views,omitempty"`
	LastRefresh         time.Time     `json:"last_refresh"`
	AvgRefreshDuration  time.Duration `json:"avg_refresh_duration_ms"`
	FailedRefreshes     int           `json:"failed_refreshes"`
}

// PerformanceMetrics represents system performance metrics
type PerformanceMetrics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	AvgResponseTime   float64 `json:"avg_response_time_ms"`
	ErrorRate         float64 `json:"error_rate_percent"`
	CacheHitRate      float64 `json:"cache_hit_rate_percent"`
}

// Alert represents a system alert
type Alert struct {
	Level       string    `json:"level"` // info, warning, critical
	Category    string    `json:"category"` // database, redis, api, partitions, views, performance
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
	Action      string    `json:"action,omitempty"`
}

// MonitoringService defines the interface for monitoring operations
type MonitoringService interface {
	GetHealthMetrics() (*HealthMetrics, error)
	GetAlerts() ([]Alert, error)
	CheckDatabaseHealth() (*DatabaseHealth, error)
	CheckRedisHealth() (*RedisHealth, error)
	CheckPartitionHealth() (*PartitionHealth, error)
	CheckMaterializedViewHealth() (*MaterializedViewHealth, error)
}
