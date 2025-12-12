package services

import (
	"fmt"
	"strings"

	"radio-backend/internal/domain"
)

// StationService handles station-related business logic
type StationService struct {
	stationRepo domain.StationRepository
	analytics   *AnalyticsService
}

// NewStationService creates a new station service
func NewStationService(stationRepo domain.StationRepository, analytics *AnalyticsService) *StationService {
	return &StationService{
		stationRepo: stationRepo,
		analytics:   analytics,
	}
}

// ListPopular returns popular stations filtered by user type
func (s *StationService) ListPopular(limit int, country string, userType domain.UserType) ([]domain.Station, error) {
	// Validate limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Fetch stations from repository
	stations, err := s.stationRepo.FindPopular(limit, country)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch popular stations: %w", err)
	}

	// Filter by user type (apply business rule)
	filtered := s.filterByUserType(stations, userType)

	return filtered, nil
}

// Search searches for stations by query
func (s *StationService) Search(query string, limit int, userType domain.UserType) ([]domain.Station, error) {
	// Validate query
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, domain.ErrInvalidQuery
	}

	if len(query) < 2 {
		return nil, domain.NewValidationError("query", "search query must be at least 2 characters")
	}

	// Validate limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Search stations
	stations, err := s.stationRepo.Search(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search stations: %w", err)
	}

	// Filter by user type
	filtered := s.filterByUserType(stations, userType)

	return filtered, nil
}

// filterByUserType filters stations based on user type
func (s *StationService) filterByUserType(stations []domain.Station, userType domain.UserType) []domain.Station {
	filtered := make([]domain.Station, 0, len(stations))

	for _, station := range stations {
		if station.IsAccessibleBy(userType) {
			filtered = append(filtered, station)
		}
	}

	return filtered
}
