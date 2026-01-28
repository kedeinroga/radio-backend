package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AudioProxyService maneja el proxy de streams de audio
type AudioProxyService struct {
	sessionService *StreamSessionService
	httpClient     *http.Client
	bufferSize     int
	updateInterval time.Duration
	logger         *slog.Logger
	activeStreams  sync.Map // sessionID -> *activeStream
}

// activeStream representa un stream activo en memoria
type activeStream struct {
	sessionID     uuid.UUID
	startTime     time.Time
	bytesStreamed int64
	lastUpdate    time.Time
	cancel        context.CancelFunc
}

// NewAudioProxyService crea una nueva instancia del servicio
func NewAudioProxyService(
	sessionService *StreamSessionService,
	logger *slog.Logger,
) *AudioProxyService {
	return &AudioProxyService{
		sessionService: sessionService,
		httpClient: &http.Client{
			Timeout: 0, // Sin timeout para streams largos
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		bufferSize:     64 * 1024, // 64KB buffer
		updateInterval: 10 * time.Second,
		logger:         logger,
	}
}

// ProxyStream hace proxy de un stream de audio desde la URL original al cliente
func (s *AudioProxyService) ProxyStream(
	ctx context.Context,
	token string,
	writer http.ResponseWriter,
	request *http.Request,
) error {
	// Validar token y obtener sesión
	session, err := s.sessionService.ValidateToken(ctx, token)
	if err != nil {
		s.logger.Warn("invalid stream token", "error", err)
		http.Error(writer, "Invalid or expired token", http.StatusUnauthorized)
		return err
	}

	// Obtener URL del stream original de la estación
	station, err := s.sessionService.stationRepo.FindByID(ctx, session.StationID)
	if err != nil {
		s.logger.Error("failed to get station", "error", err, "station_id", session.StationID)
		http.Error(writer, "Station not found", http.StatusNotFound)
		return err
	}

	if station.StreamURL == "" {
		s.logger.Error("station has no stream URL", "station_id", session.StationID)
		http.Error(writer, "Station stream not available", http.StatusServiceUnavailable)
		return fmt.Errorf("no stream URL for station")
	}

	s.logger.Info("starting stream proxy",
		"session_id", session.SessionID,
		"station_id", session.StationID,
		"station_name", station.Name,
		"stream_url", station.StreamURL,
	)

	// Crear contexto con cancelación
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Registrar stream activo
	active := &activeStream{
		sessionID:     session.SessionID,
		startTime:     time.Now(),
		bytesStreamed: 0,
		lastUpdate:    time.Now(),
		cancel:        cancel,
	}
	s.activeStreams.Store(session.SessionID, active)
	defer s.activeStreams.Delete(session.SessionID)

	// Iniciar goroutine para actualizar métricas periódicamente
	metricsDone := make(chan struct{})
	go s.updateMetricsPeriodically(streamCtx, session.SessionID, active, metricsDone)
	defer func() {
		<-metricsDone // Esperar a que termine la actualización de métricas
	}()

	// Conectar al stream original
	streamReq, err := http.NewRequestWithContext(streamCtx, "GET", station.StreamURL, nil)
	if err != nil {
		s.logger.Error("failed to create stream request", "error", err)
		http.Error(writer, "Failed to connect to stream", http.StatusInternalServerError)
		return err
	}

	// Copiar headers del cliente original (User-Agent, etc.)
	if userAgent := request.Header.Get("User-Agent"); userAgent != "" {
		streamReq.Header.Set("User-Agent", userAgent)
	}
	streamReq.Header.Set("Icy-MetaData", "1") // Solicitar metadata de Shoutcast/Icecast

	// Realizar request al stream
	resp, err := s.httpClient.Do(streamReq)
	if err != nil {
		s.logger.Error("failed to connect to stream", "error", err, "url", station.StreamURL)
		http.Error(writer, "Failed to connect to stream", http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		s.logger.Error("stream returned non-200 status",
			"status", resp.StatusCode,
			"url", station.StreamURL,
		)
		http.Error(writer, "Stream not available", http.StatusBadGateway)
		return fmt.Errorf("stream returned status %d", resp.StatusCode)
	}

	// Copiar headers de respuesta del stream al cliente
	for key, values := range resp.Header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}

	// Asegurar headers importantes para streaming
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	writer.Header().Set("Pragma", "no-cache")
	writer.Header().Set("Expires", "0")
	writer.Header().Set("Connection", "keep-alive")

	// Escribir headers
	writer.WriteHeader(http.StatusOK)

	// Flusher para enviar datos inmediatamente
	flusher, ok := writer.(http.Flusher)
	if !ok {
		s.logger.Error("writer does not support flushing")
		return fmt.Errorf("streaming unsupported")
	}

	// Buffer para lectura
	buffer := make([]byte, s.bufferSize)

	// Copiar stream chunk por chunk
	for {
		select {
		case <-streamCtx.Done():
			s.logger.Info("stream context cancelled", "session_id", session.SessionID)
			return streamCtx.Err()
		default:
			// Leer chunk del stream original
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				// Escribir chunk al cliente
				written, writeErr := writer.Write(buffer[:n])
				if writeErr != nil {
					s.logger.Error("failed to write to client",
						"error", writeErr,
						"session_id", session.SessionID,
					)
					return writeErr
				}

				// Actualizar contador de bytes
				active.bytesStreamed += int64(written)

				// Flush para enviar datos inmediatamente
				flusher.Flush()
			}

			// Manejar errores de lectura
			if err == io.EOF {
				s.logger.Info("stream ended (EOF)", "session_id", session.SessionID)
				return nil
			}
			if err != nil {
				s.logger.Error("error reading from stream",
					"error", err,
					"session_id", session.SessionID,
				)
				return err
			}
		}
	}
}

// updateMetricsPeriodically actualiza las métricas de la sesión periódicamente
func (s *AudioProxyService) updateMetricsPeriodically(
	ctx context.Context,
	sessionID uuid.UUID,
	active *activeStream,
	done chan struct{},
) {
	defer close(done)

	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Última actualización antes de salir
			s.updateSessionMetrics(sessionID, active)
			return
		case <-ticker.C:
			s.updateSessionMetrics(sessionID, active)
		}
	}
}

// updateSessionMetrics actualiza las métricas en la base de datos
func (s *AudioProxyService) updateSessionMetrics(sessionID uuid.UUID, active *activeStream) {
	duration := time.Since(active.startTime)
	bytes := active.bytesStreamed

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.sessionService.UpdateMetrics(ctx, sessionID, bytes, duration)
	if err != nil {
		s.logger.Error("failed to update session metrics",
			"error", err,
			"session_id", sessionID,
		)
		return
	}

	active.lastUpdate = time.Now()

	s.logger.Debug("session metrics updated",
		"session_id", sessionID,
		"bytes", bytes,
		"duration", duration,
		"bitrate_kbps", calculateBitrate(bytes, duration),
	)
}

// GetActiveStreamCount retorna el número de streams activos
func (s *AudioProxyService) GetActiveStreamCount() int {
	count := 0
	s.activeStreams.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetActiveStreams retorna información sobre streams activos
func (s *AudioProxyService) GetActiveStreams() []StreamInfo {
	var streams []StreamInfo
	s.activeStreams.Range(func(key, value interface{}) bool {
		active := value.(*activeStream)
		duration := time.Since(active.startTime)
		bitrate := calculateBitrate(active.bytesStreamed, duration)

		streams = append(streams, StreamInfo{
			SessionID:     active.sessionID,
			Duration:      duration,
			BytesStreamed: active.bytesStreamed,
			BitrateKbps:   bitrate,
			LastUpdate:    active.lastUpdate,
		})
		return true
	})
	return streams
}

// CancelStream cancela un stream activo
func (s *AudioProxyService) CancelStream(sessionID uuid.UUID) error {
	value, ok := s.activeStreams.Load(sessionID)
	if !ok {
		return fmt.Errorf("stream not found")
	}

	active := value.(*activeStream)
	active.cancel()

	s.logger.Info("stream cancelled", "session_id", sessionID)
	return nil
}

// StreamInfo representa información sobre un stream activo
type StreamInfo struct {
	SessionID     uuid.UUID
	Duration      time.Duration
	BytesStreamed int64
	BitrateKbps   float64
	LastUpdate    time.Time
}

// calculateBitrate calcula el bitrate en kbps
func calculateBitrate(bytes int64, duration time.Duration) float64 {
	if duration.Seconds() == 0 {
		return 0
	}
	// (bytes * 8 bits/byte) / (duration in seconds) / 1000 = kbps
	return float64(bytes*8) / duration.Seconds() / 1000
}

// HealthCheck verifica la salud del servicio de proxy
func (s *AudioProxyService) HealthCheck() bool {
	// Verificar que el cliente HTTP está funcional
	return s.httpClient != nil
}
