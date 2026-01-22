package domain

import "time"

// MaintenanceOperation representa una operación de mantenimiento
type MaintenanceOperation string

const (
	MaintenanceRefreshViews      MaintenanceOperation = "refresh_views"
	MaintenanceCleanupPartitions MaintenanceOperation = "cleanup_partitions"
	MaintenanceCheckPartitions   MaintenanceOperation = "check_partitions"
	MaintenancePartitionStatus   MaintenanceOperation = "partition_status"
	MaintenanceFullMaintenance   MaintenanceOperation = "full_maintenance"
)

// RefreshResult representa el resultado de refrescar una vista materializada
type RefreshResult struct {
	ViewName     string    `json:"view_name"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	DurationMs   int64     `json:"duration_ms"`
	RowsAffected int64     `json:"rows_affected"`
	Status       string    `json:"status"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}

// PartitionCleanupResult representa el resultado de limpiar particiones
type PartitionCleanupResult struct {
	TableName     string    `json:"table_name"`
	PartitionName string    `json:"partition_name"`
	PartitionDate time.Time `json:"partition_date"`
	Dropped       bool      `json:"dropped"`
	Message       string    `json:"message"`
}

// PartitionCheckResult representa el resultado de verificar particiones
type PartitionCheckResult struct {
	TableName       string    `json:"table_name"`
	MonthDate       time.Time `json:"month_date"`
	PartitionName   string    `json:"partition_name"`
	PartitionsExist bool      `json:"partitions_exist"`
	Message         string    `json:"message"`
}

// PartitionStatusResult representa el estado de una partición
type PartitionStatusResult struct {
	PartitionName string    `json:"partition_name"`
	TableName     string    `json:"table_name"`
	PartitionDate time.Time `json:"partition_date"`
	RowCount      int64     `json:"row_count"`
	SizeMB        float64   `json:"size_mb"`
	IndexSizeMB   float64   `json:"index_size_mb"`
	TotalSizeMB   float64   `json:"total_size_mb"`
}

// RefreshStatistics representa las estadísticas de refresh de vistas
type RefreshStatistics struct {
	ViewName            string     `json:"view_name"`
	TotalRefreshes      int64      `json:"total_refreshes"`
	SuccessfulRefreshes int64      `json:"successful_refreshes"`
	FailedRefreshes     int64      `json:"failed_refreshes"`
	AvgDurationMs       float64    `json:"avg_duration_ms"`
	MaxDurationMs       int64      `json:"max_duration_ms"`
	MinDurationMs       int64      `json:"min_duration_ms"`
	LastRefresh         *time.Time `json:"last_refresh"`
	LastStatus          string     `json:"last_status"`
}

// MaintenanceRecommendation representa una recomendación de mantenimiento
type MaintenanceRecommendation struct {
	Operation MaintenanceOperation `json:"operation"`
	Priority  string               `json:"priority"` // "critical", "warning", "info"
	Reason    string               `json:"reason"`
	ShouldRun bool                 `json:"should_run"`
	LastRunAt *time.Time           `json:"last_run_at,omitempty"`
	NextRunAt *time.Time           `json:"next_run_at,omitempty"`
}

// MaintenanceRepository define las operaciones de mantenimiento
type MaintenanceRepository interface {
	// Refresh de vistas materializadas
	RefreshAllViews() ([]RefreshResult, error)
	RefreshSEOViews() ([]RefreshResult, error)
	RefreshAnalyticsViews() ([]RefreshResult, error)
	GetRefreshStatistics(daysBack int) ([]RefreshStatistics, error)

	// Gestión de particiones
	CleanupOldPartitions(retentionMonths int) ([]PartitionCleanupResult, error)
	CheckFuturePartitions(monthsAhead int) ([]PartitionCheckResult, error)
	GetPartitionStatus() ([]PartitionStatusResult, error)

	// Recomendaciones
	GetMaintenanceRecommendations() ([]MaintenanceRecommendation, error)
}
