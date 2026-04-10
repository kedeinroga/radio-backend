package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"radio-backend/internal/domain"
)

// SecurityRepository implements domain.SecurityRepository
type SecurityRepository struct {
	db *sql.DB
}

// NewSecurityRepository creates a new security repository
func NewSecurityRepository(db *sql.DB) *SecurityRepository {
	return &SecurityRepository{db: db}
}

// GetMetrics retrieves security metrics for a given period
func (r *SecurityRepository) GetMetrics(period string) (*domain.SecurityMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics := &domain.SecurityMetrics{}
	now := time.Now()

	// Calculate date ranges based on period
	var weekStart, todayStart, prevWeekStart time.Time

	switch period {
	case "7d":
		todayStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		weekStart = todayStart.AddDate(0, 0, -7)
		prevWeekStart = weekStart.AddDate(0, 0, -7)
	case "30d":
		todayStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		weekStart = todayStart.AddDate(0, 0, -30)
		prevWeekStart = weekStart.AddDate(0, 0, -30)
	default:
		return nil, fmt.Errorf("invalid period: %s (allowed: 7d, 30d)", period)
	}

	// Get total logins today
	loginQuery := `
		SELECT COUNT(*)
		FROM security_events
		WHERE event_type IN ('login_success', 'token.issued')
		AND timestamp >= $1
		AND timestamp < $2
	`

	err := r.db.QueryRowContext(ctx, loginQuery, todayStart, now).Scan(&metrics.TotalLoginsToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get logins today: %w", err)
	}

	// Get total logins for the period (week or 30 days)
	err = r.db.QueryRowContext(ctx, loginQuery, weekStart, now).Scan(&metrics.TotalLoginsWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get logins for period: %w", err)
	}

	// Get failed attempts today
	failedQuery := `
		SELECT COUNT(*)
		FROM security_events
		WHERE event_type = 'login_failed'
		AND timestamp >= $1
		AND timestamp < $2
	`

	err = r.db.QueryRowContext(ctx, failedQuery, todayStart, now).Scan(&metrics.FailedAttemptsToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed attempts today: %w", err)
	}

	// Get failed attempts for the period
	err = r.db.QueryRowContext(ctx, failedQuery, weekStart, now).Scan(&metrics.FailedAttemptsWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed attempts for period: %w", err)
	}

	// Get active sessions count
	sessionQuery := `
		SELECT COUNT(*)
		FROM sessions
		WHERE is_active = true
		AND expires_at > $1
	`

	err = r.db.QueryRowContext(ctx, sessionQuery, now).Scan(&metrics.ActiveSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	// Get unique locations for the period (based on country from sessions)
	locationQuery := `
		SELECT COUNT(DISTINCT COALESCE(country, 'unknown'))
		FROM sessions
		WHERE created_at >= $1
		AND created_at < $2
	`

	err = r.db.QueryRowContext(ctx, locationQuery, weekStart, now).Scan(&metrics.UniqueLocationsWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique locations: %w", err)
	}

	// Calculate trends (compare with previous period)
	var prevLogins, prevFailedAttempts int64

	// Previous period logins
	err = r.db.QueryRowContext(ctx, loginQuery, prevWeekStart, weekStart).Scan(&prevLogins)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous period logins: %w", err)
	}

	// Previous period failed attempts
	err = r.db.QueryRowContext(ctx, failedQuery, prevWeekStart, weekStart).Scan(&prevFailedAttempts)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous period failed attempts: %w", err)
	}

	// Calculate percentage change
	if prevLogins > 0 {
		metrics.Trends.LoginsTrend = float64(metrics.TotalLoginsWeek-prevLogins) / float64(prevLogins) * 100
	} else if metrics.TotalLoginsWeek > 0 {
		metrics.Trends.LoginsTrend = 100.0 // 100% increase from zero
	}

	if prevFailedAttempts > 0 {
		metrics.Trends.FailedAttemptsTrend = float64(metrics.FailedAttemptsWeek-prevFailedAttempts) / float64(prevFailedAttempts) * 100
	} else if metrics.FailedAttemptsWeek > 0 {
		metrics.Trends.FailedAttemptsTrend = 100.0
	}

	return metrics, nil
}

// GetLogs retrieves security logs with pagination and filtering
func (r *SecurityRepository) GetLogs(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := &domain.SecurityLogResult{
		Page:  filter.Page,
		Limit: filter.Limit,
		Logs:  []*domain.SecurityLog{},
	}

	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.EventType != "" {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIndex))
		args = append(args, filter.EventType)
		argIndex++
	}

	// Search in multiple fields if search query is provided
	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		searchCondition := fmt.Sprintf(`(
			LOWER(se.event_type) LIKE $%d OR
			LOWER(COALESCE(se.ip_address, '')) LIKE $%d OR
			LOWER(COALESCE(se.reason, '')) LIKE $%d OR
			LOWER(COALESCE(u.email, '')) LIKE $%d
		)`, argIndex, argIndex+1, argIndex+2, argIndex+3)
		conditions = append(conditions, searchCondition)
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		argIndex += 4
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM security_events se
		LEFT JOIN users u ON se.user_id = u.id
		%s
	`, whereClause)

	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&result.Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs count: %w", err)
	}

	// Get paginated logs with user email
	offset := (filter.Page - 1) * filter.Limit

	logsQuery := fmt.Sprintf(`
		SELECT
			se.id,
			se.timestamp,
			se.event_type,
			se.user_id,
			u.email,
			se.token_id,
			se.session_id,
			se.ip_address,
			se.user_agent,
			se.reason,
			se.metadata
		FROM security_events se
		LEFT JOIN users u ON se.user_id = u.id
		%s
		ORDER BY se.timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, offset)

	rows, err := r.db.QueryContext(ctx, logsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		log := &domain.SecurityLog{}
		var userID, email, tokenID, sessionID, ipAddress, userAgent, reason sql.NullString
		var metadataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.EventType,
			&userID,
			&email,
			&tokenID,
			&sessionID,
			&ipAddress,
			&userAgent,
			&reason,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		// Handle nullable fields
		if userID.Valid {
			log.UserID = &userID.String
		}
		if email.Valid {
			log.Email = &email.String
		}
		if tokenID.Valid {
			log.TokenID = &tokenID.String
		}
		if sessionID.Valid {
			log.SessionID = &sessionID.String
		}
		if ipAddress.Valid {
			log.IPAddress = &ipAddress.String
		}
		if userAgent.Valid {
			log.UserAgent = &userAgent.String
		}
		if reason.Valid {
			log.Reason = &reason.String
		}

		// Parse metadata JSON
		if metadataJSON != nil {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
				log.Metadata = metadata
			}
		}

		result.Logs = append(result.Logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating logs: %w", err)
	}

	return result, nil
}

// LogSecurityEvent logs a security event
func (r *SecurityRepository) LogSecurityEvent(event *domain.SecurityEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO security_events (
			timestamp, event_type, user_id, token_id, session_id,
			ip_address, user_agent, reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	var userID *string
	if event.UserID != "" {
		userID = &event.UserID
	}

	_, err = r.db.ExecContext(
		ctx, query,
		event.Timestamp,
		event.Event,
		userID,
		event.TokenID,
		nil, // session_id - can be extracted from metadata if needed
		event.IPAddress,
		event.UserAgent,
		event.Reason,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to log security event: %w", err)
	}

	return nil
}

// GetSuspiciousSourceStats returns aggregated statistics for suspicious_request_source events.
// period must be one of: "24h", "7d", "30d".
func (r *SecurityRepository) GetSuspiciousSourceStats(period string) (*domain.SuspiciousSourceStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var since string
	switch period {
	case "24h":
		since = "24 hours"
	case "7d":
		since = "7 days"
	case "30d":
		since = "30 days"
	default:
		return nil, fmt.Errorf("invalid period: %s (allowed: 24h, 7d, 30d)", period)
	}

	stats := &domain.SuspiciousSourceStats{Period: period}

	// Total count
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM security_events
		 WHERE event_type = 'suspicious_request_source'
		   AND timestamp >= NOW() - $1::interval`,
		since,
	).Scan(&stats.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count suspicious events: %w", err)
	}

	// Breakdown by source
	rows, err := r.db.QueryContext(ctx,
		`SELECT COALESCE(metadata->>'source', 'unknown') AS source, COUNT(*) AS cnt
		 FROM security_events
		 WHERE event_type = 'suspicious_request_source'
		   AND timestamp >= NOW() - $1::interval
		 GROUP BY source
		 ORDER BY cnt DESC`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query by source: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sc domain.SourceCount
		if err := rows.Scan(&sc.Source, &sc.Count); err != nil {
			return nil, fmt.Errorf("failed to scan source row: %w", err)
		}
		stats.BySouce = append(stats.BySouce, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating source rows: %w", err)
	}

	// Top IPs (up to 10)
	ipRows, err := r.db.QueryContext(ctx,
		`SELECT COALESCE(ip_address, 'unknown'), COUNT(*) AS cnt, MAX(timestamp)
		 FROM security_events
		 WHERE event_type = 'suspicious_request_source'
		   AND timestamp >= NOW() - $1::interval
		 GROUP BY ip_address
		 ORDER BY cnt DESC
		 LIMIT 10`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query top IPs: %w", err)
	}
	defer ipRows.Close()
	for ipRows.Next() {
		var ic domain.IPCount
		var lastSeen time.Time
		if err := ipRows.Scan(&ic.IP, &ic.Count, &lastSeen); err != nil {
			return nil, fmt.Errorf("failed to scan IP row: %w", err)
		}
		ic.LastSeen = lastSeen.UTC().Format(time.RFC3339)
		stats.TopIPs = append(stats.TopIPs, ic)
	}
	if err := ipRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating IP rows: %w", err)
	}

	// Top targeted paths (up to 10)
	pathRows, err := r.db.QueryContext(ctx,
		`SELECT COALESCE(metadata->>'path', 'unknown') AS path, COUNT(*) AS cnt
		 FROM security_events
		 WHERE event_type = 'suspicious_request_source'
		   AND timestamp >= NOW() - $1::interval
		 GROUP BY path
		 ORDER BY cnt DESC
		 LIMIT 10`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query top paths: %w", err)
	}
	defer pathRows.Close()
	for pathRows.Next() {
		var pc domain.PathCount
		if err := pathRows.Scan(&pc.Path, &pc.Count); err != nil {
			return nil, fmt.Errorf("failed to scan path row: %w", err)
		}
		stats.TopPaths = append(stats.TopPaths, pc)
	}
	if err := pathRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating path rows: %w", err)
	}

	return stats, nil
}
