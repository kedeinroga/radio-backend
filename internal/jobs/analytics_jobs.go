package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"radio-backend/internal/infrastructure/database"
)

// AnalyticsJobs maneja los trabajos de agregación de analytics
type AnalyticsJobs struct {
	db     *database.Connection
	logger *slog.Logger
}

// NewAnalyticsJobs crea una nueva instancia de AnalyticsJobs
func NewAnalyticsJobs(
	db *database.Connection,
	logger *slog.Logger,
) *AnalyticsJobs {
	return &AnalyticsJobs{
		db:     db,
		logger: logger,
	}
}

// RefreshMaterializedViews actualiza las vistas materializadas de analytics
// Ejecuta cada 15 minutos
func (j *AnalyticsJobs) RefreshMaterializedViews() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	views := []string{
		"stream_analytics",
		"ad_performance_summary",
	}

	for _, view := range views {
		start := time.Now()

		// REFRESH MATERIALIZED VIEW CONCURRENTLY permite lecturas durante el refresh
		query := fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s", view)
		_, err := j.db.DB.ExecContext(ctx, query)
		if err != nil {
			j.logger.Error("failed to refresh materialized view",
				"view", view,
				"error", err,
			)
			continue
		}

		duration := time.Since(start)
		j.logger.Info("materialized view refreshed",
			"view", view,
			"duration", duration,
		)
	}
}

// GenerateDailySummary genera un resumen diario de analytics
// Ejecuta cada día a las 1:00 AM
func (j *AnalyticsJobs) GenerateDailySummary() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	yesterday := time.Now().AddDate(0, 0, -1)
	dateStr := yesterday.Format("2006-01-02")

	j.logger.Info("generating daily summary",
		"date", dateStr,
	)

	// Resumen de sesiones de streaming
	sessionQuery := `
		SELECT
			COUNT(*) as total_sessions,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT station_id) as unique_stations,
			SUM(listening_duration) as total_duration,
			SUM(bytes_streamed) as total_bytes,
			AVG(listening_duration) as avg_duration
		FROM stream_sessions
		WHERE DATE(created_at) = $1
		AND status = 'completed'
	`

	var (
		totalSessions  int64
		uniqueUsers    int64
		uniqueStations int64
		totalDuration  interface{} // interval
		totalBytes     int64
		avgDuration    interface{} // interval
	)

	err := j.db.DB.QueryRowContext(ctx, sessionQuery, dateStr).Scan(
		&totalSessions,
		&uniqueUsers,
		&uniqueStations,
		&totalDuration,
		&totalBytes,
		&avgDuration,
	)
	if err != nil {
		j.logger.Error("failed to generate daily streaming summary",
			"error", err,
		)
		return
	}

	j.logger.Info("daily streaming summary",
		"date", dateStr,
		"total_sessions", totalSessions,
		"unique_users", uniqueUsers,
		"unique_stations", uniqueStations,
		"total_bytes_gb", float64(totalBytes)/(1024*1024*1024),
	)

	// Resumen de impresiones de anuncios
	adQuery := `
		SELECT
			COUNT(*) as total_impressions,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT advertisement_id) as unique_ads,
			SUM(CASE WHEN clicked THEN 1 ELSE 0 END) as total_clicks,
			AVG(CASE WHEN clicked THEN 1.0 ELSE 0.0 END) as ctr
		FROM ad_impressions
		WHERE DATE(created_at) = $1
	`

	var (
		totalImpressions int64
		uniqueAdUsers    int64
		uniqueAds        int64
		totalClicks      int64
		ctr              float64
	)

	err = j.db.DB.QueryRowContext(ctx, adQuery, dateStr).Scan(
		&totalImpressions,
		&uniqueAdUsers,
		&uniqueAds,
		&totalClicks,
		&ctr,
	)
	if err != nil {
		j.logger.Error("failed to generate daily ad summary",
			"error", err,
		)
		return
	}

	j.logger.Info("daily ad summary",
		"date", dateStr,
		"total_impressions", totalImpressions,
		"unique_users", uniqueAdUsers,
		"unique_ads", uniqueAds,
		"total_clicks", totalClicks,
		"ctr", fmt.Sprintf("%.2f%%", ctr*100),
	)
}

// GenerateWeeklyReport genera un reporte semanal
// Ejecuta cada lunes a las 6:00 AM
func (j *AnalyticsJobs) GenerateWeeklyReport() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Calcular la semana anterior (lunes a domingo)
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday())-6) // Lunes pasado
	weekEnd := weekStart.AddDate(0, 0, 6)                 // Domingo pasado

	j.logger.Info("generating weekly report",
		"week_start", weekStart.Format("2006-01-02"),
		"week_end", weekEnd.Format("2006-01-02"),
	)

	// Top 10 estaciones más escuchadas
	stationsQuery := `
		SELECT
			station_id,
			COUNT(*) as session_count,
			COUNT(DISTINCT user_id) as unique_listeners,
			SUM(listening_duration) as total_duration
		FROM stream_sessions
		WHERE created_at >= $1 AND created_at < $2
		AND status = 'completed'
		GROUP BY station_id
		ORDER BY session_count DESC
		LIMIT 10
	`

	rows, err := j.db.DB.QueryContext(ctx, stationsQuery, weekStart, weekEnd)
	if err != nil {
		j.logger.Error("failed to generate weekly stations report",
			"error", err,
		)
		return
	}
	defer rows.Close()

	j.logger.Info("=== TOP 10 STATIONS (LAST WEEK) ===")
	rank := 1
	for rows.Next() {
		var (
			stationID       string
			sessionCount    int64
			uniqueListeners int64
			totalDuration   interface{}
		)

		if err := rows.Scan(&stationID, &sessionCount, &uniqueListeners, &totalDuration); err != nil {
			j.logger.Error("failed to scan station row", "error", err)
			continue
		}

		j.logger.Info("top station",
			"rank", rank,
			"station_id", stationID,
			"sessions", sessionCount,
			"unique_listeners", uniqueListeners,
		)
		rank++
	}

	// Top anuncios con mejor performance
	adsQuery := `
		SELECT
			advertisement_id,
			COUNT(*) as impression_count,
			SUM(CASE WHEN clicked THEN 1 ELSE 0 END) as click_count,
			AVG(CASE WHEN clicked THEN 1.0 ELSE 0.0 END) as ctr
		FROM ad_impressions
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY advertisement_id
		HAVING COUNT(*) >= 10
		ORDER BY ctr DESC, impression_count DESC
		LIMIT 10
	`

	rows, err = j.db.DB.QueryContext(ctx, adsQuery, weekStart, weekEnd)
	if err != nil {
		j.logger.Error("failed to generate weekly ads report",
			"error", err,
		)
		return
	}
	defer rows.Close()

	j.logger.Info("=== TOP 10 ADS (LAST WEEK) ===")
	rank = 1
	for rows.Next() {
		var (
			adID            string
			impressionCount int64
			clickCount      int64
			ctr             float64
		)

		if err := rows.Scan(&adID, &impressionCount, &clickCount, &ctr); err != nil {
			j.logger.Error("failed to scan ad row", "error", err)
			continue
		}

		j.logger.Info("top ad",
			"rank", rank,
			"ad_id", adID,
			"impressions", impressionCount,
			"clicks", clickCount,
			"ctr", fmt.Sprintf("%.2f%%", ctr*100),
		)
		rank++
	}
}
