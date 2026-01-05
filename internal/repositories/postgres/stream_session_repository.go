package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/google/uuid"
)

type StreamSessionRepository struct {
	db *database.Connection
}

func NewStreamSessionRepository(db *database.Connection) *StreamSessionRepository {
	return &StreamSessionRepository{db: db}
}

// Create crea una nueva sesión de streaming
func (r *StreamSessionRepository) Create(ctx context.Context, session *domain.StreamSession) error {
	query := `
		INSERT INTO stream_sessions (
			session_id, user_id, station_id, ad_id,
			stream_token, token_expires_at,
			started_at, last_heartbeat,
			bytes_streamed, listening_duration,
			status, user_agent, ip_address, country_code
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at
	`

	err := r.db.DB.QueryRowContext(
		ctx, query,
		session.SessionID,
		session.UserID,
		session.StationID,
		session.AdID,
		session.StreamToken,
		session.TokenExpiresAt,
		session.StartedAt,
		session.LastHeartbeat,
		session.BytesStreamed,
		session.ListeningDuration,
		session.Status,
		session.UserAgent,
		session.IPAddress,
		session.CountryCode,
	).Scan(&session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create stream session: %w", err)
	}

	return nil
}

// GetByID obtiene una sesión por su ID
func (r *StreamSessionRepository) GetByID(ctx context.Context, sessionID uuid.UUID) (*domain.StreamSession, error) {
	query := `
		SELECT
			session_id, user_id, station_id, ad_id,
			stream_token, token_expires_at,
			started_at, ended_at, last_heartbeat,
			bytes_streamed, listening_duration,
			status, user_agent, ip_address, country_code,
			created_at, updated_at
		FROM stream_sessions
		WHERE session_id = $1
	`

	session := &domain.StreamSession{}
	var listeningDurationStr string

	err := r.db.DB.QueryRowContext(ctx, query, sessionID).Scan(
		&session.SessionID,
		&session.UserID,
		&session.StationID,
		&session.AdID,
		&session.StreamToken,
		&session.TokenExpiresAt,
		&session.StartedAt,
		&session.EndedAt,
		&session.LastHeartbeat,
		&session.BytesStreamed,
		&listeningDurationStr,
		&session.Status,
		&session.UserAgent,
		&session.IPAddress,
		&session.CountryCode,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get stream session: %w", err)
	}

	// Parsear intervalo de PostgreSQL a time.Duration
	session.ListeningDuration, err = parsePostgresInterval(listeningDurationStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse listening duration: %w", err)
	}

	return session, nil
}

// GetByToken obtiene una sesión por su token de streaming
func (r *StreamSessionRepository) GetByToken(ctx context.Context, token string) (*domain.StreamSession, error) {
	query := `
		SELECT
			session_id, user_id, station_id, ad_id,
			stream_token, token_expires_at,
			started_at, ended_at, last_heartbeat,
			bytes_streamed, listening_duration,
			status, user_agent, ip_address, country_code,
			created_at, updated_at
		FROM stream_sessions
		WHERE stream_token = $1
	`

	session := &domain.StreamSession{}
	var listeningDurationStr string

	err := r.db.DB.QueryRowContext(ctx, query, token).Scan(
		&session.SessionID,
		&session.UserID,
		&session.StationID,
		&session.AdID,
		&session.StreamToken,
		&session.TokenExpiresAt,
		&session.StartedAt,
		&session.EndedAt,
		&session.LastHeartbeat,
		&session.BytesStreamed,
		&listeningDurationStr,
		&session.Status,
		&session.UserAgent,
		&session.IPAddress,
		&session.CountryCode,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get stream session by token: %w", err)
	}

	// Parsear intervalo de PostgreSQL a time.Duration
	session.ListeningDuration, err = parsePostgresInterval(listeningDurationStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse listening duration: %w", err)
	}

	return session, nil
}

// Update actualiza una sesión completa
func (r *StreamSessionRepository) Update(ctx context.Context, session *domain.StreamSession) error {
	query := `
		UPDATE stream_sessions SET
			ended_at = $1,
			last_heartbeat = $2,
			bytes_streamed = $3,
			listening_duration = $4,
			status = $5,
			updated_at = NOW()
		WHERE session_id = $6
	`

	result, err := r.db.DB.ExecContext(
		ctx, query,
		session.EndedAt,
		session.LastHeartbeat,
		session.BytesStreamed,
		session.ListeningDuration,
		session.Status,
		session.SessionID,
	)

	if err != nil {
		return fmt.Errorf("failed to update stream session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// UpdateHeartbeat actualiza el timestamp de last_heartbeat
func (r *StreamSessionRepository) UpdateHeartbeat(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE stream_sessions
		SET last_heartbeat = NOW(), updated_at = NOW()
		WHERE session_id = $1 AND status = 'active'
	`

	result, err := r.db.DB.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// UpdateMetrics actualiza bytes_streamed y listening_duration
func (r *StreamSessionRepository) UpdateMetrics(ctx context.Context, sessionID uuid.UUID, bytes int64, duration time.Duration) error {
	query := `
		UPDATE stream_sessions
		SET
			bytes_streamed = $1,
			listening_duration = $2,
			updated_at = NOW()
		WHERE session_id = $3 AND status = 'active'
	`

	result, err := r.db.DB.ExecContext(ctx, query, bytes, duration, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update metrics: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetActiveSessions obtiene todas las sesiones activas de un usuario
func (r *StreamSessionRepository) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*domain.StreamSession, error) {
	query := `
		SELECT
			session_id, user_id, station_id, ad_id,
			stream_token, token_expires_at,
			started_at, ended_at, last_heartbeat,
			bytes_streamed, listening_duration,
			status, user_agent, ip_address, country_code,
			created_at, updated_at
		FROM stream_sessions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY started_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.StreamSession
	for rows.Next() {
		session := &domain.StreamSession{}
		var listeningDurationStr string

		err := rows.Scan(
			&session.SessionID,
			&session.UserID,
			&session.StationID,
			&session.AdID,
			&session.StreamToken,
			&session.TokenExpiresAt,
			&session.StartedAt,
			&session.EndedAt,
			&session.LastHeartbeat,
			&session.BytesStreamed,
			&listeningDurationStr,
			&session.Status,
			&session.UserAgent,
			&session.IPAddress,
			&session.CountryCode,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stream session: %w", err)
		}

		// Parsear intervalo de PostgreSQL a time.Duration
		session.ListeningDuration, err = parsePostgresInterval(listeningDurationStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse listening duration: %w", err)
		}

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stream sessions: %w", err)
	}

	return sessions, nil
}

// EndSession marca una sesión como terminada con el estado especificado
func (r *StreamSessionRepository) EndSession(ctx context.Context, sessionID uuid.UUID, status domain.StreamStatus, duration time.Duration) error {
	query := `
		UPDATE stream_sessions
		SET
			ended_at = NOW(),
			status = $1,
			listening_duration = $2,
			updated_at = NOW()
		WHERE session_id = $3 AND status = 'active'
	`

	result, err := r.db.DB.ExecContext(ctx, query, status, duration, sessionID)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// CleanupAbandonedSessions marca sesiones sin heartbeat reciente como abandonadas
func (r *StreamSessionRepository) CleanupAbandonedSessions(ctx context.Context, olderThan time.Duration) (int, error) {
	query := `
		UPDATE stream_sessions
		SET
			status = 'abandoned',
			ended_at = last_heartbeat,
			listening_duration = last_heartbeat - started_at,
			updated_at = NOW()
		WHERE status = 'active'
		  AND last_heartbeat < NOW() - $1::INTERVAL
	`

	result, err := r.db.DB.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup abandoned sessions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// GetSessionsByStation obtiene las sesiones de una estación en un rango de tiempo
func (r *StreamSessionRepository) GetSessionsByStation(ctx context.Context, stationID string, from, to time.Time) ([]*domain.StreamSession, error) {
	query := `
		SELECT
			session_id, user_id, station_id, ad_id,
			stream_token, token_expires_at,
			started_at, ended_at, last_heartbeat,
			bytes_streamed, listening_duration,
			status, user_agent, ip_address, country_code,
			created_at, updated_at
		FROM stream_sessions
		WHERE station_id = $1
		  AND started_at >= $2
		  AND started_at <= $3
		ORDER BY started_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, stationID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by station: %w", err)
	}
	defer rows.Close()

	return r.scanSessions(rows)
}

// GetSessionsByUser obtiene las sesiones de un usuario en un rango de tiempo
func (r *StreamSessionRepository) GetSessionsByUser(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*domain.StreamSession, error) {
	query := `
		SELECT
			session_id, user_id, station_id, ad_id,
			stream_token, token_expires_at,
			started_at, ended_at, last_heartbeat,
			bytes_streamed, listening_duration,
			status, user_agent, ip_address, country_code,
			created_at, updated_at
		FROM stream_sessions
		WHERE user_id = $1
		  AND started_at >= $2
		  AND started_at <= $3
		ORDER BY started_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user: %w", err)
	}
	defer rows.Close()

	return r.scanSessions(rows)
}

// CountActiveSessionsByUser cuenta las sesiones activas de un usuario
func (r *StreamSessionRepository) CountActiveSessionsByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM stream_sessions
		WHERE user_id = $1 AND status = 'active'
	`

	var count int
	err := r.db.DB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}

	return count, nil
}

// scanSessions es un helper para escanear múltiples sesiones
func (r *StreamSessionRepository) scanSessions(rows *sql.Rows) ([]*domain.StreamSession, error) {
	var sessions []*domain.StreamSession

	for rows.Next() {
		session := &domain.StreamSession{}
		var listeningDurationStr string

		err := rows.Scan(
			&session.SessionID,
			&session.UserID,
			&session.StationID,
			&session.AdID,
			&session.StreamToken,
			&session.TokenExpiresAt,
			&session.StartedAt,
			&session.EndedAt,
			&session.LastHeartbeat,
			&session.BytesStreamed,
			&listeningDurationStr,
			&session.Status,
			&session.UserAgent,
			&session.IPAddress,
			&session.CountryCode,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stream session: %w", err)
		}

		// Parsear intervalo de PostgreSQL a time.Duration
		session.ListeningDuration, err = parsePostgresInterval(listeningDurationStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse listening duration: %w", err)
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stream sessions: %w", err)
	}

	return sessions, nil
}

// parsePostgresInterval parsea un intervalo de PostgreSQL (ej: "00:05:30") a time.Duration
func parsePostgresInterval(interval string) (time.Duration, error) {
	if interval == "" || interval == "00:00:00" {
		return 0, nil
	}

	// PostgreSQL retorna intervalos como "HH:MM:SS" o "X days HH:MM:SS"
	// Para simplificar, asumimos formato "HH:MM:SS"
	duration, err := time.ParseDuration(interval)
	if err != nil {
		// Intentar parsear como segundos si falla
		var seconds float64
		_, err := fmt.Sscanf(interval, "%f", &seconds)
		if err != nil {
			return 0, err
		}
		return time.Duration(seconds) * time.Second, nil
	}

	return duration, nil
}
