package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserAdProfile representa el perfil de publicidad de un usuario
type UserAdProfile struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	UserID               uuid.UUID  `json:"user_id" db:"user_id"`
	IsPremium            bool       `json:"is_premium" db:"is_premium"`
	PremiumExpiresAt     *time.Time `json:"premium_expires_at,omitempty" db:"premium_expires_at"`
	AdsShownToday        int        `json:"ads_shown_today" db:"ads_shown_today"`
	AdsShownThisHour     int        `json:"ads_shown_this_hour" db:"ads_shown_this_hour"`
	LastAdShownAt        *time.Time `json:"last_ad_shown_at,omitempty" db:"last_ad_shown_at"`
	TotalAdsShown        int        `json:"total_ads_shown" db:"total_ads_shown"`
	TotalAdClicks        int        `json:"total_ad_clicks" db:"total_ad_clicks"`
	PreferredGenres      []string   `json:"preferred_genres,omitempty" db:"preferred_genres"`
	ListeningTimes       *string    `json:"listening_times,omitempty" db:"listening_times"` // JSONB
	StripeCustomerID     *string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	SubscriptionStatus   *string    `json:"subscription_status,omitempty" db:"subscription_status"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// FrequencyCaps define los límites de frecuencia de anuncios
type FrequencyCaps struct {
	MaxAdsPerHour int
	MaxAdsPerDay  int
}

// DefaultFrequencyCaps retorna los límites por defecto
func DefaultFrequencyCaps() FrequencyCaps {
	return FrequencyCaps{
		MaxAdsPerHour: 6,  // Máximo 6 anuncios por hora
		MaxAdsPerDay:  30, // Máximo 30 anuncios por día
	}
}

// CanShowAd verifica si se puede mostrar un anuncio al usuario
func (p *UserAdProfile) CanShowAd(caps FrequencyCaps) bool {
	// Los usuarios premium no ven anuncios
	if p.IsPremium && p.PremiumExpiresAt != nil && p.PremiumExpiresAt.After(time.Now()) {
		return false
	}

	// Verificar límite por hora
	if p.AdsShownThisHour >= caps.MaxAdsPerHour {
		return false
	}

	// Verificar límite por día
	if p.AdsShownToday >= caps.MaxAdsPerDay {
		return false
	}

	return true
}

// IncrementAdsShown incrementa los contadores de anuncios mostrados
func (p *UserAdProfile) IncrementAdsShown() {
	p.AdsShownToday++
	p.AdsShownThisHour++
	p.TotalAdsShown++
	now := time.Now()
	p.LastAdShownAt = &now
}

// CTR calcula el Click-Through Rate del usuario (0-100)
func (p *UserAdProfile) CTR() float64 {
	if p.TotalAdsShown == 0 {
		return 0
	}
	return (float64(p.TotalAdClicks) / float64(p.TotalAdsShown)) * 100
}

// IsActive verifica si el usuario ha visto anuncios recientemente (últimos 30 días)
func (p *UserAdProfile) IsActive() bool {
	if p.LastAdShownAt == nil {
		return false
	}
	return p.LastAdShownAt.After(time.Now().AddDate(0, 0, -30))
}

// HasPremium verifica si el usuario tiene suscripción premium activa
func (p *UserAdProfile) HasPremium() bool {
	return p.IsPremium && p.PremiumExpiresAt != nil && p.PremiumExpiresAt.After(time.Now())
}

// UserAdProfileRepository define la interfaz para persistencia de perfiles
type UserAdProfileRepository interface {
	Create(profile *UserAdProfile) error
	GetByID(id uuid.UUID) (*UserAdProfile, error)
	GetByUserID(userID uuid.UUID) (*UserAdProfile, error)
	GetByStripeSubscriptionID(subscriptionID string) (*UserAdProfile, error)
	Update(profile *UserAdProfile) error
	IncrementAdsShown(userID uuid.UUID) error
	IncrementAdClicks(userID uuid.UUID) error
	UpdatePremiumStatus(userID uuid.UUID, isPremium bool, expiresAt *time.Time) error
	GetOrCreate(userID uuid.UUID) (*UserAdProfile, error)
	ResetDailyCounters() error  // Llamado por un cron job
	ResetHourlyCounters() error // Llamado por un cron job
}
