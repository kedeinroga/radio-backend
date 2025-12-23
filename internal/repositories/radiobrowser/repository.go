package radiobrowser

import (
	"fmt"
	"strconv"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/logger"

	"github.com/go-resty/resty/v2"
	"github.com/sony/gobreaker"
)

// Repository implements domain.StationRepository using Radio Browser API
type Repository struct {
	client  *resty.Client
	baseURL string
	cb      *gobreaker.CircuitBreaker
}

// NewRepository creates a new Radio Browser repository
func NewRepository(baseURL string) *Repository {
	client := resty.New()
	client.SetHeader("User-Agent", "RadioBackend/1.0")

	// Configure timeouts - reduced to 5 seconds for faster failure detection
	client.SetTimeout(5 * time.Second)
	client.SetRetryCount(1) // Reduced to 1 retry to fail faster
	client.SetRetryWaitTime(500 * time.Millisecond)
	client.SetRetryMaxWaitTime(1 * time.Second)

	// Add retry condition - only retry on network errors, not 5xx
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		// Only retry on network errors, not server errors
		return err != nil && r == nil
	})

	// Configure Circuit Breaker
	cbSettings := gobreaker.Settings{
		Name:        "RadioBrowserAPI",
		MaxRequests: 3,                // Allow 3 requests in half-open state
		Interval:    30 * time.Second, // Reset failure count after 30s
		Timeout:     60 * time.Second, // Try to recover after 60s
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit after 3 consecutive failures
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("circuit breaker state changed", "name", name, "from", from.String(), "to", to.String())
		},
	}

	return &Repository{
		client:  client,
		baseURL: baseURL,
		cb:      gobreaker.NewCircuitBreaker(cbSettings),
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

// FindByID returns a station by its UUID
func (r *Repository) FindByID(id string) (*domain.Station, error) {
	result, err := r.cb.Execute(func() (interface{}, error) {
		var stations []RadioBrowserStation

		resp, err := r.client.R().
			SetResult(&stations).
			SetQueryParam("uuids", id).
			Get(r.baseURL + "/json/stations/byuuid")

		if err != nil {
			logger.Error("radiobrowser API error on FindByID", "id", id, "error", err)
			return nil, fmt.Errorf("failed to fetch station by ID: %w", err)
		}

		if resp.IsError() {
			logger.Error("radiobrowser API returned error status", "id", id, "status", resp.Status(), "body", resp.String())
			return nil, fmt.Errorf("radio browser API error: %s", resp.Status())
		}

		if len(stations) == 0 {
			return nil, domain.ErrStationNotFound
		}

		domainStations := r.toDomainStations(stations)
		return &domainStations[0], nil
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			logger.Warn("circuit breaker is open, skipping API call", "method", "FindByID", "id", id)
			return nil, fmt.Errorf("external API temporarily unavailable")
		}
		return nil, err
	}

	return result.(*domain.Station), nil
}

// FindPopular returns popular stations
func (r *Repository) FindPopular(limit int, country string) ([]domain.Station, error) {
	result, err := r.cb.Execute(func() (interface{}, error) {
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
			logger.Error("radiobrowser API error on FindPopular", "limit", limit, "country", country, "error", err)
			return nil, fmt.Errorf("failed to fetch popular stations: %w", err)
		}

		if resp.IsError() {
			logger.Error("radiobrowser API returned error status", "limit", limit, "country", country, "status", resp.Status(), "body", resp.String())
			return nil, fmt.Errorf("radio browser API error: %s", resp.Status())
		}

		return r.toDomainStations(stations), nil
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			logger.Warn("circuit breaker is open, skipping API call", "method", "FindPopular", "limit", limit, "country", country)
			return nil, fmt.Errorf("external API temporarily unavailable")
		}
		return nil, err
	}

	return result.([]domain.Station), nil
}

// Search searches for stations
func (r *Repository) Search(query string, limit int) ([]domain.Station, error) {
	logger.Info("calling radiobrowser API", "query", query, "limit", limit, "url", r.baseURL, "circuit_state", r.cb.State().String())
	
	result, err := r.cb.Execute(func() (interface{}, error) {
		var stations []RadioBrowserStation

		resp, err := r.client.R().
			SetResult(&stations).
			SetQueryParam("name", query).
			SetQueryParam("limit", strconv.Itoa(limit)).
			Get(r.baseURL + "/json/stations/search")

		if err != nil {
			logger.Error("radiobrowser API error on Search", "query", query, "limit", limit, "error", err)
			return nil, fmt.Errorf("failed to search stations: %w", err)
		}

		if resp.IsError() {
			logger.Error("radiobrowser API returned error status", "query", query, "limit", limit, "status", resp.Status(), "body", resp.String())
			return nil, fmt.Errorf("radio browser API error: %s", resp.Status())
		}

		logger.Info("radiobrowser API response", "query", query, "stations_found", len(stations), "status", resp.StatusCode())
		
		return r.toDomainStations(stations), nil
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			logger.Warn("circuit breaker is open, API temporarily unavailable", "method", "Search", "query", query, "limit", limit)
			return nil, fmt.Errorf("external API temporarily unavailable, please try again later")
		}
		return nil, err
	}

	return result.([]domain.Station), nil
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
