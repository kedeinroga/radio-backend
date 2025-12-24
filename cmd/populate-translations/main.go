package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"radio-backend/internal/config"
	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/infrastructure/logger"
	"radio-backend/internal/repositories/postgres"
	"radio-backend/internal/services"

	_ "github.com/lib/pq"
)

// Script para poblar traducciones iniciales de estaciones existentes
func main() {
	fmt.Println("🌍 Iniciando población de traducciones...")

	// 1. Inicializar logger
	logger.Init("text", "info")

	// 2. Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 3. Conectar a la base de datos
	db, err := database.NewConnection(
		cfg.Database.URL,
		cfg.Database.MaxConnections,
		cfg.Database.MaxIdleConnections,
	)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Conectado a la base de datos")

	// 4. Inicializar repositorios
	stationRepo := postgres.NewStationCacheRepository(db)
	translationRepo := postgres.NewTranslationRepository(db.DB)

	// 5. Inicializar servicio
	translationService := services.NewTranslationService(translationRepo, stationRepo)

	// 6. Obtener todas las estaciones activas
	stations, err := getAllStations(db.DB)
	if err != nil {
		log.Fatalf("Error getting stations: %v", err)
	}

	if len(stations) == 0 {
		fmt.Println("⚠️  No hay estaciones en la base de datos")
		return
	}

	fmt.Printf("📊 Encontradas %d estaciones\n", len(stations))

	// 7. Generar traducciones para cada estación
	languages := []i18n.Language{i18n.LanguageES, i18n.LanguageEN, i18n.LanguageFR, i18n.LanguageDE}
	translations := make([]*domain.StationTranslation, 0)

	for _, station := range stations {
		for _, lang := range languages {
			// Verificar si ya existe la traducción
			existing, err := translationService.GetTranslation(station.ID, lang)
			if err == nil && existing != nil {
				fmt.Printf("⏭️  Traducción ya existe: %s (%s)\n", station.Name, lang)
				continue
			}

			// Generar traducción por defecto
			translation := generateDefaultTranslation(&station, lang)
			translations = append(translations, translation)

			fmt.Printf("➕ Generando traducción: %s (%s)\n", station.Name, lang)
		}
	}

	if len(translations) == 0 {
		fmt.Println("✅ Todas las traducciones ya existen")
		return
	}

	// 8. Insertar traducciones en bulk
	fmt.Printf("\n💾 Guardando %d traducciones...\n", len(translations))

	if err := translationService.BulkCreateTranslations(translations); err != nil {
		log.Fatalf("Error creating translations: %v", err)
	}

	fmt.Println("\n🎉 ¡Traducciones pobladas exitosamente!")
	fmt.Printf("📈 Total de traducciones creadas: %d\n", len(translations))
	fmt.Printf("🌐 Idiomas: %v\n", languages)
}

// getAllStations obtiene todas las estaciones activas de la base de datos
func getAllStations(db *sql.DB) ([]domain.Station, error) {
	query := `
		SELECT id, name, stream_url, stream_url_resolved, image_url, tags, country, 
		       votes, is_premium_only, source, last_synced_at, sync_count, is_active,
		       created_at, updated_at
		FROM stations
		WHERE is_active = true
		ORDER BY votes DESC
		LIMIT 1000
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query stations: %w", err)
	}
	defer rows.Close()

	var stations []domain.Station
	for rows.Next() {
		var station domain.Station
		var tags []string
		var lastSyncedAt sql.NullTime
		var imageURL sql.NullString

		err := rows.Scan(
			&station.ID,
			&station.Name,
			&station.StreamURL,
			&station.StreamURLResolved,
			&imageURL,
			&tags,
			&station.Country,
			&station.Votes,
			&station.IsPremiumOnly,
			&station.Source,
			&lastSyncedAt,
			&station.SyncCount,
			&station.IsActive,
			&station.CreatedAt,
			&station.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan station: %w", err)
		}

		if imageURL.Valid {
			station.ImageURL = imageURL.String
		}
		station.Tags = tags
		if lastSyncedAt.Valid {
			station.LastSyncedAt = &lastSyncedAt.Time
		}

		stations = append(stations, station)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stations: %w", err)
	}

	return stations, nil
}

// generateDefaultTranslation genera una traducción por defecto basada en templates
func generateDefaultTranslation(station *domain.Station, lang i18n.Language) *domain.StationTranslation {
	templates := getTranslationTemplates()
	template := templates[lang]

	// Generar título
	title := fmt.Sprintf("%s - %s", station.Name, template.titleSuffix)

	// Generar descripción
	description := fmt.Sprintf(template.descriptionFormat, station.Name, station.Country)

	// Generar keywords
	keywords := generateKeywords(station, lang)

	return &domain.StationTranslation{
		StationID:    station.ID,
		LanguageCode: lang,
		Title:        title,
		Description:  description,
		Keywords:     keywords, // Ya es []string
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// translationTemplate define el template para cada idioma
type translationTemplate struct {
	titleSuffix       string
	descriptionFormat string
	keywordPrefix     string
}

// getTranslationTemplates retorna los templates por idioma
func getTranslationTemplates() map[i18n.Language]translationTemplate {
	return map[i18n.Language]translationTemplate{
		i18n.LanguageES: {
			titleSuffix:       "Radio Online Gratis",
			descriptionFormat: "Escucha %s en vivo. Radio online desde %s con transmisión en directo y alta calidad de audio.",
			keywordPrefix:     "radio",
		},
		i18n.LanguageEN: {
			titleSuffix:       "Free Online Radio",
			descriptionFormat: "Listen to %s live. Online radio from %s with live streaming and high audio quality.",
			keywordPrefix:     "radio",
		},
		i18n.LanguageFR: {
			titleSuffix:       "Radio en Ligne Gratuite",
			descriptionFormat: "Écoutez %s en direct. Radio en ligne depuis %s avec streaming en direct et haute qualité audio.",
			keywordPrefix:     "radio",
		},
		i18n.LanguageDE: {
			titleSuffix:       "Kostenloses Online-Radio",
			descriptionFormat: "Hören Sie %s live. Online-Radio aus %s mit Live-Streaming und hoher Audioqualität.",
			keywordPrefix:     "radio",
		},
	}
}

// generateKeywords genera keywords basadas en el idioma
func generateKeywords(station *domain.Station, lang i18n.Language) []string {
	templates := getTranslationTemplates()
	template := templates[lang]

	keywords := []string{
		template.keywordPrefix,
		station.Name,
		station.Country,
	}

	// Agregar tags de la estación
	if len(station.Tags) > 0 {
		// Limitar a las primeras 3 tags
		maxTags := 3
		if len(station.Tags) < maxTags {
			maxTags = len(station.Tags)
		}
		keywords = append(keywords, station.Tags[:maxTags]...)
	}

	// Agregar palabras clave específicas por idioma
	switch lang {
	case i18n.LanguageES:
		keywords = append(keywords, "online", "gratis", "en vivo", "transmisión")
	case i18n.LanguageEN:
		keywords = append(keywords, "online", "free", "live", "streaming")
	case i18n.LanguageFR:
		keywords = append(keywords, "en ligne", "gratuit", "direct", "streaming")
	case i18n.LanguageDE:
		keywords = append(keywords, "online", "kostenlos", "live", "streaming")
	}

	return keywords
}
