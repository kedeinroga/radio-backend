package domain

import (
	"time"

	"github.com/google/uuid"
)

// AdClick representa un click en un anuncio
type AdClick struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	ImpressionID         uuid.UUID  `json:"impression_id" db:"impression_id"`
	AdvertisementID      uuid.UUID  `json:"advertisement_id" db:"advertisement_id"`
	UserID               *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	IPAddress            string     `json:"ip_address" db:"ip_address"`
	UserAgent            string     `json:"user_agent" db:"user_agent"`
	Referrer             *string    `json:"referrer,omitempty" db:"referrer"`
	ClickPosition        *string    `json:"click_position,omitempty" db:"click_position"` // e.g., "top-banner", "mid-roll"
	Converted            bool       `json:"converted" db:"converted"`
	ConversionValueCents *int       `json:"conversion_value_cents,omitempty" db:"conversion_value_cents"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

// Validate valida los datos del click
func (c *AdClick) Validate() error {
	if c.ImpressionID == uuid.Nil {
		return ErrInvalidClickData
	}
	if c.AdvertisementID == uuid.Nil {
		return ErrInvalidClickData
	}
	if c.IPAddress == "" {
		return ErrInvalidClickData
	}
	if c.UserAgent == "" {
		return ErrInvalidClickData
	}
	return nil
}

// TimeToClick calcula el tiempo desde la impresión hasta el click
func (c *AdClick) TimeToClick(impression *AdImpression) time.Duration {
	if impression == nil {
		return 0
	}
	return c.CreatedAt.Sub(impression.CreatedAt)
}

// IsSuspicious detecta patrones sospechosos de clicks
func (c *AdClick) IsSuspicious(impression *AdImpression) bool {
	if impression == nil {
		return true
	}

	// Click muy rápido (< 100ms) es sospechoso
	timeToClick := c.TimeToClick(impression)
	if timeToClick < 100*time.Millisecond {
		return true
	}

	// IPs diferentes entre impresión y click es sospechoso
	if c.IPAddress != impression.IPAddress {
		return true
	}

	// User agents diferentes es sospechoso
	if c.UserAgent != impression.UserAgent {
		return true
	}

	return false
}

// AdClickRepository define la interfaz para persistencia de clicks
type AdClickRepository interface {
	Create(click *AdClick) error
	GetByID(id uuid.UUID) (*AdClick, error)
	GetByAdvertisementID(adID uuid.UUID, limit int) ([]*AdClick, error)
	GetByImpressionID(impressionID uuid.UUID) (*AdClick, error)
	CountByAdvertisementID(adID uuid.UUID, since time.Time) (int64, error)
	CountByIPAddress(ipAddress string, since time.Time) (int64, error)
	UpdateConversion(clickID uuid.UUID, valueCents int) error
	GetRecentByIPAddress(ipAddress string, since time.Time) ([]*AdClick, error)
}
