package services

import (
	"fmt"

	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"
	"radio-backend/internal/infrastructure/logger"
)

// TranslationService maneja la lógica de negocio para traducciones
type TranslationService struct {
	translationRepo domain.TranslationRepository
	stationRepo     domain.StationRepository
}

// NewTranslationService crea una nueva instancia del servicio de traducciones
func NewTranslationService(
	translationRepo domain.TranslationRepository,
	stationRepo domain.StationRepository,
) *TranslationService {
	return &TranslationService{
		translationRepo: translationRepo,
		stationRepo:     stationRepo,
	}
}

// CreateTranslation crea una nueva traducción después de validar
func (s *TranslationService) CreateTranslation(req *domain.CreateTranslationRequest) (*domain.StationTranslation, error) {
	logger.Info("creating translation", "station_id", req.StationID, "language", req.LanguageCode)

	// Validar idioma
	lang := i18n.ParseLanguage(req.LanguageCode)
	if !i18n.IsSupported(lang) {
		return nil, domain.ErrUnsupportedLanguage
	}

	// Verificar que la estación existe
	_, err := s.stationRepo.FindByID(req.StationID)
	if err != nil {
		logger.Warn("station not found for translation", "station_id", req.StationID)
		return nil, domain.ErrStationNotFound
	}

	// Construir entidad
	translation := &domain.StationTranslation{
		StationID:    req.StationID,
		LanguageCode: lang,
		Title:        req.Title,
		Description:  req.Description,
		Keywords:     req.Keywords,
	}

	// Validar entidad
	if err := translation.Validate(); err != nil {
		return nil, err
	}

	// Crear en repositorio
	if err := s.translationRepo.Create(translation); err != nil {
		logger.Error("failed to create translation", "error", err)
		return nil, err
	}

	logger.Info("translation created successfully", "station_id", req.StationID, "language", lang)
	return translation, nil
}

// GetTranslation obtiene una traducción específica
func (s *TranslationService) GetTranslation(stationID string, lang i18n.Language) (*domain.StationTranslation, error) {
	return s.translationRepo.Get(stationID, lang)
}

// UpdateTranslation actualiza una traducción existente
func (s *TranslationService) UpdateTranslation(stationID string, lang i18n.Language, req *domain.UpdateTranslationRequest) (*domain.StationTranslation, error) {
	logger.Info("updating translation", "station_id", stationID, "language", lang)

	// Verificar que existe
	existing, err := s.translationRepo.Get(stationID, lang)
	if err != nil {
		return nil, err
	}

	// Actualizar campos
	existing.Title = req.Title
	existing.Description = req.Description
	existing.Keywords = req.Keywords

	// Validar
	if err := existing.Validate(); err != nil {
		return nil, err
	}

	// Actualizar en repositorio
	if err := s.translationRepo.Update(existing); err != nil {
		logger.Error("failed to update translation", "error", err)
		return nil, err
	}

	logger.Info("translation updated successfully", "station_id", stationID, "language", lang)
	return existing, nil
}

// DeleteTranslation elimina una traducción
func (s *TranslationService) DeleteTranslation(stationID string, lang i18n.Language) error {
	logger.Info("deleting translation", "station_id", stationID, "language", lang)

	if err := s.translationRepo.Delete(stationID, lang); err != nil {
		logger.Error("failed to delete translation", "error", err)
		return err
	}

	logger.Info("translation deleted successfully", "station_id", stationID, "language", lang)
	return nil
}

// ListTranslationsByStation lista todas las traducciones de una estación
func (s *TranslationService) ListTranslationsByStation(stationID string) ([]*domain.StationTranslation, error) {
	// Verificar que la estación existe
	_, err := s.stationRepo.FindByID(stationID)
	if err != nil {
		logger.Warn("station not found for listing translations", "station_id", stationID)
		return nil, domain.ErrStationNotFound
	}

	return s.translationRepo.ListByStation(stationID)
}

// GetAvailableLanguages obtiene los idiomas disponibles para una estación
func (s *TranslationService) GetAvailableLanguages(stationID string) ([]i18n.Language, error) {
	return s.translationRepo.GetAvailableLanguages(stationID)
}

// GetOrGenerateTranslation obtiene traducción de BD o genera una por defecto
// Esta es la función principal que usa SEOService
func (s *TranslationService) GetOrGenerateTranslation(
	station *domain.Station,
	lang i18n.Language,
) *domain.StationTranslation {
	// Intentar obtener de BD
	translation, err := s.translationRepo.Get(station.ID, lang)
	if err == nil {
		return translation
	}

	// Generar traducción por defecto
	return s.generateDefaultTranslation(station, lang)
}

// generateDefaultTranslation genera una traducción por defecto cuando no existe en BD
func (s *TranslationService) generateDefaultTranslation(station *domain.Station, lang i18n.Language) *domain.StationTranslation {
	templates := s.getTranslationTemplates()

	template, ok := templates[lang]
	if !ok {
		template = templates[i18n.DefaultLanguage]
	}

	// Construir título
	title := fmt.Sprintf(template.TitleFormat, station.Name)

	// Construir descripción
	description := fmt.Sprintf(template.DescriptionFormat, station.Name, station.Country)
	if len(station.Tags) > 0 {
		description = fmt.Sprintf(template.DescriptionWithTagsFormat, station.Name, station.Tags[0], station.Country)
	}

	// Keywords básicos
	keywords := []string{station.Name, template.KeywordRadio, template.KeywordLive, template.KeywordOnline}
	if station.Country != "" {
		keywords = append(keywords, station.Country)
	}
	if len(station.Tags) > 0 {
		keywords = append(keywords, station.Tags...)
	}

	return &domain.StationTranslation{
		StationID:    station.ID,
		LanguageCode: lang,
		Title:        title,
		Description:  description,
		Keywords:     keywords,
	}
}

// translationTemplate define el template de traducción para un idioma
type translationTemplate struct {
	TitleFormat               string
	DescriptionFormat         string
	DescriptionWithTagsFormat string
	KeywordRadio              string
	KeywordLive               string
	KeywordOnline             string
}

// getTranslationTemplates retorna los templates de traducción para cada idioma
func (s *TranslationService) getTranslationTemplates() map[i18n.Language]translationTemplate {
	return map[i18n.Language]translationTemplate{
		i18n.LanguageES: {
			TitleFormat:               "Escucha %s en vivo - Radio Online",
			DescriptionFormat:         "Transmisión en vivo de %s desde %s. Escucha tu radio favorita online gratis.",
			DescriptionWithTagsFormat: "Transmisión en vivo de %s: %s desde %s. Escucha tu radio favorita online gratis.",
			KeywordRadio:              "radio",
			KeywordLive:               "en vivo",
			KeywordOnline:             "online",
		},
		i18n.LanguageEN: {
			TitleFormat:               "Listen to %s Live - Online Radio",
			DescriptionFormat:         "Live stream of %s from %s. Listen to your favorite radio online for free.",
			DescriptionWithTagsFormat: "Live stream of %s: %s from %s. Listen to your favorite radio online for free.",
			KeywordRadio:              "radio",
			KeywordLive:               "live",
			KeywordOnline:             "online",
		},
		i18n.LanguageFR: {
			TitleFormat:               "Écoutez %s en direct - Radio en ligne",
			DescriptionFormat:         "Diffusion en direct de %s depuis %s. Écoutez votre radio préférée en ligne gratuitement.",
			DescriptionWithTagsFormat: "Diffusion en direct de %s: %s depuis %s. Écoutez votre radio préférée en ligne gratuitement.",
			KeywordRadio:              "radio",
			KeywordLive:               "en direct",
			KeywordOnline:             "en ligne",
		},
		i18n.LanguageDE: {
			TitleFormat:               "Hören Sie %s live - Online-Radio",
			DescriptionFormat:         "Live-Übertragung von %s aus %s. Hören Sie Ihr Lieblingsradio kostenlos online.",
			DescriptionWithTagsFormat: "Live-Übertragung von %s: %s aus %s. Hören Sie Ihr Lieblingsradio kostenlos online.",
			KeywordRadio:              "Radio",
			KeywordLive:               "live",
			KeywordOnline:             "online",
		},
	}
}

// BulkCreateTranslations crea múltiples traducciones en una transacción
func (s *TranslationService) BulkCreateTranslations(translations []*domain.StationTranslation) error {
	logger.Info("bulk creating translations", "count", len(translations))

	// Validar todas las traducciones
	for _, translation := range translations {
		if err := translation.Validate(); err != nil {
			return fmt.Errorf("invalid translation for station %s, language %s: %w",
				translation.StationID, translation.LanguageCode, err)
		}
	}

	// Crear en bulk
	if err := s.translationRepo.BulkCreate(translations); err != nil {
		logger.Error("failed to bulk create translations", "error", err)
		return err
	}

	logger.Info("bulk translations created successfully", "count", len(translations))
	return nil
}
