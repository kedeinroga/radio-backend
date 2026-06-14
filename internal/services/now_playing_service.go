package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/icy"
)

// NowPlayingService maneja la captura y consulta de "sonando ahora" / "sonó recientemente".
type NowPlayingService struct {
	repo      domain.StationTrackRepository
	icyClient *http.Client
	logger    *slog.Logger
}

// NewNowPlayingService crea una nueva instancia del servicio.
func NewNowPlayingService(
	repo domain.StationTrackRepository,
	fetchTimeout time.Duration,
	logger *slog.Logger,
) *NowPlayingService {
	return &NowPlayingService{
		repo: repo,
		icyClient: &http.Client{
			Timeout: fetchTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		logger: logger,
	}
}

// CaptureForStation obtiene el StreamTitle ICY actual de la estación y, si cambió
// respecto a la última pista registrada, lo persiste como nueva entrada del historial.
//
// Errores "esperables" del stream (sin metadata, sin título) no se propagan como
// fallos del sistema: se reportan vía error para que el job los contabilice, pero
// no son condiciones excepcionales.
func (s *NowPlayingService) CaptureForStation(ctx context.Context, stationID, streamURL string) error {
	if streamURL == "" {
		return fmt.Errorf("station %s has no stream URL", stationID)
	}

	rawTitle, err := icy.FetchNowPlaying(ctx, s.icyClient, streamURL)
	if err != nil {
		return err
	}

	// Dedup: no insertar si el título no cambió respecto al último registrado.
	latest, err := s.repo.GetLatest(ctx, stationID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("failed to read latest track: %w", err)
	}
	if latest != nil && latest.RawTitle == rawTitle {
		return nil
	}

	artist, title := domain.ParseStreamTitle(rawTitle)
	track := &domain.StationTrack{
		StationID: stationID,
		RawTitle:  rawTitle,
		Artist:    artist,
		Title:     title,
		Source:    domain.TrackSourcePoll,
		PlayedAt:  time.Now(),
	}

	if err := s.repo.Insert(ctx, track); err != nil {
		return fmt.Errorf("failed to persist track: %w", err)
	}

	s.logger.Debug("captured now-playing track",
		"station_id", stationID,
		"artist", artist,
		"title", title,
	)

	return nil
}

// GetNowPlaying retorna la pista que suena actualmente en la estación.
// Retorna (nil, nil) si no hay historial.
func (s *NowPlayingService) GetNowPlaying(ctx context.Context, stationID string) (*domain.StationTrack, error) {
	track, err := s.repo.GetLatest(ctx, stationID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get now playing: %w", err)
	}
	return track, nil
}

// GetRecentTracks retorna las últimas pistas de la estación (máx. 50).
func (s *NowPlayingService) GetRecentTracks(ctx context.Context, stationID string, limit int) ([]*domain.StationTrack, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	tracks, err := s.repo.GetRecent(ctx, stationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent tracks: %w", err)
	}
	return tracks, nil
}

// CleanupOldTracks elimina el historial más antiguo que retentionDays.
func (s *NowPlayingService) CleanupOldTracks(ctx context.Context, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	age := time.Duration(retentionDays) * 24 * time.Hour

	count, err := s.repo.DeleteOlderThan(ctx, age)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old tracks: %w", err)
	}
	return count, nil
}
