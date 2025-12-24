package domain

import (
	"time"

	"radio-backend/internal/i18n"
)

// StationTranslation representa la traducción de una estación en un idioma específico
type StationTranslation struct {
	StationID    string        `json:"station_id"`
	LanguageCode i18n.Language `json:"language_code"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Keywords     []string      `json:"keywords"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// TranslationRepository define las operaciones para gestionar traducciones
type TranslationRepository interface {
	// Create crea una nueva traducción
	Create(translation *StationTranslation) error

	// Get obtiene una traducción específica por station_id y language_code
	Get(stationID string, languageCode i18n.Language) (*StationTranslation, error)

	// Update actualiza una traducción existente
	Update(translation *StationTranslation) error

	// Delete elimina una traducción específica
	Delete(stationID string, languageCode i18n.Language) error

	// ListByStation lista todas las traducciones de una estación
	ListByStation(stationID string) ([]*StationTranslation, error)

	// ListByLanguage lista todas las traducciones de un idioma (paginado)
	ListByLanguage(languageCode i18n.Language, limit, offset int) ([]*StationTranslation, error)

	// Exists verifica si existe una traducción
	Exists(stationID string, languageCode i18n.Language) (bool, error)

	// BulkCreate crea múltiples traducciones en una transacción
	BulkCreate(translations []*StationTranslation) error

	// GetAvailableLanguages obtiene los idiomas disponibles para una estación
	GetAvailableLanguages(stationID string) ([]i18n.Language, error)
}

// CreateTranslationRequest representa la solicitud para crear una traducción
type CreateTranslationRequest struct {
	StationID    string   `json:"station_id" binding:"required"`
	LanguageCode string   `json:"language_code" binding:"required,len=2"`
	Title        string   `json:"title" binding:"required,max=200"`
	Description  string   `json:"description" binding:"required"`
	Keywords     []string `json:"keywords"`
}

// UpdateTranslationRequest representa la solicitud para actualizar una traducción
type UpdateTranslationRequest struct {
	Title       string   `json:"title" binding:"required,max=200"`
	Description string   `json:"description" binding:"required"`
	Keywords    []string `json:"keywords"`
}

// TranslationResponse representa la respuesta de una traducción
type TranslationResponse struct {
	StationID    string   `json:"station_id"`
	LanguageCode string   `json:"language_code"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Keywords     []string `json:"keywords"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// ToResponse convierte StationTranslation a TranslationResponse
func (t *StationTranslation) ToResponse() *TranslationResponse {
	return &TranslationResponse{
		StationID:    t.StationID,
		LanguageCode: t.LanguageCode.String(),
		Title:        t.Title,
		Description:  t.Description,
		Keywords:     t.Keywords,
		CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    t.UpdatedAt.Format(time.RFC3339),
	}
}

// Validate valida los campos de una traducción
func (t *StationTranslation) Validate() error {
	if t.StationID == "" {
		return ErrInvalidStationID
	}

	if !i18n.IsSupported(t.LanguageCode) {
		return ErrUnsupportedLanguage
	}

	if t.Title == "" || len(t.Title) > 200 {
		return ErrInvalidTitle
	}

	if t.Description == "" {
		return ErrInvalidDescription
	}

	return nil
}
