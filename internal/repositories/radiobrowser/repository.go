package radiobrowser

import (
	"fmt"
	"strconv"

	"radio-backend/internal/domain"

	"github.com/go-resty/resty/v2"
)

// Repository implements domain.StationRepository using Radio Browser API
type Repository struct {
	client  *resty.Client
	baseURL string
}

// NewRepository creates a new Radio Browser repository
func NewRepository(baseURL string) *Repository {
	client := resty.New()
	client.SetHeader("User-Agent", "RadioBackend/1.0")

	return &Repository{
		client:  client,
		baseURL: baseURL,
	}
}

// RadioBrowserStation represents a station from the Radio Browser API
type RadioBrowserStation struct {
	StationUUID string `json:"stationuuid"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	URLResolved string `json:"url_resolved"`
	Favicon     string `json:"favicon"`
	Tags        string `json:"tags"`
	Country     string `json:"country"`
	Votes       int    `json:"votes"`
}

// FindPopular returns popular stations
func (r *Repository) FindPopular(limit int, country string) ([]domain.Station, error) {
	var stations []RadioBrowserStation

	req := r.client.R().
		SetResult(&stations).
		SetQueryParam("limit", strconv.Itoa(limit)).
		SetQueryParam("order", "votes").
		SetQueryParam("reverse", "true")

	if country != "" {
		req.SetQueryParam("country", country)
	}

	resp, err := req.Get(r.baseURL + "/json/stations/search")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch popular stations: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("radio browser API error: %s", resp.Status())
	}

	return r.toDomainStations(stations), nil
}

// Search searches for stations
func (r *Repository) Search(query string, limit int) ([]domain.Station, error) {
	var stations []RadioBrowserStation

	resp, err := r.client.R().
		SetResult(&stations).
		SetQueryParam("name", query).
		SetQueryParam("limit", strconv.Itoa(limit)).
		Get(r.baseURL + "/json/stations/search")

	if err != nil {
		return nil, fmt.Errorf("failed to search stations: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("radio browser API error: %s", resp.Status())
	}

	return r.toDomainStations(stations), nil
}

// toDomainStations converts Radio Browser stations to domain stations
func (r *Repository) toDomainStations(rbStations []RadioBrowserStation) []domain.Station {
	stations := make([]domain.Station, 0, len(rbStations))

	for _, rbs := range rbStations {
		station := domain.Station{
			ID:            rbs.StationUUID,
			Name:          rbs.Name,
			StreamURL:     r.getStreamURL(rbs),
			ImageURL:      rbs.Favicon,
			Tags:          r.parseTags(rbs.Tags),
			Country:       rbs.Country,
			Votes:         rbs.Votes,
			IsPremiumOnly: false, // Default to false, can be configured later
		}
		stations = append(stations, station)
	}

	return stations
}

// getStreamURL returns the best stream URL
func (r *Repository) getStreamURL(station RadioBrowserStation) string {
	if station.URLResolved != "" {
		return station.URLResolved
	}
	return station.URL
}

// parseTags parses comma-separated tags
func (r *Repository) parseTags(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}

	tags := []string{}
	for _, tag := range splitTags(tagsStr) {
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	return tags
}

// splitTags splits tags by comma
func splitTags(s string) []string {
	var tags []string
	current := ""

	for _, char := range s {
		if char == ',' {
			if current != "" {
				tags = append(tags, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		tags = append(tags, current)
	}

	return tags
}
