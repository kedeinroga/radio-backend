package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TrackSource indica el mecanismo de captura de una pista.
type TrackSource string

const (
	// TrackSourcePoll indica que la pista fue capturada por el job de sondeo.
	TrackSourcePoll TrackSource = "poll"
	// TrackSourceProxy indica que la pista fue capturada desde el proxy de audio.
	TrackSourceProxy TrackSource = "proxy"
)

// StationTrack representa una canción detectada sonando en una estación.
type StationTrack struct {
	ID        uuid.UUID
	StationID string
	RawTitle  string // StreamTitle ICY crudo
	Artist    string // Parseado desde RawTitle (puede ir vacío)
	Title     string // Parseado desde RawTitle
	Source    TrackSource
	PlayedAt  time.Time
	CreatedAt time.Time
}

// StationTrackRepository define la interfaz para el historial de pistas por estación.
type StationTrackRepository interface {
	// Insert guarda una nueva pista detectada.
	Insert(ctx context.Context, track *StationTrack) error

	// GetLatest obtiene la última pista detectada de una estación ("now playing").
	// Retorna ErrNotFound si la estación no tiene historial.
	GetLatest(ctx context.Context, stationID string) (*StationTrack, error)

	// GetRecent obtiene las últimas pistas de una estación ordenadas por played_at DESC.
	GetRecent(ctx context.Context, stationID string, limit int) ([]*StationTrack, error)

	// DeleteOlderThan elimina pistas más antiguas que la duración indicada.
	// Retorna el número de filas eliminadas.
	DeleteOlderThan(ctx context.Context, age time.Duration) (int, error)
}

// ParseStreamTitle separa un StreamTitle ICY crudo en artista y título.
//
// Formatos típicos:
//
//	"Artist - Song"       -> artist="Artist", title="Song"
//	"Song"                -> artist="",       title="Song"
//	"Artist - Song - Live"-> artist="Artist", title="Song - Live"
//
// El primer " - " se usa como separador; el resto se conserva en el título.
func ParseStreamTitle(raw string) (artist, title string) {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return "", ""
	}

	if idx := strings.Index(cleaned, " - "); idx >= 0 {
		artist = strings.TrimSpace(cleaned[:idx])
		title = strings.TrimSpace(cleaned[idx+len(" - "):])
		// Si una de las partes queda vacía, tratar todo como título.
		if artist == "" || title == "" {
			return "", cleaned
		}
		return artist, title
	}

	return "", cleaned
}
