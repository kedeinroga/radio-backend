// cmd/migrate/main.go
// Ejecutar migraciones de base de datos usando golang-migrate.
//
// Uso:
//
//	go run ./cmd/migrate -action up              # aplicar todas las pendientes
//	go run ./cmd/migrate -action down -steps 1  # revertir la última
//	go run ./cmd/migrate -action version         # ver versión actual
//	go run ./cmd/migrate -action force -version 1 # forzar versión (recovery)
//
// Variables de entorno requeridas:
//
//	DATABASE_URL   — postgresql://user:pass@host:5432/dbname?sslmode=require
//	MIGRATIONS_DIR — (opcional) ruta a la carpeta de migrations, default: ./migrations
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	action := flag.String("action", "up", "Acción: up | down | version | force")
	steps := flag.Int("steps", 1, "Número de pasos para down (default: 1)")
	version := flag.Int("version", 0, "Versión para force (solo con -action force)")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL no está definida")
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}

	source := fmt.Sprintf("file://%s", migrationsDir)

	m, err := migrate.New(source, dbURL)
	if err != nil {
		log.Fatalf("Error inicializando migrate: %v", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("Error cerrando source: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("Error cerrando db: %v", dbErr)
		}
	}()

	currentVersion, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		log.Fatalf("Error obteniendo versión actual: %v", err)
	}
	if dirty {
		log.Fatalf("La base de datos está en estado 'dirty' (versión %d). "+
			"Resolver manualmente con: -action force -version %d", currentVersion, currentVersion)
	}

	log.Printf("Versión actual de migraciones: %d", currentVersion)

	switch *action {
	case "up":
		log.Println("Aplicando migraciones pendientes...")
		err = m.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No hay migraciones pendientes.")
			return
		}
		if err != nil {
			log.Fatalf("Error aplicando migraciones: %v", err)
		}
		newVersion, _, _ := m.Version()
		log.Printf("Migraciones aplicadas. Nueva versión: %d", newVersion)

	case "down":
		log.Printf("Revirtiendo %d migración(es)...", *steps)
		err = m.Steps(-(*steps))
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Nada que revertir.")
			return
		}
		if err != nil {
			log.Fatalf("Error revirtiendo migraciones: %v", err)
		}
		newVersion, _, _ := m.Version()
		log.Printf("Revertido. Versión actual: %d", newVersion)

	case "version":
		v, dirty, err := m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			log.Println("Sin migraciones aplicadas (versión 0)")
			return
		}
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		log.Printf("Versión: %d | Dirty: %v", v, dirty)

	case "force":
		if *version == 0 {
			log.Fatal("-version es requerido con -action force")
		}
		log.Printf("Forzando versión a %d (uso de emergencia)...", *version)
		if err = m.Force(*version); err != nil {
			log.Fatalf("Error forzando versión: %v", err)
		}
		log.Printf("Versión forzada a %d.", *version)

	default:
		log.Fatalf("Acción desconocida: %s. Usar: up | down | version | force", *action)
	}
}
