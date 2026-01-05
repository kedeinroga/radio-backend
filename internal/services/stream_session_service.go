package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"radio-backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// StreamSessionService maneja la lógica de negocio para sesiones de streaming
type StreamSessionService struct {
	sessionRepo    domain.StreamSessionRepository
	stationRepo    domain.StationRepository
	adRepo         domain.AdvertisementRepository
	impressionRepo domain.AdImpressionRepository
	jwtSecret      []byte
	tokenDuration  time.Duration
	logger         *slog.Logger
}

// StreamTokenClaims representa los claims del JWT para streaming
type StreamTokenClaims struct {
	SessionID uuid.UUID `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
	StationID string    `json:"station_id"`
	jwt.RegisteredClaims
}

// NewStreamSessionService crea una nueva instancia del servicio
func NewStreamSessionService(
	sessionRepo domain.StreamSessionRepository,
	stationRepo domain.StationRepository,
	adRepo domain.AdvertisementRepository,
	impressionRepo domain.AdImpressionRepository,
	jwtSecret []byte,
	tokenDuration time.Duration,
	logger *slog.Logger,
) *StreamSessionService {
	return &StreamSessionService{
		sessionRepo:    sessionRepo,
		stationRepo:    stationRepo,
		adRepo:         adRepo,
		impressionRepo: impressionRepo,
		jwtSecret:      jwtSecret,
		tokenDuration:  tokenDuration,
		logger:         logger,
	}
}

// StartSession inicia una nueva sesión de streaming
func (s *StreamSessionService) StartSession(
	ctx context.Context,
	userID uuid.UUID,
	stationID string,
	adID *uuid.UUID,
	userAgent string,
	ipAddress string,
) (*domain.StreamSession, string, error) {
	// Verificar que la estación existe
	station, err := s.stationRepo.FindByID(stationID)
	if err != nil {
		return nil, "", fmt.Errorf("station not found: %w", err)
	}

	// Si hay adID, verificar que el anuncio existe
	if adID != nil {
		_, err := s.adRepo.GetByID(*adID)
		if err != nil {
			return nil, "", fmt.Errorf("advertisement not found: %w", err)
		}
	}

	// Verificar límite de sesiones activas (máximo 3)
	activeCount, err := s.sessionRepo.CountActiveSessionsByUser(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to count active sessions: %w", err)
	}
	if activeCount >= 3 {
		return nil, "", fmt.Errorf("maximum active sessions reached (3)")
	}

	// Crear nueva sesión
	sessionID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(s.tokenDuration)

	session := &domain.StreamSession{
		SessionID:         sessionID,
		UserID:            userID,
		StationID:         stationID,
		AdID:              adID,
		StartedAt:         now,
		LastHeartbeat:     now,
		TokenExpiresAt:    expiresAt,
		BytesStreamed:     0,
		ListeningDuration: 0,
		Status:            domain.StreamStatusActive,
		UserAgent:         userAgent,
		IPAddress:         ipAddress,
		CountryCode:       extractCountryCode(ipAddress),
	}

	// Generar JWT token
	token, err := s.generateStreamToken(sessionID, userID, stationID, expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	session.StreamToken = token

	// Guardar sesión en base de datos
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	// Construir stream URL
	streamURL := fmt.Sprintf("/api/v1/stream/proxy?token=%s", token)

	s.logger.Info("stream session started",
		"session_id", sessionID,
		"user_id", userID,
		"station_id", stationID,
		"station_name", station.Name,
		"has_ad", adID != nil,
	)

	return session, streamURL, nil
}

// ValidateToken valida un token de streaming y retorna la sesión
func (s *StreamSessionService) ValidateToken(
	ctx context.Context,
	tokenString string,
) (*domain.StreamSession, error) {
	// Parsear JWT token
	token, err := jwt.ParseWithClaims(tokenString, &StreamTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar método de firma
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*StreamTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Buscar sesión en base de datos
	session, err := s.sessionRepo.GetByID(ctx, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Verificar que la sesión está activa
	if !session.IsActive() {
		return nil, fmt.Errorf("session is not active")
	}

	// Verificar que no ha expirado
	if session.IsExpired() {
		return nil, fmt.Errorf("session has expired")
	}

	// Verificar que el token coincide
	if session.StreamToken != tokenString {
		return nil, fmt.Errorf("token mismatch")
	}

	return session, nil
}

// Heartbeat actualiza el heartbeat de una sesión
func (s *StreamSessionService) Heartbeat(
	ctx context.Context,
	sessionID uuid.UUID,
) error {
	err := s.sessionRepo.UpdateHeartbeat(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	s.logger.Debug("session heartbeat updated", "session_id", sessionID)
	return nil
}

// UpdateMetrics actualiza las métricas de una sesión
func (s *StreamSessionService) UpdateMetrics(
	ctx context.Context,
	sessionID uuid.UUID,
	bytes int64,
	duration time.Duration,
) error {
	err := s.sessionRepo.UpdateMetrics(ctx, sessionID, bytes, duration)
	if err != nil {
		return fmt.Errorf("failed to update metrics: %w", err)
	}

	s.logger.Debug("session metrics updated",
		"session_id", sessionID,
		"bytes", bytes,
		"duration", duration,
	)
	return nil
}

// EndSession termina una sesión y valida impresiones de anuncios si aplica
func (s *StreamSessionService) EndSession(
	ctx context.Context,
	sessionID uuid.UUID,
) error {
	// Obtener sesión actual
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Calcular duración final
	duration := session.CalculateDuration()

	// Marcar sesión como completada
	if err := s.sessionRepo.EndSession(ctx, sessionID, domain.StreamStatusCompleted, duration); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Si hay ad_id y la sesión es válida para impresión, validar la impresión
	if session.IsValidForAdImpression() {
		if err := s.validateAdImpression(ctx, *session.AdID, session.UserID); err != nil {
			s.logger.Error("failed to validate ad impression",
				"error", err,
				"session_id", sessionID,
				"ad_id", session.AdID,
			)
			// No retornar error, la sesión ya fue cerrada exitosamente
		} else {
			s.logger.Info("ad impression validated",
				"session_id", sessionID,
				"ad_id", session.AdID,
				"duration", duration,
			)
		}
	}

	s.logger.Info("stream session ended",
		"session_id", sessionID,
		"user_id", session.UserID,
		"station_id", session.StationID,
		"duration", duration,
		"bytes", session.BytesStreamed,
	)

	return nil
}

// CleanupAbandonedSessions limpia sesiones abandonadas (sin heartbeat reciente)
func (s *StreamSessionService) CleanupAbandonedSessions(ctx context.Context) (int, error) {
	// Sesiones sin heartbeat por más de 60 segundos se consideran abandonadas
	timeout := 60 * time.Second

	count, err := s.sessionRepo.CleanupAbandonedSessions(ctx, timeout)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup abandoned sessions: %w", err)
	}

	if count > 0 {
		s.logger.Info("abandoned sessions cleaned up", "count", count)
	}

	return count, nil
}

// GetActiveSessions obtiene las sesiones activas de un usuario
func (s *StreamSessionService) GetActiveSessions(
	ctx context.Context,
	userID uuid.UUID,
) ([]*domain.StreamSession, error) {
	sessions, err := s.sessionRepo.GetActiveSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	return sessions, nil
}

// GetSessionsByStation obtiene las sesiones de una estación en un rango de tiempo
func (s *StreamSessionService) GetSessionsByStation(
	ctx context.Context,
	stationID string,
	from, to time.Time,
) ([]*domain.StreamSession, error) {
	sessions, err := s.sessionRepo.GetSessionsByStation(ctx, stationID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by station: %w", err)
	}

	return sessions, nil
}

// generateStreamToken genera un JWT token para streaming
func (s *StreamSessionService) generateStreamToken(
	sessionID uuid.UUID,
	userID uuid.UUID,
	stationID string,
	expiresAt time.Time,
) (string, error) {
	claims := StreamTokenClaims{
		SessionID: sessionID,
		UserID:    userID,
		StationID: stationID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "radio-backend",
			Subject:   userID.String(),
			ID:        generateRandomID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// validateAdImpression valida una impresión de anuncio
func (s *StreamSessionService) validateAdImpression(
	ctx context.Context,
	adID uuid.UUID,
	userID uuid.UUID,
) error {
	// Buscar la última impresión no validada de este usuario y anuncio
	// (esto requiere un nuevo método en AdImpressionRepository)
	// Por ahora, simplemente marcamos como validada la impresión más reciente

	// TODO: Implementar método en repository para actualizar impresión específica
	// impression, err := s.impressionRepo.GetLatestUnvalidated(ctx, userID, adID)
	// if err != nil {
	//     return err
	// }
	// return s.impressionRepo.MarkAsValidated(ctx, impression.ID)

	s.logger.Info("ad impression validation skipped (not implemented)",
		"ad_id", adID,
		"user_id", userID,
	)
	return nil
}

// generateRandomID genera un ID aleatorio para JTI
func generateRandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(b)
}

// extractCountryCode extrae el código de país de una IP (stub - requiere GeoIP)
func extractCountryCode(ipAddress string) string {
	// TODO: Implementar con librería de GeoIP (maxminddb, ip2location, etc.)
	return "XX" // Placeholder
}
