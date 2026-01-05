package jobs

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"radio-backend/internal/infrastructure/database"
)

// HealthJobs maneja los trabajos de monitoreo y health checks
type HealthJobs struct {
	db     *database.Connection
	logger *slog.Logger
}

// NewHealthJobs crea una nueva instancia de HealthJobs
func NewHealthJobs(
	db *database.Connection,
	logger *slog.Logger,
) *HealthJobs {
	return &HealthJobs{
		db:     db,
		logger: logger,
	}
}

// CheckDatabaseHealth verifica la salud de la base de datos
// Ejecuta cada 1 minuto
func (j *HealthJobs) CheckDatabaseHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ping a la base de datos
	start := time.Now()
	if err := j.db.DB.PingContext(ctx); err != nil {
		j.logger.Error("database health check failed",
			"error", err,
		)
		return
	}
	pingDuration := time.Since(start)

	// Obtener estadísticas de conexiones
	stats := j.db.DB.Stats()

	// Verificar si hay demasiadas conexiones abiertas
	if stats.OpenConnections > 90 {
		j.logger.Warn("high database connection count",
			"open_connections", stats.OpenConnections,
			"max_open", stats.MaxOpenConnections,
		)
	}

	// Log health check exitoso (solo en debug para no saturar logs)
	j.logger.Debug("database health check passed",
		"ping_duration", pingDuration,
		"open_connections", stats.OpenConnections,
		"in_use", stats.InUse,
		"idle", stats.Idle,
	)

	// Si el ping tarda mucho, registrar warning
	if pingDuration > 100*time.Millisecond {
		j.logger.Warn("slow database ping",
			"duration", pingDuration,
		)
	}
}

// CheckRedisHealth verifica la salud de Redis
// Ejecuta cada 1 minuto
// TODO: Implementar cuando tengamos Redis configurado
func (j *HealthJobs) CheckRedisHealth() {
	j.logger.Debug("redis health check skipped - not implemented yet")
}

// CollectSystemMetrics recolecta métricas del sistema
// Ejecuta cada 5 minutos
func (j *HealthJobs) CollectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convertir bytes a MB para mejor legibilidad
	allocMB := float64(m.Alloc) / 1024 / 1024
	totalAllocMB := float64(m.TotalAlloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	j.logger.Info("system metrics",
		"goroutines", runtime.NumGoroutine(),
		"alloc_mb", allocMB,
		"total_alloc_mb", totalAllocMB,
		"sys_mb", sysMB,
		"num_gc", m.NumGC,
	)

	// Warning si hay demasiadas goroutines
	if numGoroutines := runtime.NumGoroutine(); numGoroutines > 10000 {
		j.logger.Warn("high goroutine count",
			"count", numGoroutines,
		)
	}

	// Warning si el heap está muy grande
	if allocMB > 1000 { // > 1GB
		j.logger.Warn("high memory allocation",
			"alloc_mb", allocMB,
		)
	}
}

// CheckDiskSpace verifica el espacio en disco
// Ejecuta cada 10 minutos
// TODO: Implementar verificación de espacio en disco
func (j *HealthJobs) CheckDiskSpace() {
	j.logger.Debug("disk space check skipped - not implemented yet")
}

// CheckAPIHealth verifica la salud de APIs externas
// Ejecuta cada 5 minutos
func (j *HealthJobs) CheckAPIHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Verificar que podemos hacer queries a la base de datos
	query := "SELECT 1"
	var result int
	err := j.db.DB.QueryRowContext(ctx, query).Scan(&result)
	if err != nil {
		j.logger.Error("api health check failed - database query error",
			"error", err,
		)
		return
	}

	j.logger.Debug("api health check passed")
}

// MonitorStreams monitorea el estado de los streams activos
// Ejecuta cada 2 minutos
func (j *HealthJobs) MonitorStreams() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Contar sesiones activas
	query := `
		SELECT
			COUNT(*) as active_count,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT station_id) as unique_stations
		FROM stream_sessions
		WHERE status = 'active'
		AND last_heartbeat > NOW() - INTERVAL '2 minutes'
	`

	var (
		activeCount    int64
		uniqueUsers    int64
		uniqueStations int64
	)

	err := j.db.DB.QueryRowContext(ctx, query).Scan(&activeCount, &uniqueUsers, &uniqueStations)
	if err != nil {
		j.logger.Error("failed to monitor active streams",
			"error", err,
		)
		return
	}

	if activeCount > 0 {
		j.logger.Info("active streams",
			"count", activeCount,
			"unique_users", uniqueUsers,
			"unique_stations", uniqueStations,
		)
	}

	// Warning si hay demasiados streams activos
	if activeCount > 10000 {
		j.logger.Warn("very high active stream count",
			"count", activeCount,
		)
	}
}
