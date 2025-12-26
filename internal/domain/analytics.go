package domain

import "time"

// RequestLog represents a logged API request
type RequestLog struct {
	ID         string
	RequestID  string
	Method     string
	Path       string
	UserID     *string
	UserType   UserType
	StatusCode int
	Duration   time.Duration
	IPAddress  string
	UserAgent  string
	CreatedAt  time.Time
}

// StationPlay represents a station play event
type StationPlay struct {
	ID        string
	StationID string
	UserID    *string
	UserType  UserType
	Duration  time.Duration
	CreatedAt time.Time
}

// SearchQuery represents a search query event
type SearchQuery struct {
	ID           string
	Query        string
	ResultsCount int
	UserID       *string
	UserType     UserType
	CreatedAt    time.Time
}

// AnalyticsRepository defines the interface for analytics data access
type AnalyticsRepository interface {
	SaveRequestLog(log *RequestLog) error
	SaveStationPlay(play *StationPlay) error
	SaveSearchQuery(query *SearchQuery) error
	GetPopularStations(from, to time.Time, limit int) ([]StationStats, error)
	GetTrendingSearches(from, to time.Time, limit int) ([]SearchStats, error)
	CountActiveUsers(from time.Time) (int64, error)
	CountGuestUsers(from time.Time) (int64, error)
}

// StationStats represents aggregated station statistics
type StationStats struct {
	StationID string
	PlayCount int
	Duration  time.Duration
	// Station details
	Name    string
	Country string
	Favicon string
	URL     string
}

// SearchStats represents aggregated search statistics
type SearchStats struct {
	Query       string
	SearchCount int
	AvgResults  float64
	TotalCount  int // Total de búsquedas en el período para calcular porcentaje
}

// AnalyticsCache defines the interface for real-time analytics caching
type AnalyticsCache interface {
	IncrementStationPlay(stationID string) error
	IncrementSearch(query string) error
	GetPopularStations(limit int) ([]string, error)
	GetTrendingSearches(limit int) ([]string, error)
	AddActiveUser(userID string) error
	GetActiveUsersCount() (int64, error)
}
