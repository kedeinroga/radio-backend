package postgres

import (
	"database/sql"
	"radio-backend/internal/domain"
	"time"
)

// MaintenanceRepository implementa domain.MaintenanceRepository
type MaintenanceRepository struct {
	db *sql.DB
}

// NewMaintenanceRepository crea un nuevo repositorio de mantenimiento
func NewMaintenanceRepository(db *sql.DB) *MaintenanceRepository {
	return &MaintenanceRepository{db: db}
}

// RefreshAllViews refresca todas las vistas materializadas
func (r *MaintenanceRepository) RefreshAllViews() ([]domain.RefreshResult, error) {
	query := `SELECT * FROM refresh_all_views_with_logging()`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.RefreshResult
	for rows.Next() {
		var result domain.RefreshResult
		err := rows.Scan(
			&result.ViewName,
			&result.DurationMs,
			&result.RowsAffected,
			&result.Status,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

// RefreshSEOViews refresca solo las vistas de SEO
func (r *MaintenanceRepository) RefreshSEOViews() ([]domain.RefreshResult, error) {
	query := `
		WITH refresh_log AS (
			SELECT * FROM refresh_all_seo_views()
		)
		INSERT INTO materialized_view_refresh_log (
			view_name, refresh_started_at, refresh_completed_at,
			duration_ms, rows_affected, status, error_message
		)
		SELECT
			view_name,
			NOW() - (duration_ms || ' milliseconds')::INTERVAL,
			NOW(),
			duration_ms,
			rows_affected,
			status,
			error_message
		FROM refresh_log
		RETURNING view_name, duration_ms, rows_affected, status
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.RefreshResult
	for rows.Next() {
		var result domain.RefreshResult
		err := rows.Scan(
			&result.ViewName,
			&result.DurationMs,
			&result.RowsAffected,
			&result.Status,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

// RefreshAnalyticsViews refresca solo las vistas de analytics
func (r *MaintenanceRepository) RefreshAnalyticsViews() ([]domain.RefreshResult, error) {
	query := `
		WITH refresh_log AS (
			SELECT * FROM refresh_all_analytics_views()
		)
		INSERT INTO materialized_view_refresh_log (
			view_name, refresh_started_at, refresh_completed_at,
			duration_ms, rows_affected, status, error_message
		)
		SELECT
			view_name,
			NOW() - (duration_ms || ' milliseconds')::INTERVAL,
			NOW(),
			duration_ms,
			rows_affected,
			status,
			error_message
		FROM refresh_log
		RETURNING view_name, duration_ms, rows_affected, status
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.RefreshResult
	for rows.Next() {
		var result domain.RefreshResult
		err := rows.Scan(
			&result.ViewName,
			&result.DurationMs,
			&result.RowsAffected,
			&result.Status,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

// GetRefreshStatistics obtiene las estadísticas de refresh
func (r *MaintenanceRepository) GetRefreshStatistics(daysBack int) ([]domain.RefreshStatistics, error) {
	query := `SELECT * FROM get_refresh_statistics($1)`

	rows, err := r.db.Query(query, daysBack)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.RefreshStatistics
	for rows.Next() {
		var stat domain.RefreshStatistics
		err := rows.Scan(
			&stat.ViewName,
			&stat.TotalRefreshes,
			&stat.SuccessfulRefreshes,
			&stat.FailedRefreshes,
			&stat.AvgDurationMs,
			&stat.MaxDurationMs,
			&stat.MinDurationMs,
			&stat.LastRefresh,
			&stat.LastStatus,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// CleanupOldPartitions limpia particiones antiguas
func (r *MaintenanceRepository) CleanupOldPartitions(retentionMonths int) ([]domain.PartitionCleanupResult, error) {
	query := `SELECT * FROM cleanup_old_partitions($1)`

	rows, err := r.db.Query(query, retentionMonths)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]domain.PartitionCleanupResult, 0)
	for rows.Next() {
		var result domain.PartitionCleanupResult
		var action string
		// cleanup_old_partitions returns (partition_name, partition_date, action)
		err := rows.Scan(
			&result.PartitionName,
			&result.PartitionDate,
			&action,
		)
		if err != nil {
			return nil, err
		}
		result.Dropped = action == "DROPPED"
		result.Message = action
		results = append(results, result)
	}

	return results, rows.Err()
}

// CheckFuturePartitions verifica que existan particiones futuras
func (r *MaintenanceRepository) CheckFuturePartitions(monthsAhead int) ([]domain.PartitionCheckResult, error) {
	query := `SELECT * FROM check_missing_partitions($1)`

	rows, err := r.db.Query(query, monthsAhead)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.PartitionCheckResult
	for rows.Next() {
		var result domain.PartitionCheckResult
		err := rows.Scan(
			&result.TableName,
			&result.MonthDate,
			&result.PartitionName,
			&result.PartitionsExist,
			&result.Message,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

// GetPartitionStatus obtiene el estado de todas las particiones
func (r *MaintenanceRepository) GetPartitionStatus() ([]domain.PartitionStatusResult, error) {
	query := `SELECT * FROM list_partition_status() ORDER BY partition_date DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.PartitionStatusResult
	for rows.Next() {
		var result domain.PartitionStatusResult
		err := rows.Scan(
			&result.PartitionName,
			&result.TableName,
			&result.PartitionDate,
			&result.RowCount,
			&result.SizeMB,
			&result.IndexSizeMB,
			&result.TotalSizeMB,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

// GetMaintenanceRecommendations genera recomendaciones de mantenimiento
func (r *MaintenanceRepository) GetMaintenanceRecommendations() ([]domain.MaintenanceRecommendation, error) {
	recommendations := []domain.MaintenanceRecommendation{}

	// 1. Verificar si hay particiones faltantes
	missingPartitions, err := r.CheckFuturePartitions(3)
	if err == nil {
		hasMissing := false
		for _, p := range missingPartitions {
			if !p.PartitionsExist {
				hasMissing = true
				break
			}
		}

		if hasMissing {
			recommendations = append(recommendations, domain.MaintenanceRecommendation{
				Operation: domain.MaintenanceCheckPartitions,
				Priority:  "critical",
				Reason:    "Hay particiones faltantes para los próximos 3 meses. Las inserciones futuras fallarán.",
				ShouldRun: true,
			})
		}
	}

	// 2. Verificar última vez que se refrescaron las vistas
	stats, err := r.GetRefreshStatistics(1)
	if err == nil {
		now := time.Now()
		for _, stat := range stats {
			if stat.LastRefresh == nil {
				recommendations = append(recommendations, domain.MaintenanceRecommendation{
					Operation: domain.MaintenanceRefreshViews,
					Priority:  "warning",
					Reason:    "La vista " + stat.ViewName + " nunca se ha refrescado",
					ShouldRun: true,
				})
			} else {
				hoursSinceRefresh := now.Sub(*stat.LastRefresh).Hours()

				// Vistas SEO: recomendar si no se han refrescado en 6+ horas
				if (stat.ViewName == "mv_top_tags_seo" || stat.ViewName == "mv_top_countries_seo") && hoursSinceRefresh > 6 {
					recommendations = append(recommendations, domain.MaintenanceRecommendation{
						Operation: domain.MaintenanceRefreshViews,
						Priority:  "info",
						Reason:    "La vista " + stat.ViewName + " no se ha refrescado en más de 6 horas",
						ShouldRun: true,
						LastRunAt: stat.LastRefresh,
					})
				}

				// Vistas Analytics: recomendar si no se han refrescado en 1+ hora
				if stat.ViewName == "mv_station_stats_7d" && hoursSinceRefresh > 1 {
					recommendations = append(recommendations, domain.MaintenanceRecommendation{
						Operation: domain.MaintenanceRefreshViews,
						Priority:  "info",
						Reason:    "La vista " + stat.ViewName + " no se ha refrescado en más de 1 hora",
						ShouldRun: true,
						LastRunAt: stat.LastRefresh,
					})
				}
			}
		}
	}

	// 3. Verificar si hay particiones antiguas para limpiar
	partitions, err := r.GetPartitionStatus()
	if err == nil {
		now := time.Now()
		oldPartitionsCount := 0
		for _, p := range partitions {
			monthsOld := int(now.Sub(p.PartitionDate).Hours() / 24 / 30)
			if monthsOld > 12 {
				oldPartitionsCount++
			}
		}

		if oldPartitionsCount > 0 {
			recommendations = append(recommendations, domain.MaintenanceRecommendation{
				Operation: domain.MaintenanceCleanupPartitions,
				Priority:  "info",
				Reason:    "Hay " + string(rune(oldPartitionsCount)) + " particiones con más de 12 meses que pueden ser eliminadas para liberar espacio",
				ShouldRun: true,
			})
		}
	}

	// Si no hay recomendaciones críticas, todo está OK
	if len(recommendations) == 0 {
		recommendations = append(recommendations, domain.MaintenanceRecommendation{
			Operation: domain.MaintenancePartitionStatus,
			Priority:  "info",
			Reason:    "Todo está funcionando correctamente. No se requiere mantenimiento inmediato.",
			ShouldRun: false,
		})
	}

	return recommendations, nil
}
