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

// GuestDetail represents aggregated request details for a single guest IP
type GuestDetail struct {
	IPAddress       string    `json:"ip_address"`
	TotalRequests   int64     `json:"total_requests"`
	UniqueEndpoints int64     `json:"unique_endpoints"`
	UserAgent       string    `json:"user_agent"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
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
	GetGuestDetails(from time.Time, limit int) ([]GuestDetail, error)
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
