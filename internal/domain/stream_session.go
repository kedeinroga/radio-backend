package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// StreamStatus representa el estado de una sesión de streaming
type StreamStatus string

const (
	StreamStatusActive    StreamStatus = "active"
	StreamStatusCompleted StreamStatus = "completed"
	StreamStatusAbandoned StreamStatus = "abandoned"
	StreamStatusError     StreamStatus = "error"
)

// StreamSession representa una sesión de streaming de audio
type StreamSession struct {
	SessionID         uuid.UUID
	UserID            uuid.UUID
	StationID         string     // Station ID es string (no UUID)
	AdID              *uuid.UUID // Nullable - solo si hay anuncio asociado
	StreamToken       string
	TokenExpiresAt    time.Time
	StartedAt         time.Time
	EndedAt           *time.Time
	LastHeartbeat     time.Time
	BytesStreamed     int64
	ListeningDuration time.Duration
	Status            StreamStatus
	UserAgent         string
	IPAddress         string
	CountryCode       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsActive verifica si la sesión está activa
func (s *StreamSession) IsActive() bool {
	return s.Status == StreamStatusActive && s.EndedAt == nil
}

// IsExpired verifica si el token ha expirado
func (s *StreamSession) IsExpired() bool {
	return time.Now().After(s.TokenExpiresAt)
}

// ShouldBeAbandoned verifica si la sesión debería marcarse como abandonada
// (sin heartbeat por más de 60 segundos)
func (s *StreamSession) ShouldBeAbandoned(timeout time.Duration) bool {
	return s.IsActive() && time.Since(s.LastHeartbeat) > timeout
}

// CalculateDuration calcula la duración de escucha
func (s *StreamSession) CalculateDuration() time.Duration {
	if s.EndedAt != nil {
		return s.EndedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
}

// IsValidForAdImpression verifica si la sesión es válida para contar como impresión de anuncio
// Requiere: mínimo 30 segundos de escucha y estado completado
func (s *StreamSession) IsValidForAdImpression() bool {
	minDuration := 30 * time.Second
	return s.AdID != nil &&
		s.Status == StreamStatusCompleted &&
		s.ListeningDuration >= minDuration
}

// StreamSessionRepository define la interfaz para el repositorio de sesiones de streaming
type StreamSessionRepository interface {
	// Create crea una nueva sesión de streaming
	Create(ctx context.Context, session *StreamSession) error

	// GetByID obtiene una sesión por su ID
	GetByID(ctx context.Context, sessionID uuid.UUID) (*StreamSession, error)

	// GetByToken obtiene una sesión por su token de streaming
	GetByToken(ctx context.Context, token string) (*StreamSession, error)

	// Update actualiza una sesión completa
	Update(ctx context.Context, session *StreamSession) error

	// UpdateHeartbeat actualiza el timestamp de last_heartbeat
	UpdateHeartbeat(ctx context.Context, sessionID uuid.UUID) error

	// UpdateMetrics actualiza bytes_streamed y listening_duration
	UpdateMetrics(ctx context.Context, sessionID uuid.UUID, bytes int64, duration time.Duration) error

	// GetActiveSessions obtiene todas las sesiones activas de un usuario
	GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*StreamSession, error)

	// EndSession marca una sesión como terminada con el estado especificado
	EndSession(ctx context.Context, sessionID uuid.UUID, status StreamStatus, duration time.Duration) error

	// CleanupAbandonedSessions marca sesiones sin heartbeat reciente como abandonadas
	// Retorna el número de sesiones limpiadas
	CleanupAbandonedSessions(ctx context.Context, olderThan time.Duration) (int, error)

	// GetSessionsByStation obtiene las sesiones de una estación en un rango de tiempo
	GetSessionsByStation(ctx context.Context, stationID string, from, to time.Time) ([]*StreamSession, error)

	// GetSessionsByUser obtiene las sesiones de un usuario en un rango de tiempo
	GetSessionsByUser(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*StreamSession, error)

	// CountActiveSessionsByUser cuenta las sesiones activas de un usuario
	CountActiveSessionsByUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// StreamAnalytics representa métricas agregadas de streaming
type StreamAnalytics struct {
	Hour               uuid.UUID
	StationID          uuid.UUID
	TotalSessions      int64
	UniqueListeners    int64
	AvgDurationSeconds float64
	TotalBytesStreamed int64
	SessionsWithAds    int64
	CompletedSessions  int64
	AbandonedSessions  int64
}

// StreamMetrics representa métricas en tiempo real de una sesión
type StreamMetrics struct {
	SessionID          uuid.UUID
	BytesPerSecond     float64
	CurrentBitrate     int64
	BufferHealth       float64 // 0.0 - 1.0
	LatencyMs          int64
	DroppedConnections int
}
