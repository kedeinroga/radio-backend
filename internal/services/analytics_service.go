package services

import (
	"fmt"
	"time"

	"radio-backend/internal/domain"
)

// AnalyticsService handles analytics-related business logic
type AnalyticsService struct {
	analyticsRepo  domain.AnalyticsRepository
	analyticsCache domain.AnalyticsCache
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(
	analyticsRepo domain.AnalyticsRepository,
	analyticsCache domain.AnalyticsCache,
) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo:  analyticsRepo,
		analyticsCache: analyticsCache,
	}
}

// TrackRequest tracks an API request
func (s *AnalyticsService) TrackRequest(log *domain.RequestLog) error {
	// Save to database (async in production)
	if err := s.analyticsRepo.SaveRequestLog(log); err != nil {
		// Log error but don't fail the request
		return fmt.Errorf("failed to save request log: %w", err)
	}

	// Track active user in cache
	if log.UserID != nil {
		_ = s.analyticsCache.AddActiveUser(*log.UserID)
	}

	return nil
}

// TrackStationPlay tracks a station play event
func (s *AnalyticsService) TrackStationPlay(stationID string, userID *string, userType domain.UserType, duration time.Duration) error {
	play := &domain.StationPlay{
		StationID: stationID,
		UserID:    userID,
		UserType:  userType,
		Duration:  duration,
	}

	// Save to database
	if err := s.analyticsRepo.SaveStationPlay(play); err != nil {
		return fmt.Errorf("failed to save station play: %w", err)
	}

	// Update cache
	_ = s.analyticsCache.IncrementStationPlay(stationID)

	return nil
}

// TrackSearch tracks a search query
func (s *AnalyticsService) TrackSearch(query string, resultsCount int, userID *string, userType domain.UserType) error {
	searchQuery := &domain.SearchQuery{
		Query:        query,
		ResultsCount: resultsCount,
		UserID:       userID,
		UserType:     userType,
	}

	// Save to database
	if err := s.analyticsRepo.SaveSearchQuery(searchQuery); err != nil {
		return fmt.Errorf("failed to save search query: %w", err)
	}

	// Update cache
	_ = s.analyticsCache.IncrementSearch(query)

	return nil
}

// GetPopularStations returns the most popular stations
func (s *AnalyticsService) GetPopularStations(timeRange string, limit int) ([]domain.StationStats, error) {
	from, to := s.parseTimeRange(timeRange)

	stats, err := s.analyticsRepo.GetPopularStations(from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular stations: %w", err)
	}

	return stats, nil
}

// GetTrendingSearches returns the most trending searches
func (s *AnalyticsService) GetTrendingSearches(timeRange string, limit int) ([]domain.SearchStats, error) {
	from, to := s.parseTimeRange(timeRange)

	stats, err := s.analyticsRepo.GetTrendingSearches(from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending searches: %w", err)
	}

	return stats, nil
}

// GetActiveUsersCount returns the count of active users
func (s *AnalyticsService) GetActiveUsersCount() (int64, error) {
	count, err := s.analyticsCache.GetActiveUsersCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get active users count: %w", err)
	}

	return count, nil
}

// parseTimeRange parses a time range string and returns from/to times
func (s *AnalyticsService) parseTimeRange(timeRange string) (time.Time, time.Time) {
	now := time.Now()
	var from time.Time

	switch timeRange {
	case "hour":
		from = now.Add(-1 * time.Hour)
	case "day":
		from = now.Add(-24 * time.Hour)
	case "week":
		from = now.Add(-7 * 24 * time.Hour)
	case "month":
		from = now.AddDate(0, -1, 0)
	default:
		from = now.Add(-24 * time.Hour) // Default to last 24 hours
	}

	return from, now
}
