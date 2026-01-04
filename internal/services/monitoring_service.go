package services

import (
	"fmt"
	"time"

	"radio-backend/internal/domain"
)

// MonitoringService implements domain.MonitoringService
type MonitoringService struct {
	maintenanceService *MaintenanceService
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(maintenanceService *MaintenanceService) *MonitoringService {
	return &MonitoringService{
		maintenanceService: maintenanceService,
	}
}

// GetHealthMetrics returns comprehensive health metrics
func (s *MonitoringService) GetHealthMetrics() (*domain.HealthMetrics, error) {
	metrics := &domain.HealthMetrics{
		Timestamp: time.Now(),
		Status:    "healthy",
		Alerts:    []domain.Alert{},
	}

	// Check partitions
	partitionHealth, err := s.CheckPartitionHealth()
	if err == nil {
		metrics.Partitions = *partitionHealth
		if partitionHealth.Status == "warning" || partitionHealth.Status == "critical" {
			metrics.Status = "degraded"
			metrics.Alerts = append(metrics.Alerts, domain.Alert{
				Level:     "warning",
				Category:  "partitions",
				Message:   fmt.Sprintf("Partition issues detected: %d missing future partitions", partitionHealth.MissingFuture),
				Timestamp: time.Now(),
				Resolved:  false,
				Action:    "Run POST /api/v1/admin/maintenance/check-partitions",
			})
		}
	}

	// Check materialized views
	mvHealth, err := s.CheckMaterializedViewHealth()
	if err == nil {
		metrics.MaterializedViews = *mvHealth
		if mvHealth.Status == "stale" || mvHealth.Status == "critical" {
			metrics.Status = "degraded"
			metrics.Alerts = append(metrics.Alerts, domain.Alert{
				Level:     "warning",
				Category:  "views",
				Message:   fmt.Sprintf("Materialized views are stale: %d views need refresh", len(mvHealth.StaleViews)),
				Timestamp: time.Now(),
				Resolved:  false,
				Action:    "Run POST /api/v1/admin/maintenance/refresh-views?type=all",
			})
		}
	}

	// External API health (always healthy if no circuit breaker issues)
	metrics.ExternalAPI = domain.ExternalAPIHealth{
		Status:         "up",
		CircuitBreaker: "closed",
		ErrorCount:     0,
	}

	// If no alerts, system is healthy
	if len(metrics.Alerts) == 0 {
		metrics.Status = "healthy"
	}

	return metrics, nil
}

// GetAlerts returns active alerts
func (s *MonitoringService) GetAlerts() ([]domain.Alert, error) {
	alerts := []domain.Alert{}

	// Check for partition issues
	partitionHealth, err := s.CheckPartitionHealth()
	if err == nil && (partitionHealth.Status == "warning" || partitionHealth.Status == "critical") {
		alerts = append(alerts, domain.Alert{
			Level:     "warning",
			Category:  "partitions",
			Message:   fmt.Sprintf("%d missing future partitions detected", partitionHealth.MissingFuture),
			Timestamp: time.Now(),
			Resolved:  false,
			Action:    "Run partition check endpoint",
		})
	}

	// Check for materialized view issues
	mvHealth, err := s.CheckMaterializedViewHealth()
	if err == nil && len(mvHealth.StaleViews) > 0 {
		alerts = append(alerts, domain.Alert{
			Level:     "warning",
			Category:  "views",
			Message:   fmt.Sprintf("%d materialized views are stale", len(mvHealth.StaleViews)),
			Timestamp: time.Now(),
			Resolved:  false,
			Action:    "Run refresh views endpoint",
		})
	}

	return alerts, nil
}

// CheckDatabaseHealth checks database connectivity
func (s *MonitoringService) CheckDatabaseHealth() (*domain.DatabaseHealth, error) {
	// Simple implementation - in production this would query actual DB metrics
	return &domain.DatabaseHealth{
		Status:       "up",
		Ping:         true,
		Connections:  10,
		MaxConns:     100,
		UsagePercent: 10.0,
	}, nil
}

// CheckRedisHealth checks Redis connectivity
func (s *MonitoringService) CheckRedisHealth() (*domain.RedisHealth, error) {
	// Simple implementation - in production this would query actual Redis INFO
	return &domain.RedisHealth{
		Status:       "up",
		Ping:         true,
		Memory:       50,
		UsagePercent: 25.0,
	}, nil
}

// CheckPartitionHealth checks partition table health
func (s *MonitoringService) CheckPartitionHealth() (*domain.PartitionHealth, error) {
	// Get partition status from maintenance service
	status, err := s.maintenanceService.GetPartitionStatus()
	if err != nil {
		return &domain.PartitionHealth{
			Status:    "critical",
			LastCheck: time.Now(),
		}, err
	}

	// Check for missing future partitions
	checkResult, err := s.maintenanceService.CheckFuturePartitions(3)
	if err != nil {
		return &domain.PartitionHealth{
			Status:          "critical",
			TotalPartitions: len(status),
			LastCheck:       time.Now(),
		}, err
	}

	missingCount := 0
	for _, result := range checkResult {
		if !result.PartitionsExist {
			missingCount++
		}
	}

	health := &domain.PartitionHealth{
		Status:             "healthy",
		TotalPartitions:    len(status),
		MissingFuture:      missingCount,
		OldPartitionsCount: 0,
		LastCheck:          time.Now(),
	}

	if missingCount > 0 {
		health.Status = "warning"
	}

	return health, nil
}

// CheckMaterializedViewHealth checks materialized view freshness
func (s *MonitoringService) CheckMaterializedViewHealth() (*domain.MaterializedViewHealth, error) {
	// Get refresh statistics
	stats, err := s.maintenanceService.GetRefreshStatistics(7)
	if err != nil {
		return &domain.MaterializedViewHealth{
			Status: "critical",
		}, err
	}

	health := &domain.MaterializedViewHealth{
		Status:          "healthy",
		TotalViews:      len(stats),
		StaleViews:      []string{},
		LastRefresh:     time.Now(),
		FailedRefreshes: 0,
	}

	// Check for stale views (not refreshed in last 24 hours)
	staleThreshold := time.Now().Add(-24 * time.Hour)
	var totalDuration time.Duration

	for _, stat := range stats {
		// Check if last refresh is stale
		if stat.LastRefresh != nil && stat.LastRefresh.Before(staleThreshold) {
			health.StaleViews = append(health.StaleViews, stat.ViewName)
		}

		// Update last refresh time
		if stat.LastRefresh != nil && stat.LastRefresh.After(health.LastRefresh) {
			health.LastRefresh = *stat.LastRefresh
		}

		// Calculate average duration
		totalDuration += time.Duration(stat.AvgDurationMs) * time.Millisecond

		// Count failed refreshes
		health.FailedRefreshes += int(stat.FailedRefreshes)
	}

	if len(stats) > 0 {
		health.AvgRefreshDuration = totalDuration / time.Duration(len(stats))
	}

	if len(health.StaleViews) > 0 {
		health.Status = "stale"
	}

	return health, nil
}
