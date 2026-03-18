package database

import (
	"errors"
	"fmt"
	"log/slog"

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
		msg := fmt.Sprintf("la base de datos está en estado dirty en versión %d — requiere intervención manual", version)
		if cfg.FailOnDirty {
			return errors.New(msg)
		}
		slog.Warn("migraciones", "advertencia", msg)
	}

	slog.Info("migraciones", "version_actual", version, "dirty", dirty)

	// Aplicar migraciones pendientes
	if err = m.Up(); err != nil {
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
