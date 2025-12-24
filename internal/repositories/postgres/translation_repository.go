package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"

	"github.com/lib/pq"
)

type translationRepository struct {
	db *sql.DB
}

// NewTranslationRepository crea una nueva instancia del repositorio de traducciones
func NewTranslationRepository(db *sql.DB) domain.TranslationRepository {
	return &translationRepository{db: db}
}

// Create crea una nueva traducción
func (r *translationRepository) Create(translation *domain.StationTranslation) error {
	query := `
		INSERT INTO station_translations (station_id, language_code, title, description, keywords, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	translation.CreatedAt = now
	translation.UpdatedAt = now

	_, err := r.db.Exec(
		query,
		translation.StationID,
		translation.LanguageCode.String(),
		translation.Title,
		translation.Description,
		pq.Array(translation.Keywords),
		translation.CreatedAt,
		translation.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			// 23505 es el código de error de PostgreSQL para violación de clave única
			if pqErr.Code == "23505" {
				return domain.ErrTranslationExists
			}
			// 23503 es el código de error de PostgreSQL para violación de clave foránea
			if pqErr.Code == "23503" {
				return domain.ErrStationNotFound
			}
		}
		return fmt.Errorf("failed to create translation: %w", err)
	}

	return nil
}

// Get obtiene una traducción específica
func (r *translationRepository) Get(stationID string, languageCode i18n.Language) (*domain.StationTranslation, error) {
	query := `
		SELECT station_id, language_code, title, description, keywords, created_at, updated_at
		FROM station_translations
		WHERE station_id = $1 AND language_code = $2
	`

	translation := &domain.StationTranslation{}
	var langCode string
	var keywords pq.StringArray

	err := r.db.QueryRow(query, stationID, languageCode.String()).Scan(
		&translation.StationID,
		&langCode,
		&translation.Title,
		&translation.Description,
		&keywords,
		&translation.CreatedAt,
		&translation.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrTranslationNotFound
		}
		return nil, fmt.Errorf("failed to get translation: %w", err)
	}

	translation.LanguageCode = i18n.ParseLanguage(langCode)
	translation.Keywords = []string(keywords)

	return translation, nil
}

// Update actualiza una traducción existente
func (r *translationRepository) Update(translation *domain.StationTranslation) error {
	query := `
		UPDATE station_translations
		SET title = $1, description = $2, keywords = $3, updated_at = $4
		WHERE station_id = $5 AND language_code = $6
	`

	translation.UpdatedAt = time.Now()

	result, err := r.db.Exec(
		query,
		translation.Title,
		translation.Description,
		pq.Array(translation.Keywords),
		translation.UpdatedAt,
		translation.StationID,
		translation.LanguageCode.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to update translation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrTranslationNotFound
	}

	return nil
}

// Delete elimina una traducción
func (r *translationRepository) Delete(stationID string, languageCode i18n.Language) error {
	query := `DELETE FROM station_translations WHERE station_id = $1 AND language_code = $2`

	result, err := r.db.Exec(query, stationID, languageCode.String())
	if err != nil {
		return fmt.Errorf("failed to delete translation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrTranslationNotFound
	}

	return nil
}

// ListByStation lista todas las traducciones de una estación
func (r *translationRepository) ListByStation(stationID string) ([]*domain.StationTranslation, error) {
	query := `
		SELECT station_id, language_code, title, description, keywords, created_at, updated_at
		FROM station_translations
		WHERE station_id = $1
		ORDER BY language_code
	`

	rows, err := r.db.Query(query, stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list translations: %w", err)
	}
	defer rows.Close()

	translations := make([]*domain.StationTranslation, 0)

	for rows.Next() {
		translation := &domain.StationTranslation{}
		var langCode string
		var keywords pq.StringArray

		err := rows.Scan(
			&translation.StationID,
			&langCode,
			&translation.Title,
			&translation.Description,
			&keywords,
			&translation.CreatedAt,
			&translation.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan translation: %w", err)
		}

		translation.LanguageCode = i18n.ParseLanguage(langCode)
		translation.Keywords = []string(keywords)
		translations = append(translations, translation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating translations: %w", err)
	}

	return translations, nil
}

// ListByLanguage lista todas las traducciones de un idioma (paginado)
func (r *translationRepository) ListByLanguage(languageCode i18n.Language, limit, offset int) ([]*domain.StationTranslation, error) {
	query := `
		SELECT station_id, language_code, title, description, keywords, created_at, updated_at
		FROM station_translations
		WHERE language_code = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, languageCode.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list translations by language: %w", err)
	}
	defer rows.Close()

	translations := make([]*domain.StationTranslation, 0)

	for rows.Next() {
		translation := &domain.StationTranslation{}
		var langCode string
		var keywords pq.StringArray

		err := rows.Scan(
			&translation.StationID,
			&langCode,
			&translation.Title,
			&translation.Description,
			&keywords,
			&translation.CreatedAt,
			&translation.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan translation: %w", err)
		}

		translation.LanguageCode = i18n.ParseLanguage(langCode)
		translation.Keywords = []string(keywords)
		translations = append(translations, translation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating translations: %w", err)
	}

	return translations, nil
}

// Exists verifica si existe una traducción
func (r *translationRepository) Exists(stationID string, languageCode i18n.Language) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM station_translations WHERE station_id = $1 AND language_code = $2)`

	var exists bool
	err := r.db.QueryRow(query, stationID, languageCode.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check translation existence: %w", err)
	}

	return exists, nil
}

// BulkCreate crea múltiples traducciones en una transacción
func (r *translationRepository) BulkCreate(translations []*domain.StationTranslation) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO station_translations (station_id, language_code, title, description, keywords, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()

	for _, translation := range translations {
		translation.CreatedAt = now
		translation.UpdatedAt = now

		_, err := stmt.Exec(
			translation.StationID,
			translation.LanguageCode.String(),
			translation.Title,
			translation.Description,
			pq.Array(translation.Keywords),
			translation.CreatedAt,
			translation.UpdatedAt,
		)

		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok {
				if pqErr.Code == "23505" {
					return domain.ErrTranslationExists
				}
				if pqErr.Code == "23503" {
					return domain.ErrStationNotFound
				}
			}
			return fmt.Errorf("failed to insert translation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetAvailableLanguages obtiene los idiomas disponibles para una estación
func (r *translationRepository) GetAvailableLanguages(stationID string) ([]i18n.Language, error) {
	query := `
		SELECT DISTINCT language_code
		FROM station_translations
		WHERE station_id = $1
		ORDER BY language_code
	`

	rows, err := r.db.Query(query, stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available languages: %w", err)
	}
	defer rows.Close()

	languages := make([]i18n.Language, 0)

	for rows.Next() {
		var langCode string
		if err := rows.Scan(&langCode); err != nil {
			return nil, fmt.Errorf("failed to scan language: %w", err)
		}
		languages = append(languages, i18n.ParseLanguage(langCode))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating languages: %w", err)
	}

	return languages, nil
}
