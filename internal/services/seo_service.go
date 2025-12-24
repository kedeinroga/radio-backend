package services

import (
	"fmt"
	"strings"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"
	"radio-backend/internal/infrastructure/logger"
)

// SEOService maneja la lógica de negocio para SEO
type SEOService struct {
	seoRepo            domain.SEORepository
	seoCache           domain.SEOCache
	translationService *TranslationService
	slugService        *SlugService
	baseURL            string
}

// NewSEOService crea una nueva instancia del servicio SEO
func NewSEOService(
	seoRepo domain.SEORepository,
	seoCache domain.SEOCache,
	translationService *TranslationService,
	slugService *SlugService,
	baseURL string,
) *SEOService {
	return &SEOService{
		seoRepo:            seoRepo,
		seoCache:           seoCache,
		translationService: translationService,
		slugService:        slugService,
		baseURL:            baseURL,
	}
}

// GetSitemapData retorna datos optimizados para generar sitemap.xml
func (s *SEOService) GetSitemapData() (*domain.SitemapData, error) {
	logger.Info("fetching sitemap data")

	// 1. Intentar obtener desde cache
	cached, err := s.seoCache.GetSitemapData()
	if err == nil && cached != nil {
		logger.Info("sitemap data retrieved from cache")
		return cached, nil
	}

	logger.Info("sitemap data cache miss, fetching from database")

	// 2. Obtener desde base de datos
	tags, err := s.seoRepo.GetPopularTags(100)
	if err != nil {
		logger.Error("failed to get popular tags", "error", err)
		return nil, fmt.Errorf("failed to get popular tags: %w", err)
	}

	countries, err := s.seoRepo.GetPopularCountries(50)
	if err != nil {
		logger.Error("failed to get popular countries", "error", err)
		return nil, fmt.Errorf("failed to get popular countries: %w", err)
	}

	total, err := s.seoRepo.GetTotalStations()
	if err != nil {
		logger.Warn("failed to get total stations, using 0", "error", err)
		total = 0
	}

	// 3. Construir datos del sitemap
	data := &domain.SitemapData{
		PopularTags:      tags,
		PopularCountries: countries,
		TotalStations:    total,
		LastUpdated:      time.Now().Format(time.RFC3339),
	}

	// 4. Guardar en cache (6 horas de TTL)
	if err := s.seoCache.SetSitemapData(data, 6*time.Hour); err != nil {
		logger.Warn("failed to cache sitemap data", "error", err)
		// No retornamos error, los datos son válidos
	}

	logger.Info("sitemap data fetched successfully",
		"tags", len(tags),
		"countries", len(countries),
		"total_stations", total)

	return data, nil
}

// EnrichStationWithSEO enriquece una estación con metadata SEO
// Ahora acepta un parámetro de idioma para traducciones multiidioma
func (s *SEOService) EnrichStationWithSEO(station *domain.Station, lang i18n.Language) {
	if station == nil {
		return
	}

	logger.Info("enriching station with SEO metadata",
		"station_id", station.ID,
		"station_name", station.Name,
		"language", lang)

	// 1. Generar slug
	station.Slug = s.slugService.Slugify(station.Name)

	// 2. Intentar obtener metadata desde cache (con idioma)
	cached, err := s.seoCache.GetStationSEO(station.ID, string(lang))
	if err == nil && cached != nil {
		logger.Info("station SEO metadata retrieved from cache",
			"station_id", station.ID,
			"language", lang)
		station.SEOMetadata = cached
		return
	}

	// 3. Generar metadata con traducciones
	metadata := s.generateMetadata(station, lang)
	station.SEOMetadata = metadata

	// 4. Guardar en cache (24 horas de TTL)
	if err := s.seoCache.SetStationSEO(station.ID, string(lang), metadata, 24*time.Hour); err != nil {
		logger.Warn("failed to cache station SEO metadata",
			"error", err,
			"station_id", station.ID,
			"language", lang)
	}

	logger.Info("station enriched with SEO metadata",
		"station_id", station.ID,
		"slug", station.Slug,
		"language", lang)
}

// EnrichStationsWithSEO enriquece múltiples estaciones con metadata SEO
func (s *SEOService) EnrichStationsWithSEO(stations []domain.Station, lang i18n.Language) {
	for i := range stations {
		s.EnrichStationWithSEO(&stations[i], lang)
	}
}

// generateMetadata genera metadata SEO para una estación
// Ahora usa TranslationService para obtener traducciones
func (s *SEOService) generateMetadata(station *domain.Station, lang i18n.Language) *domain.SEOMetadata {
	// Obtener traducción (de BD o generada por defecto)
	translation := s.translationService.GetOrGenerateTranslation(station, lang)

	// Validar y proporcionar fallback de imagen
	imageURL := s.validateImageURL(station.ImageURL)

	// Generar canonical URL
	canonicalURL := fmt.Sprintf("%s/radio/%s", s.baseURL, station.Slug)

	// Last modified
	lastModified := station.UpdatedAt.Format(time.RFC3339)

	// Alternate names (opcional)
	alternateNames := s.generateAlternateNames(station)

	// Generar hreflang tags para SEO multiidioma
	hreflangTags := s.generateHreflangTags(station.Slug)

	return &domain.SEOMetadata{
		Title:          translation.Title,
		Description:    translation.Description,
		CanonicalURL:   canonicalURL,
		Keywords:       translation.Keywords,
		ImageURL:       imageURL,
		LastModified:   lastModified,
		AlternateNames: alternateNames,
		Language:       lang.String(),
		HreflangTags:   hreflangTags,
	}
}
// validateImageURL valida y proporciona fallback para la URL de imagen
func (s *SEOService) validateImageURL(imageURL string) string {
	// Si no hay imagen o es inválida, usar fallback
	if imageURL == "" || !s.isValidURL(imageURL) {
		return s.baseURL + "/assets/default-radio-icon.png"
	}

	// Verificar que la URL sea segura (HTTPS preferido)
	if strings.HasPrefix(imageURL, "http://") {
		// Intentar convertir a HTTPS
		httpsURL := strings.Replace(imageURL, "http://", "https://", 1)
		return httpsURL
	}

	return imageURL
}

// isValidURL valida que una URL tenga formato correcto
func (s *SEOService) isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// generateAlternateNames genera nombres alternativos para la estación
func (s *SEOService) generateAlternateNames(station *domain.Station) []string {
	names := []string{}

	// Agregar nombre con tags
	if len(station.Tags) > 0 {
		for _, tag := range station.Tags[:min(2, len(station.Tags))] {
			altName := fmt.Sprintf("%s %s", station.Name, tag)
			names = append(names, altName)
		}
	}

	// Agregar nombre con país
	if station.Country != "" {
		altName := fmt.Sprintf("%s %s", station.Name, station.Country)
		names = append(names, altName)
	}

	return names
}

// generateHreflangTags genera tags hreflang para SEO internacional
func (s *SEOService) generateHreflangTags(slug string) []domain.HreflangTag {
	tags := make([]domain.HreflangTag, 0, len(i18n.SupportedLanguages))

	for _, lang := range i18n.SupportedLanguages {
		tags = append(tags, domain.HreflangTag{
			Language: lang.String(),
			URL:      fmt.Sprintf("%s/radio/%s?lang=%s", s.baseURL, slug, lang),
		})
	}

	return tags
}

// InvalidateSitemapCache invalida el cache de sitemap
func (s *SEOService) InvalidateSitemapCache() error {
	logger.Info("invalidating sitemap cache")
	if err := s.seoCache.InvalidateSitemapData(); err != nil {
		logger.Error("failed to invalidate sitemap cache", "error", err)
		return err
	}
	logger.Info("sitemap cache invalidated successfully")
	return nil
}

// RefreshSEOStats actualiza las estadísticas SEO
func (s *SEOService) RefreshSEOStats() error {
	logger.Info("refreshing SEO statistics")

	// 1. Actualizar stats de tags
	if err := s.seoRepo.UpdateTagStats(); err != nil {
		logger.Error("failed to update tag stats", "error", err)
		return fmt.Errorf("failed to update tag stats: %w", err)
	}

	// 2. Actualizar stats de países
	if err := s.seoRepo.UpdateCountryStats(); err != nil {
		logger.Error("failed to update country stats", "error", err)
		return fmt.Errorf("failed to update country stats: %w", err)
	}

	// 3. Invalidar cache de sitemap para forzar refresh
	if err := s.InvalidateSitemapCache(); err != nil {
		logger.Warn("failed to invalidate cache after stats update", "error", err)
	}

	logger.Info("SEO statistics refreshed successfully")
	return nil
}

// Helper function para min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
