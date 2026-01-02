package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/logger"
)

// SEORepository implementa domain.SEORepository usando PostgreSQL
type SEORepository struct {
	db *sql.DB
}

// NewSEORepository crea una nueva instancia del repositorio SEO
func NewSEORepository(db *sql.DB) *SEORepository {
	return &SEORepository{db: db}
}

// GetPopularTags obtiene los tags más populares desde la vista materializada
func (r *SEORepository) GetPopularTags(limit int) ([]domain.PopularTag, error) {
	query := `
		SELECT tag, slug, station_count, active_count
		FROM mv_top_tags_seo
		ORDER BY station_count DESC, active_count DESC
		LIMIT $1
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		logger.Error("failed to query popular tags", "error", err)
		return nil, fmt.Errorf("failed to query popular tags: %w", err)
	}
	defer rows.Close()

	var tags []domain.PopularTag
	for rows.Next() {
		var tag domain.PopularTag
		if err := rows.Scan(&tag.Name, &tag.Slug, &tag.StationCount, &tag.ActiveCount); err != nil {
			logger.Error("failed to scan tag row", "error", err)
			continue
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		logger.Error("error iterating tag rows", "error", err)
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}

	logger.Info("retrieved popular tags", "count", len(tags))
	return tags, nil
}

// GetPopularCountries obtiene los países más populares desde la vista materializada
func (r *SEORepository) GetPopularCountries(limit int) ([]domain.PopularCountry, error) {
	query := `
		SELECT country_name, slug, station_count
		FROM mv_top_countries_seo
		ORDER BY station_count DESC
		LIMIT $1
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		logger.Error("failed to query popular countries", "error", err)
		return nil, fmt.Errorf("failed to query popular countries: %w", err)
	}
	defer rows.Close()

	var countries []domain.PopularCountry
	for rows.Next() {
		var country domain.PopularCountry
		// Usamos Name para ambos campos ya que no tenemos código separado
		if err := rows.Scan(&country.Name, &country.Slug, &country.StationCount); err != nil {
			logger.Error("failed to scan country row", "error", err)
			continue
		}
		// Dejar Code vacío o igual a Name
		country.Code = country.Name
		countries = append(countries, country)
	}

	if err := rows.Err(); err != nil {
		logger.Error("error iterating country rows", "error", err)
		return nil, fmt.Errorf("error iterating country rows: %w", err)
	}

	logger.Info("retrieved popular countries", "count", len(countries))
	return countries, nil
}

// GetTotalStations obtiene el total de estaciones activas
func (r *SEORepository) GetTotalStations() (int, error) {
	query := `
		SELECT COUNT(*)
		FROM stations
		WHERE is_active = true
	`

	var total int
	err := r.db.QueryRow(query).Scan(&total)
	if err != nil {
		logger.Error("failed to query total stations", "error", err)
		return 0, fmt.Errorf("failed to query total stations: %w", err)
	}

	logger.Info("retrieved total stations", "total", total)
	return total, nil
}

// UpdateTagStats actualiza las estadísticas de tags desde stations_cache
func (r *SEORepository) UpdateTagStats() error {
	logger.Info("updating tag statistics")

	// Usar una transacción para atomicidad
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Obtener todos los tags únicos y sus conteos
	query := `
		WITH tag_data AS (
			SELECT
				LEFT(LOWER(TRIM(unnest(tags))), 255) as tag,
				COUNT(*) as station_count,
				COUNT(*) FILTER (
					WHERE stream_url IS NOT NULL
					AND stream_url != ''
					AND is_active = true
				) as active_count
			FROM stations
			WHERE array_length(tags, 1) > 0
			GROUP BY LEFT(LOWER(TRIM(unnest(tags))), 255)
		)
		INSERT INTO seo_tag_stats (tag, slug, station_count, active_count, last_updated)
		SELECT
			tag,
			LEFT(LOWER(REGEXP_REPLACE(tag, '[^a-z0-9]+', '-', 'g')), 255) as slug,
			station_count,
			active_count,
			NOW()
		FROM tag_data
		WHERE tag IS NOT NULL
		  AND tag != ''
		  AND LENGTH(tag) <= 255
		  AND LENGTH(tag) > 1  -- Filtrar tags de un solo caracter
		ON CONFLICT (tag)
		DO UPDATE SET
			slug = EXCLUDED.slug,
			station_count = EXCLUDED.station_count,
			active_count = EXCLUDED.active_count,
			last_updated = EXCLUDED.last_updated
	`

	result, err := tx.Exec(query)
	if err != nil {
		logger.Error("failed to update tag stats", "error", err)
		return fmt.Errorf("failed to update tag stats: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	logger.Info("tag stats updated", "rows_affected", rowsAffected)

	// 2. Eliminar tags que ya no tienen estaciones
	cleanupQuery := `
		DELETE FROM seo_tag_stats
		WHERE station_count = 0 OR active_count = 0
	`

	cleanupResult, err := tx.Exec(cleanupQuery)
	if err != nil {
		logger.Error("failed to cleanup tag stats", "error", err)
		return fmt.Errorf("failed to cleanup tag stats: %w", err)
	}

	deletedRows, _ := cleanupResult.RowsAffected()
	logger.Info("cleaned up empty tags", "deleted_rows", deletedRows)

	// 3. Refresh materialized view
	refreshQuery := `REFRESH MATERIALIZED VIEW CONCURRENTLY mv_top_tags_seo`
	if _, err := tx.Exec(refreshQuery); err != nil {
		// Si falla el CONCURRENTLY (primera vez), intentar sin él
		logger.Warn("concurrent refresh failed, trying without CONCURRENTLY", "error", err)
		refreshQuery = `REFRESH MATERIALIZED VIEW mv_top_tags_seo`
		if _, err := tx.Exec(refreshQuery); err != nil {
			logger.Error("failed to refresh materialized view", "error", err)
			return fmt.Errorf("failed to refresh materialized view: %w", err)
		}
	}

	logger.Info("materialized view refreshed")

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("tag statistics updated successfully")
	return nil
}

// UpdateCountryStats actualiza las estadísticas de países desde stations
func (r *SEORepository) UpdateCountryStats() error {
	logger.Info("updating country statistics")

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Actualizar estadísticas de países
	query := `
		WITH country_data AS (
			SELECT
				TRIM(country) as country_name,
				COUNT(*) as station_count
			FROM stations
			WHERE country IS NOT NULL
			AND country != ''
			AND is_active = true
			AND LENGTH(TRIM(country)) > 0
			GROUP BY TRIM(country)
		)
		INSERT INTO seo_country_stats (country_name, slug, station_count, last_updated)
		SELECT
			country_name,
			LEFT(LOWER(REGEXP_REPLACE(country_name, '[^a-z0-9]+', '-', 'g')), 100) as slug,
			station_count,
			NOW()
		FROM country_data
		WHERE LENGTH(country_name) > 0
		ON CONFLICT (country_name)
		DO UPDATE SET
			slug = EXCLUDED.slug,
			station_count = EXCLUDED.station_count,
			last_updated = EXCLUDED.last_updated
	`

	result, err := tx.Exec(query)
	if err != nil {
		logger.Error("failed to update country stats", "error", err)
		return fmt.Errorf("failed to update country stats: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	logger.Info("country stats updated", "rows_affected", rowsAffected)

	// 2. Eliminar países sin estaciones
	cleanupQuery := `
		DELETE FROM seo_country_stats
		WHERE station_count = 0
	`

	cleanupResult, err := tx.Exec(cleanupQuery)
	if err != nil {
		logger.Error("failed to cleanup country stats", "error", err)
		return fmt.Errorf("failed to cleanup country stats: %w", err)
	}

	deletedRows, _ := cleanupResult.RowsAffected()
	logger.Info("cleaned up empty countries", "deleted_rows", deletedRows)

	// 3. Refresh materialized view
	refreshQuery := `REFRESH MATERIALIZED VIEW CONCURRENTLY mv_top_countries_seo`
	if _, err := tx.Exec(refreshQuery); err != nil {
		logger.Warn("concurrent refresh failed, trying without CONCURRENTLY", "error", err)
		refreshQuery = `REFRESH MATERIALIZED VIEW mv_top_countries_seo`
		if _, err := tx.Exec(refreshQuery); err != nil {
			logger.Error("failed to refresh materialized view", "error", err)
			return fmt.Errorf("failed to refresh materialized view: %w", err)
		}
	}

	logger.Info("materialized view refreshed")

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("country statistics updated successfully")
	return nil
}

// slugify es una función helper simple para generar slugs en SQL
// Nota: Para casos complejos, es mejor usar el SlugService de Go
func slugify(text string) string {
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)
	// Esta es una implementación básica, el SlugService es más robusto
	return text
}
