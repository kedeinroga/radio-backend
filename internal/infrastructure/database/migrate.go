package database

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateConfig configura el comportamiento de las migraciones al inicio.
type MigrateConfig struct {
	// MigrationsPath es la ruta a los archivos .sql (default: "migrations")
	MigrationsPath string
	// FailOnDirty detiene el servidor si la DB está en estado dirty (recomendado en prod)
	FailOnDirty bool
}

// DefaultMigrateConfig devuelve configuración segura para producción.
func DefaultMigrateConfig() MigrateConfig {
	return MigrateConfig{
		MigrationsPath: "migrations",
		FailOnDirty:    true,
	}
}

// RunMigrations aplica todas las migraciones pendientes.
// Usa su propia conexión (vía databaseURL) para no cerrar el pool principal.
// Si no hay cambios, retorna nil silenciosamente.
// Si la DB está en estado dirty, retorna error (según config).
func RunMigrations(databaseURL string, cfg MigrateConfig) error {
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "migrations"
	}

	source := fmt.Sprintf("file://%s", cfg.MigrationsPath)
	m, err := migrate.New(source, databaseURL)
	if err != nil {
		return fmt.Errorf("cargando migraciones desde %s: %w", source, err)
	}
	defer m.Close()

	// Verificar estado previo
	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("verificando versión de migraciones: %w", err)
	}

	if dirty {
		// La migración anterior falló en una transacción PostgreSQL — los cambios DDL
		// fueron revertidos. Forzar a la versión anterior limpia para poder re-aplicar.
		prevVersion := int(version) - 1
		slog.Warn("migraciones", "advertencia", fmt.Sprintf(
			"dirty state en versión %d — forzando a versión %d y reintentando", version, prevVersion,
		))
		if err := retryOnLock(func() error { return m.Force(prevVersion) }); err != nil {
			return fmt.Errorf("forzando versión %d para salir de dirty state: %w", prevVersion, err)
		}
		version, dirty, _ = m.Version()
		slog.Info("migraciones", "version_tras_force", version, "dirty", dirty)
	}

	slog.Info("migraciones", "version_actual", version, "dirty", dirty)

	// Aplicar migraciones pendientes (con retry si otra instancia tiene el lock)
	if err = retryOnLock(func() error { return m.Up() }); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("migraciones", "estado", "sin cambios pendientes")
			return nil
		}
		return fmt.Errorf("aplicando migraciones: %w", err)
	}

	newVersion, _, _ := m.Version()
	slog.Info("migraciones", "estado", "aplicadas correctamente", "nueva_version", newVersion)

	return nil
}

// retryOnLock reintenta fn hasta 6 veces (espera exponencial de 5–30s) cuando
// golang-migrate no puede adquirir el advisory lock porque otra instancia está
// migrando. Esto evita que Cloud Run falle el deploy por race entre instancias.
func retryOnLock(fn func() error) error {
	const maxRetries = 6
	delay := 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !isLockError(err) {
			return err
		}
		slog.Warn("migraciones", "advertencia", fmt.Sprintf(
			"lock en uso (intento %d/%d), reintentando en %s…", attempt, maxRetries, delay,
		))
		time.Sleep(delay)
		if delay < 30*time.Second {
			delay += 5 * time.Second
		}
	}
	return fmt.Errorf("no se pudo adquirir el lock de migraciones tras %d intentos", maxRetries)
}

func isLockError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, migrate.ErrLocked) ||
		strings.Contains(err.Error(), "try lock failed") ||
		strings.Contains(err.Error(), "lock")
}

// MigrationVersion retorna la versión actual sin ejecutar nada.
// Útil para health checks y endpoints de diagnóstico.
func MigrationVersion(databaseURL, migrationsPath string) (version uint, dirty bool, err error) {
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}

	source := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(source, databaseURL)
	if err != nil {
		return 0, false, fmt.Errorf("cargando migraciones: %w", err)
	}
	defer m.Close()

	v, d, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, d, err
}
