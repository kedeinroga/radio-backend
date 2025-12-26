package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/google/uuid"
)

// AnalyticsRepository implements domain.AnalyticsRepository using PostgreSQL
type AnalyticsRepository struct {
	db *database.Connection
}

// NewAnalyticsRepository creates a new PostgreSQL analytics repository
func NewAnalyticsRepository(db *database.Connection) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// SaveRequestLog saves a request log
func (r *AnalyticsRepository) SaveRequestLog(log *domain.RequestLog) error {
	log.ID = uuid.New().String()
	log.CreatedAt = time.Now()

	query := `
		INSERT INTO request_logs (
			id, request_id, method, path, user_id, user_type,
			status_code, duration_ms, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.DB.Exec(
		query,
		log.ID,
		log.RequestID,
		log.Method,
		log.Path,
		log.UserID,
		log.UserType,
		log.StatusCode,
		log.Duration.Milliseconds(),
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save request log: %w", err)
	}

	return nil
}

// SaveStationPlay saves a station play event
func (r *AnalyticsRepository) SaveStationPlay(play *domain.StationPlay) error {
	play.ID = uuid.New().String()
	play.CreatedAt = time.Now()

	query := `
		INSERT INTO station_plays (id, station_id, user_id, user_type, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.DB.Exec(
		query,
		play.ID,
		play.StationID,
		play.UserID,
		play.UserType,
		play.Duration.Milliseconds(),
		play.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save station play: %w", err)
	}

	return nil
}

// SaveSearchQuery saves a search query event
func (r *AnalyticsRepository) SaveSearchQuery(query *domain.SearchQuery) error {
	query.ID = uuid.New().String()
	query.CreatedAt = time.Now()

	sqlQuery := `
		INSERT INTO search_queries (id, query, results_count, user_id, user_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.DB.Exec(
		sqlQuery,
		query.ID,
		query.Query,
		query.ResultsCount,
		query.UserID,
		query.UserType,
		query.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save search query: %w", err)
	}

	return nil
}

// GetPopularStations returns the most popular stations in a time range
func (r *AnalyticsRepository) GetPopularStations(from, to time.Time, limit int) ([]domain.StationStats, error) {
	query := `
		SELECT station_id, COUNT(*) as play_count, SUM(duration_ms) as total_duration
		FROM station_plays
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY station_id
		ORDER BY play_count DESC
		LIMIT $3
	`

	rows, err := r.db.DB.Query(query, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular stations: %w", err)
	}
	defer rows.Close()

	var stats []domain.StationStats
	for rows.Next() {
		var s domain.StationStats
		var durationMs int64

		if err := rows.Scan(&s.StationID, &s.PlayCount, &durationMs); err != nil {
			return nil, fmt.Errorf("failed to scan station stats: %w", err)
		}

		s.Duration = time.Duration(durationMs) * time.Millisecond
		stats = append(stats, s)
	}

	return stats, nil
}

// CountActiveUsers returns the count of unique users with activity since the given time
func (r *AnalyticsRepository) CountActiveUsers(from time.Time) (int64, error) {
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM request_logs
		WHERE user_id IS NOT NULL
		AND created_at >= $1
	`

	var count int64
	err := r.db.DB.QueryRow(query, from).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active users: %w", err)
	}

	return count, nil
}

// CountGuestUsers returns the count of unique guest users (by IP) with activity since the given time
func (r *AnalyticsRepository) CountGuestUsers(from time.Time) (int64, error) {
	query := `
		SELECT COUNT(DISTINCT ip_address)
		FROM request_logs
		WHERE user_type = 'guest'
		AND ip_address IS NOT NULL
		AND created_at >= $1
	`

	var count int64
	err := r.db.DB.QueryRow(query, from).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count guest users: %w", err)
	}

	return count, nil
}
// GetTrendingSearches returns the most trending searches in a time range
func (r *AnalyticsRepository) GetTrendingSearches(from, to time.Time, limit int) ([]domain.SearchStats, error) {
	query := `
		SELECT query, COUNT(*) as search_count, AVG(results_count) as avg_results
		FROM search_queries
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY query
		ORDER BY search_count DESC
		LIMIT $3
	`

	rows, err := r.db.DB.Query(query, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending searches: %w", err)
	}
	defer rows.Close()

	var stats []domain.SearchStats
	for rows.Next() {
		var s domain.SearchStats
		var avgResults sql.NullFloat64

		if err := rows.Scan(&s.Query, &s.SearchCount, &avgResults); err != nil {
			return nil, fmt.Errorf("failed to scan search stats: %w", err)
		}

		if avgResults.Valid {
			s.AvgResults = avgResults.Float64
		}

		stats = append(stats, s)
	}

	return stats, nil
}
