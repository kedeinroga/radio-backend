package jobs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"
	"radio-backend/internal/infrastructure/icy"
	"radio-backend/internal/services"
)

// NowPlayingJobs maneja la captura periódica de "sonando ahora" y la retención del historial.
type NowPlayingJobs struct {
	nowPlayingService *services.NowPlayingService
	stationService    *services.StationService
	logger            *slog.Logger

	topStations    int
	maxConcurrency int
	retentionDays  int
}

// NewNowPlayingJobs crea una nueva instancia.
func NewNowPlayingJobs(
	nowPlayingService *services.NowPlayingService,
	stationService *services.StationService,
	topStations int,
	maxConcurrency int,
	retentionDays int,
	logger *slog.Logger,
) *NowPlayingJobs {
	if topStations <= 0 {
		topStations = 200
	}
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	if retentionDays <= 0 {
		retentionDays = 30
	}
	return &NowPlayingJobs{
		nowPlayingService: nowPlayingService,
		stationService:    stationService,
		logger:            logger,
		topStations:       topStations,
		maxConcurrency:    maxConcurrency,
		retentionDays:     retentionDays,
	}
}

// PollPopularStations sondea las top-N estaciones populares y captura su "sonando ahora".
// Ejecuta cada 5 minutos (configurable). Acota la concurrencia saliente con un worker pool.
func (j *NowPlayingJobs) PollPopularStations() {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	stations, err := j.stationService.ListPopular(
		ctx,
		j.topStations,
		"", // sin filtro de país
		domain.UserTypeGuest,
		i18n.DefaultLanguage,
	)
	if err != nil {
		j.logger.Error("now-playing poll: failed to list popular stations", "error", err)
		return
	}

	var (
		captured  atomic.Int64
		noMeta    atomic.Int64
		failed    atomic.Int64
		semaphore = make(chan struct{}, j.maxConcurrency)
		wg        sync.WaitGroup
	)

	for _, station := range stations {
		streamURL := station.StreamURLResolved
		if streamURL == "" {
			streamURL = station.StreamURL
		}
		if streamURL == "" {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}
		go func(stationID, url string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			err := j.nowPlayingService.CaptureForStation(ctx, stationID, url)
			switch {
			case err == nil:
				captured.Add(1)
			case errors.Is(err, icy.ErrNoMetadata),
				errors.Is(err, icy.ErrEmptyTitle),
				errors.Is(err, icy.ErrPlaylistUnresolved):
				noMeta.Add(1)
			default:
				failed.Add(1)
				j.logger.Debug("now-playing poll: capture failed",
					"station_id", stationID,
					"error", err,
				)
			}
		}(station.ID, streamURL)
	}

	wg.Wait()

	j.logger.Info("now-playing poll completed",
		"stations", len(stations),
		"captured", captured.Load(),
		"no_metadata", noMeta.Load(),
		"failed", failed.Load(),
	)
}

// CleanupOldTracks elimina el historial de pistas más antiguo que la retención configurada.
// Ejecuta una vez al día.
func (j *NowPlayingJobs) CleanupOldTracks() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	count, err := j.nowPlayingService.CleanupOldTracks(ctx, j.retentionDays)
	if err != nil {
		j.logger.Error("now-playing cleanup failed", "error", err)
		return
	}

	if count > 0 {
		j.logger.Info("old now-playing tracks cleaned", "count", count)
	}
}
