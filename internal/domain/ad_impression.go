package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AdImpression representa una impresión de anuncio
type AdImpression struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	AdvertisementID      uuid.UUID  `json:"advertisement_id" db:"advertisement_id"`
	UserID               *uuid.UUID `json:"user_id,omitempty" db:"user_id"` // NULL para usuarios no autenticados
	SessionID            string     `json:"session_id" db:"session_id"`
	StationID            *string    `json:"station_id,omitempty" db:"station_id"`
	IPAddress            string     `json:"ip_address" db:"ip_address"`
	UserAgent            string     `json:"user_agent" db:"user_agent"`
	CountryCode          *string    `json:"country_code,omitempty" db:"country_code"`
	City                 *string    `json:"city,omitempty" db:"city"`
	DeviceType           string     `json:"device_type" db:"device_type"` // mobile, tablet, desktop
	Viewable             bool       `json:"viewable" db:"viewable"`
	ImpressionDurationMs *int       `json:"impression_duration_ms,omitempty" db:"impression_duration_ms"`
	ImpressionToken      string     `json:"impression_token" db:"impression_token"` // HMAC token para validación
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

// Validate valida los datos de la impresión
func (i *AdImpression) Validate() error {
	if i.AdvertisementID == uuid.Nil {
		return ErrInvalidImpressionData
	}
	if i.SessionID == "" {
		return ErrInvalidImpressionData
	}
	if i.IPAddress == "" {
		return ErrInvalidImpressionData
	}
	if i.UserAgent == "" {
		return ErrInvalidImpressionData
	}
	if i.ImpressionToken == "" {
		return ErrInvalidImpressionToken
	}
	return nil
}

// IsViewable determina si la impresión es considerada viewable según estándares IAB
// (debe estar visible al menos 1 segundo)
func (i *AdImpression) IsViewable() bool {
	if !i.Viewable {
		return false
	}
	if i.ImpressionDurationMs == nil {
		return false
	}
	// IAB standard: al menos 1000ms visible
	return *i.ImpressionDurationMs >= 1000
}

// AdImpressionRepository define la interfaz para persistencia de impresiones
type AdImpressionRepository interface {
	Create(ctx context.Context, impression *AdImpression) error
	GetByID(ctx context.Context, id uuid.UUID) (*AdImpression, error)
	GetByAdvertisementID(ctx context.Context, adID uuid.UUID, limit int) ([]*AdImpression, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, since time.Time) ([]*AdImpression, error)
	CountByAdvertisementID(ctx context.Context, adID uuid.UUID, since time.Time) (int64, error)
	CountByIPAddress(ctx context.Context, ipAddress string, since time.Time) (int64, error)
	CountViewableImpressions(ctx context.Context, adID uuid.UUID, since time.Time) (int64, error)
	GetRecentBySessionID(ctx context.Context, sessionID string, since time.Time) ([]*AdImpression, error)
}
