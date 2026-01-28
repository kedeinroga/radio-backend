package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AdFormat representa el formato del anuncio
type AdFormat string

const (
	AdFormatBanner       AdFormat = "banner"
	AdFormatInterstitial AdFormat = "interstitial"
	AdFormatAudio        AdFormat = "audio"
	AdFormatNative       AdFormat = "native"
)

// AdType representa el tipo de contenido del anuncio
type AdType string

const (
	AdTypeImage AdType = "image"
	AdTypeVideo AdType = "video"
	AdTypeAudio AdType = "audio"
	AdTypeHTML  AdType = "html"
)

// AdStatus representa el estado del anuncio
type AdStatus string

const (
	AdStatusDraft     AdStatus = "draft"
	AdStatusActive    AdStatus = "active"
	AdStatusPaused    AdStatus = "paused"
	AdStatusCompleted AdStatus = "completed"
	AdStatusExpired   AdStatus = "expired"
)

// Advertisement representa un anuncio publicitario
type Advertisement struct {
	ID                    uuid.UUID `json:"id" db:"id"`
	CampaignID            uuid.UUID `json:"campaign_id" db:"campaign_id"`
	Title                 string    `json:"title" db:"title"`
	Description           *string   `json:"description,omitempty" db:"description"`
	AdvertiserName        string    `json:"advertiser_name" db:"advertiser_name"`
	AdFormat              AdFormat  `json:"ad_format" db:"ad_format"`
	AdType                AdType    `json:"ad_type" db:"ad_type"`
	MediaURL              string    `json:"media_url" db:"media_url"`
	ClickURL              string    `json:"click_url" db:"click_url"`
	CompanionBannerURL    *string   `json:"companion_banner_url,omitempty" db:"companion_banner_url"`
	Width                 *int      `json:"width,omitempty" db:"width"`
	Height                *int      `json:"height,omitempty" db:"height"`
	DurationSeconds       *int      `json:"duration_seconds,omitempty" db:"duration_seconds"`
	TargetCountries       []string  `json:"target_countries,omitempty" db:"target_countries"`
	TargetGenres          []string  `json:"target_genres,omitempty" db:"target_genres"`
	TargetLanguages       []string  `json:"target_languages,omitempty" db:"target_languages"`
	TargetDevices         []string  `json:"target_devices,omitempty" db:"target_devices"`
	StartDate             time.Time `json:"start_date" db:"start_date"`
	EndDate               time.Time `json:"end_date" db:"end_date"`
	DailyBudgetCents      *int      `json:"daily_budget_cents,omitempty" db:"daily_budget_cents"`
	TotalBudgetCents      int       `json:"total_budget_cents" db:"total_budget_cents"`
	CPMRateCents          *int      `json:"cpm_rate_cents,omitempty" db:"cpm_rate_cents"`
	CPCRateCents          *int      `json:"cpc_rate_cents,omitempty" db:"cpc_rate_cents"`
	Status                AdStatus  `json:"status" db:"status"`
	Priority              int       `json:"priority" db:"priority"`
	MaxImpressionsPerUser *int      `json:"max_impressions_per_user,omitempty" db:"max_impressions_per_user"`
	MaxImpressionsPerDay  *int      `json:"max_impressions_per_day,omitempty" db:"max_impressions_per_day"`
	ImpressionsCount      int       `json:"impressions_count" db:"impressions_count"`
	ClicksCount           int       `json:"clicks_count" db:"clicks_count"`
	SpendCents            int       `json:"spend_cents" db:"spend_cents"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// Validate valida los datos del anuncio
func (a *Advertisement) Validate() error {
	if a.Title == "" {
		return ErrInvalidAdTitle
	}
	if !a.AdFormat.IsValid() {
		return ErrInvalidAdFormat
	}
	if !a.AdType.IsValid() {
		return ErrInvalidAdType
	}
	if a.MediaURL == "" {
		return ErrInvalidMediaURL
	}
	if a.ClickURL == "" {
		return ErrInvalidClickURL
	}

	// Validar pricing model
	if a.CPMRateCents == nil && a.CPCRateCents == nil {
		return ErrInvalidPricingModel
	}

	// Validar dimensiones para banners
	if a.AdFormat == AdFormatBanner && (a.Width == nil || a.Height == nil) {
		return ErrInvalidDimensions
	}

	// Validar duración para audio/video
	if (a.AdFormat == AdFormatAudio || a.AdType == AdTypeVideo) && a.DurationSeconds == nil {
		return ErrInvalidDuration
	}

	if a.EndDate.Before(a.StartDate) || a.EndDate.Equal(a.StartDate) {
		return ErrInvalidCampaignDates
	}

	return nil
}

// IsValid valida si el formato es válido
func (f AdFormat) IsValid() bool {
	switch f {
	case AdFormatBanner, AdFormatInterstitial, AdFormatAudio, AdFormatNative:
		return true
	}
	return false
}

// IsValid valida si el tipo es válido
func (t AdType) IsValid() bool {
	switch t {
	case AdTypeImage, AdTypeVideo, AdTypeAudio, AdTypeHTML:
		return true
	}
	return false
}

// IsActive verifica si el anuncio está activo
func (a *Advertisement) IsActive() bool {
	now := time.Now()
	return a.Status == AdStatusActive &&
		now.After(a.StartDate) &&
		now.Before(a.EndDate)
}

// HasBudget verifica si el anuncio tiene presupuesto disponible
func (a *Advertisement) HasBudget() bool {
	return a.SpendCents < a.TotalBudgetCents
}

// CTR calcula el Click-Through Rate (0-100)
func (a *Advertisement) CTR() float64 {
	if a.ImpressionsCount == 0 {
		return 0
	}
	return (float64(a.ClicksCount) / float64(a.ImpressionsCount)) * 100
}

// CalculateCost calcula el costo de una impresión o click
func (a *Advertisement) CalculateCost(isClick bool) int {
	if isClick && a.CPCRateCents != nil {
		return *a.CPCRateCents
	}
	if !isClick && a.CPMRateCents != nil {
		// CPM es por 1000 impresiones
		return *a.CPMRateCents / 1000
	}
	return 0
}

// MatchesTargeting verifica si el anuncio coincide con los criterios de targeting
func (a *Advertisement) MatchesTargeting(country, genre, language, device string) bool {
	// Si no hay targeting específico, coincide con todo
	if len(a.TargetCountries) > 0 && !contains(a.TargetCountries, country) {
		return false
	}
	if len(a.TargetGenres) > 0 && !contains(a.TargetGenres, genre) {
		return false
	}
	if len(a.TargetLanguages) > 0 && !contains(a.TargetLanguages, language) {
		return false
	}
	if len(a.TargetDevices) > 0 && !contains(a.TargetDevices, device) {
		return false
	}
	return true
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// AdvertisementRepository define la interfaz para persistencia de anuncios
type AdvertisementRepository interface {
	Create(ctx context.Context, ad *Advertisement) error
	GetByID(ctx context.Context, id uuid.UUID) (*Advertisement, error)
	GetByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*Advertisement, error)
	Update(ctx context.Context, ad *Advertisement) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveAds(ctx context.Context) ([]*Advertisement, error)
	GetEligibleAds(ctx context.Context, country, genre, language, device string) ([]*Advertisement, error)
	IncrementImpressions(ctx context.Context, adID uuid.UUID) error
	IncrementClicks(ctx context.Context, adID uuid.UUID) error
	IncrementSpend(ctx context.Context, adID uuid.UUID, amountCents int) error
}
