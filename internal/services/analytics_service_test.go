package services

import (
	"testing"
	"time"

	"radio-backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockAnalyticsRepository struct {
	mock.Mock
}

func (m *MockAnalyticsRepository) SaveRequestLog(log *domain.RequestLog) error {
	args := m.Called(log)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) SaveStationPlay(play *domain.StationPlay) error {
	args := m.Called(play)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) SaveSearchQuery(query *domain.SearchQuery) error {
	args := m.Called(query)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetPopularStations(from, to time.Time, limit int) ([]domain.StationStats, error) {
	args := m.Called(from, to, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.StationStats), args.Error(1)
}

func (m *MockAnalyticsRepository) GetTrendingSearches(from, to time.Time, limit int) ([]domain.SearchStats, error) {
	args := m.Called(from, to, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.SearchStats), args.Error(1)
}

func (m *MockAnalyticsRepository) CountActiveUsers(from time.Time) (int64, error) {
	args := m.Called(from)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAnalyticsRepository) CountGuestUsers(from time.Time) (int64, error) {
	args := m.Called(from)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAnalyticsRepository) GetGuestDetails(from time.Time, limit int) ([]domain.GuestDetail, error) {
	args := m.Called(from, limit)
	return args.Get(0).([]domain.GuestDetail), args.Error(1)
}

type MockAnalyticsCache struct {
	mock.Mock
}

func (m *MockAnalyticsCache) AddActiveUser(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockAnalyticsCache) GetActiveUsersCount() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAnalyticsCache) IncrementStationPlay(stationID string) error {
	args := m.Called(stationID)
	return args.Error(0)
}

func (m *MockAnalyticsCache) IncrementSearch(query string) error {
	args := m.Called(query)
	return args.Error(0)
}

func (m *MockAnalyticsCache) GetPopularStations(limit int) ([]string, error) {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAnalyticsCache) GetTrendingSearches(limit int) ([]string, error) {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// Tests
func TestAnalyticsService_TrackRequest(t *testing.T) {
	t.Run("successfully tracks request with user ID", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		userID := "user-123"
		log := &domain.RequestLog{
			UserID: &userID,
			Method: "GET",
			Path:   "/api/stations",
		}

		mockRepo.On("SaveRequestLog", log).Return(nil)
		mockCache.On("AddActiveUser", userID).Return(nil)

		err := service.TrackRequest(log)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("tracks request without user ID", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		log := &domain.RequestLog{
			UserID: nil,
			Method: "GET",
			Path:   "/api/stations",
		}

		mockRepo.On("SaveRequestLog", log).Return(nil)

		err := service.TrackRequest(log)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockCache.AssertNotCalled(t, "AddActiveUser")
	})
}

func TestAnalyticsService_TrackStationPlay(t *testing.T) {
	t.Run("successfully tracks station play", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		stationID := "station-1"
		userID := "user-1"
		userType := domain.UserTypeGuest
		duration := 5 * time.Minute

		mockRepo.On("SaveStationPlay", mock.MatchedBy(func(play *domain.StationPlay) bool {
			return play.StationID == stationID &&
				play.UserID != nil && *play.UserID == userID &&
				play.UserType == userType &&
				play.Duration == duration
		})).Return(nil)
		mockCache.On("IncrementStationPlay", stationID).Return(nil)

		err := service.TrackStationPlay(stationID, &userID, userType, duration)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

func TestAnalyticsService_TrackSearch(t *testing.T) {
	t.Run("successfully tracks search query", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		query := "rock music"
		resultsCount := 10
		userID := "user-1"
		userType := domain.UserTypeGuest

		mockRepo.On("SaveSearchQuery", mock.MatchedBy(func(sq *domain.SearchQuery) bool {
			return sq.Query == query &&
				sq.ResultsCount == resultsCount &&
				sq.UserID != nil && *sq.UserID == userID &&
				sq.UserType == userType
		})).Return(nil)
		mockCache.On("IncrementSearch", query).Return(nil)

		err := service.TrackSearch(query, resultsCount, &userID, userType)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

func TestAnalyticsService_GetPopularStations(t *testing.T) {
	t.Run("returns popular stations for time range", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		expectedStats := []domain.StationStats{
			{
				StationID: "station-1",
				PlayCount: 100,
				Name:      "Rock FM",
				Country:   "USA",
				Favicon:   "https://example.com/favicon.png",
				URL:       "https://stream.example.com/rock",
			},
			{
				StationID: "station-2",
				PlayCount: 50,
				Name:      "Pop Radio",
				Country:   "UK",
				Favicon:   "https://example.com/pop.png",
				URL:       "https://stream.example.com/pop",
			},
		}

		mockRepo.On("GetPopularStations", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).
			Return(expectedStats, nil)

		stats, err := service.GetPopularStations("day", 10)

		assert.NoError(t, err)
		assert.Len(t, stats, 2)
		assert.Equal(t, expectedStats, stats)
		mockRepo.AssertExpectations(t)
	})
}

func TestAnalyticsService_GetTrendingSearches(t *testing.T) {
	t.Run("returns trending searches for time range", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		expectedStats := []domain.SearchStats{
			{Query: "rock", SearchCount: 50, TotalCount: 100},
			{Query: "jazz", SearchCount: 30, TotalCount: 100},
		}

		mockRepo.On("GetTrendingSearches", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).
			Return(expectedStats, nil)

		stats, err := service.GetTrendingSearches("week", 10)

		assert.NoError(t, err)
		assert.Len(t, stats, 2)
		assert.Equal(t, expectedStats, stats)
		mockRepo.AssertExpectations(t)
	})
}

func TestAnalyticsService_GetActiveUsersCount(t *testing.T) {
	t.Run("returns active users count from database", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		expectedCount := int64(42)

		// Mock should be called with a time approximately 24 hours ago
		mockRepo.On("CountActiveUsers", mock.MatchedBy(func(from time.Time) bool {
			// Check if the time is approximately 24 hours ago (within 1 second tolerance)
			expectedFrom := time.Now().Add(-24 * time.Hour)
			diff := expectedFrom.Sub(from)
			return diff >= -time.Second && diff <= time.Second
		})).Return(expectedCount, nil)

		count, err := service.GetActiveUsersCount()

		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		mockRepo.AssertExpectations(t)
	})
}

func TestAnalyticsService_GetGuestUsersCount(t *testing.T) {
	t.Run("returns guest users count from database", func(t *testing.T) {
		mockRepo := new(MockAnalyticsRepository)
		mockCache := new(MockAnalyticsCache)

		service := NewAnalyticsService(mockRepo, mockCache)

		expectedCount := int64(128)

		// Mock should be called with a time approximately 24 hours ago
		mockRepo.On("CountGuestUsers", mock.MatchedBy(func(from time.Time) bool {
			// Check if the time is approximately 24 hours ago (within 1 second tolerance)
			expectedFrom := time.Now().Add(-24 * time.Hour)
			diff := expectedFrom.Sub(from)
			return diff >= -time.Second && diff <= time.Second
		})).Return(expectedCount, nil)

		count, err := service.GetGuestUsersCount()

		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		mockRepo.AssertExpectations(t)
	})
}
