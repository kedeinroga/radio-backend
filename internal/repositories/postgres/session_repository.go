package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"radio-backend/internal/domain"
)

// SessionRepository implements domain.SessionRepository
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(session *domain.Session) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deviceInfoJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal device info: %w", err)
	}

	browser := session.DeviceInfo["browser"]
	os := session.DeviceInfo["os"]
	deviceType := session.DeviceInfo["device_type"]
	country := session.DeviceInfo["country"]
	city := session.DeviceInfo["city"]

	query := `
		INSERT INTO sessions (
			user_id, session_id, token_id, ip_address, user_agent,
			browser, os, device_type, country, city,
			created_at, last_activity, expires_at, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	err = r.db.QueryRowContext(
		ctx, query,
		session.UserID, session.SessionID, session.TokenID, session.IPAddress, session.UserAgent,
		browser, os, deviceType, country, city,
		session.CreatedAt, session.LastActivity, session.ExpiresAt, session.IsActive,
	).Scan(&session.ID)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Store full device info as JSON if needed
	_ = deviceInfoJSON

	return nil
}

// FindByID finds a session by ID
func (r *SessionRepository) FindByID(sessionID string) (*domain.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, session_id, token_id, ip_address, user_agent,
		       browser, os, device_type, country, city,
		       created_at, last_activity, expires_at, is_active
		FROM sessions
		WHERE session_id = $1
	`

	session := &domain.Session{
		DeviceInfo: make(map[string]string),
	}
	var browser, os, deviceType, country, city sql.NullString

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.UserID, &session.SessionID, &session.TokenID,
		&session.IPAddress, &session.UserAgent,
		&browser, &os, &deviceType, &country, &city,
		&session.CreatedAt, &session.LastActivity, &session.ExpiresAt, &session.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if browser.Valid {
		session.DeviceInfo["browser"] = browser.String
	}
	if os.Valid {
		session.DeviceInfo["os"] = os.String
	}
	if deviceType.Valid {
		session.DeviceInfo["device_type"] = deviceType.String
	}
	if country.Valid {
		session.DeviceInfo["country"] = country.String
	}
	if city.Valid {
		session.DeviceInfo["city"] = city.String
	}

	return session, nil
}

// FindByUserID finds all sessions for a user
func (r *SessionRepository) FindByUserID(userID string) ([]*domain.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, session_id, token_id, ip_address, user_agent,
		       browser, os, device_type, country, city,
		       created_at, last_activity, expires_at, is_active
		FROM sessions
		WHERE user_id = $1 AND is_active = true
		ORDER BY last_activity DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		session := &domain.Session{
			DeviceInfo: make(map[string]string),
		}
		var browser, os, deviceType, country, city sql.NullString

		err := rows.Scan(
			&session.ID, &session.UserID, &session.SessionID, &session.TokenID,
			&session.IPAddress, &session.UserAgent,
			&browser, &os, &deviceType, &country, &city,
			&session.CreatedAt, &session.LastActivity, &session.ExpiresAt, &session.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if browser.Valid {
			session.DeviceInfo["browser"] = browser.String
		}
		if os.Valid {
			session.DeviceInfo["os"] = os.String
		}
		if deviceType.Valid {
			session.DeviceInfo["device_type"] = deviceType.String
		}
		if country.Valid {
			session.DeviceInfo["country"] = country.String
		}
		if city.Valid {
			session.DeviceInfo["city"] = city.String
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// FindByTokenID finds a session by token ID
func (r *SessionRepository) FindByTokenID(tokenID string) (*domain.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, session_id, token_id, ip_address, user_agent,
		       browser, os, device_type, country, city,
		       created_at, last_activity, expires_at, is_active
		FROM sessions
		WHERE token_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	session := &domain.Session{
		DeviceInfo: make(map[string]string),
	}
	var browser, os, deviceType, country, city sql.NullString

	err := r.db.QueryRowContext(ctx, query, tokenID).Scan(
		&session.ID, &session.UserID, &session.SessionID, &session.TokenID,
		&session.IPAddress, &session.UserAgent,
		&browser, &os, &deviceType, &country, &city,
		&session.CreatedAt, &session.LastActivity, &session.ExpiresAt, &session.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if browser.Valid {
		session.DeviceInfo["browser"] = browser.String
	}
	if os.Valid {
		session.DeviceInfo["os"] = os.String
	}
	if deviceType.Valid {
		session.DeviceInfo["device_type"] = deviceType.String
	}
	if country.Valid {
		session.DeviceInfo["country"] = country.String
	}
	if city.Valid {
		session.DeviceInfo["city"] = city.String
	}

	return session, nil
}

// UpdateLastActivity updates the last activity timestamp
func (r *SessionRepository) UpdateLastActivity(sessionID string, lastActivity time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE sessions
		SET last_activity = $1
		WHERE session_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, lastActivity, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update last activity: %w", err)
	}

	return nil
}

// Delete deletes a session
func (r *SessionRepository) Delete(sessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE sessions
		SET is_active = false
		WHERE session_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteByUserID deletes all sessions for a user
func (r *SessionRepository) DeleteByUserID(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE sessions
		SET is_active = false
		WHERE user_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// DeleteExpired deletes expired sessions
func (r *SessionRepository) DeleteExpired() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE sessions
		SET is_active = false
		WHERE expires_at < NOW() AND is_active = true
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

// SecurityEventRepository handles security event logging
type SecurityEventRepository struct {
	db *sql.DB
}

// NewSecurityEventRepository creates a new security event repository
func NewSecurityEventRepository(db *sql.DB) *SecurityEventRepository {
	return &SecurityEventRepository{db: db}
}

// Create creates a new security event
func (r *SecurityEventRepository) Create(event *domain.SecurityEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO security_events (
			timestamp, event_type, user_id, token_id, 
			ip_address, user_agent, reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.ExecContext(
		ctx, query,
		event.Timestamp, event.Event, event.UserID, event.TokenID,
		event.IPAddress, event.UserAgent, event.Reason, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create security event: %w", err)
	}

	return nil
}
