package helpers

import (
	"fmt"
	"os"
	"testing"

	"radio-backend/internal/infrastructure/database"
)

// SetupTestDB configura una base de datos de prueba
func SetupTestDB(t *testing.T) *database.Connection {
	t.Helper()

	// Usar base de datos de test desde variables de entorno
	// o fallback a localhost
	dbURL := GetEnv("TEST_DATABASE_URL", "postgres://test:test@localhost:5432/radio_test?sslmode=disable")

	db, err := database.NewConnection(dbURL, 10, 5)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Limpiar la base de datos antes de cada test
	CleanDatabase(t, db)

	return db
}

// CleanDatabase limpia todas las tablas de la base de datos
func CleanDatabase(t *testing.T, db *database.Connection) {
	t.Helper()

	// Orden de limpieza importante debido a foreign keys
	tables := []string{
		"stream_sessions",
		"ad_impressions",
		"ad_clicks",
		"user_ad_profiles",
		"advertisements",
		"ad_campaigns",
		"user_favorites",
		"sessions",
		"users",
		"stations_cache",
		"search_cache",
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := db.DB.Exec(query)
		if err != nil {
			// No fallar el test si la tabla no existe
			t.Logf("Warning: failed to truncate %s: %v", table, err)
		}
	}

	t.Log("Database cleaned successfully")
}

// TeardownTestDB cierra la conexión y limpia la base de datos
func TeardownTestDB(t *testing.T, db *database.Connection) {
	t.Helper()

	CleanDatabase(t, db)
	db.Close()

	t.Log("Test database closed")
}

// GetEnv obtiene una variable de entorno o retorna un valor por defecto
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
