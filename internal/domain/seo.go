package domain

import "time"

// SEOMetadata contiene información optimizada para motores de búsqueda
type SEOMetadata struct {
	Title          string         `json:"title" example:"Rock FM 100.1 - Classic Rock Radio"`
	Description    string         `json:"description" example:"Listen to the best classic rock hits from the 80s and 90s on Rock FM 100.1"`
	CanonicalURL   string         `json:"canonical_url" example:"https://radioapp.com/stations/rock-fm-100-1"`
	Keywords       []string       `json:"keywords" example:"rock,classic rock,80s,radio"`
	ImageURL       string         `json:"image_url" example:"https://cdn.rockfm.com/logo.png"`
	LastModified   string         `json:"last_modified" example:"2025-12-23T19:30:00Z"` // ISO 8601 para schema.org
	AlternateNames []string       `json:"alternate_names" example:"Rock FM,Rock Radio 100.1"`
	Language       string         `json:"language" example:"es"`                        // Código de idioma (es, en, fr, de)
	HreflangTags   []HreflangTag  `json:"hreflang_tags,omitempty"`                     // Tags hreflang para SEO multiidioma
}

// HreflangTag representa un tag hreflang para SEO internacional
type HreflangTag struct {
	Language string `json:"lang" example:"es"`
	URL      string `json:"url" example:"https://radioapp.com/stations/rock-fm-100-1?lang=es"`
}

// Slug es un Value Object para URLs amigables
type Slug struct {
	Original  string `json:"original" example:"Rock FM 100.1"`
	Slugified string `json:"slugified" example:"rock-fm-100-1"`
}

// PopularTag representa un tag/género con estadísticas
type PopularTag struct {
	Name         string `json:"name" example:"rock"`
	Slug         string `json:"slug" example:"rock"`
	StationCount int    `json:"station_count" example:"150"`
	ActiveCount  int    `json:"active_count" example:"145"` // Con stream activo
}

// PopularCountry representa un país con estadísticas
type PopularCountry struct {
	Code         string `json:"code" example:"US"` // ISO 3166-1 alpha-2
	Name         string `json:"name" example:"United States"`
	Slug         string `json:"slug" example:"united-states"`
	StationCount int    `json:"station_count" example:"250"`
}

// SitemapData agrupa datos para generar sitemap.xml
type SitemapData struct {
	PopularTags      []PopularTag     `json:"popular_tags"`
	PopularCountries []PopularCountry `json:"popular_countries"`
	TotalStations    int              `json:"total_stations" example:"500"`
	LastUpdated      string           `json:"last_updated" example:"2025-12-23T19:30:00Z"`
}

// SEORepository define operaciones de agregación
type SEORepository interface {
	GetPopularTags(limit int) ([]PopularTag, error)
	GetPopularCountries(limit int) ([]PopularCountry, error)
	GetTotalStations() (int, error)
	UpdateTagStats() error
	UpdateCountryStats() error
}

// SEOCache define operaciones de cache para SEO
type SEOCache interface {
	GetSitemapData() (*SitemapData, error)
	SetSitemapData(data *SitemapData, ttl time.Duration) error
	GetStationSEO(stationID string, language string) (*SEOMetadata, error)
	SetStationSEO(stationID string, language string, metadata *SEOMetadata, ttl time.Duration) error
	InvalidateSitemapData() error
	InvalidateStationSEO(stationID string) error
}
