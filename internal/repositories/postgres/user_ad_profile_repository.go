package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UserAdProfileRepository implementa domain.UserAdProfileRepository
type UserAdProfileRepository struct {
	db *database.Connection
}

// NewUserAdProfileRepository crea una nueva instancia del repositorio
func NewUserAdProfileRepository(db *database.Connection) *UserAdProfileRepository {
	return &UserAdProfileRepository{db: db}
}

// GetOrCreate obtiene un perfil o lo crea si no existe
func (r *UserAdProfileRepository) GetOrCreate(userID uuid.UUID) (*domain.UserAdProfile, error) {
	// Primero intentamos obtenerlo
	profile, err := r.GetByUserID(userID)
	if err == nil {
		return profile, nil
	}

	// Si no existe, lo creamos
	now := time.Now()
	profile = &domain.UserAdProfile{
		ID:                   uuid.New(),
		UserID:               userID,
		IsPremium:            false,
		PremiumExpiresAt:     nil,
		AdsShownToday:        0,
		AdsShownThisHour:     0,
		LastAdShownAt:        nil,
		TotalAdsShown:        0,
		TotalAdClicks:        0,
		PreferredGenres:      []string{},
		ListeningTimes:       nil,
		StripeCustomerID:     nil,
		StripeSubscriptionID: nil,
		SubscriptionStatus:   nil,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	err = r.Create(profile)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// GetByID obtiene un perfil por ID
func (r *UserAdProfileRepository) GetByID(id uuid.UUID) (*domain.UserAdProfile, error) {
	query := `
		SELECT
			id, user_id, is_premium, premium_until,
			ads_shown_today, ads_shown_this_hour,
			last_ad_shown_at, total_ad_impressions, total_ad_clicks,
			favorite_genres, listening_times,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		FROM user_ad_profiles
		WHERE id = $1
	`

	return r.scanProfile(r.db.DB.QueryRow(query, id))
}

// GetByUserID obtiene un perfil por user_id
func (r *UserAdProfileRepository) GetByUserID(userID uuid.UUID) (*domain.UserAdProfile, error) {
	query := `
		SELECT
			id, user_id, is_premium, premium_until,
			ads_shown_today, ads_shown_this_hour,
			last_ad_shown_at, total_ad_impressions, total_ad_clicks,
			favorite_genres, listening_times,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		FROM user_ad_profiles
		WHERE user_id = $1
	`

	return r.scanProfile(r.db.DB.QueryRow(query, userID))
}

// GetByStripeSubscriptionID obtiene un perfil por stripe_subscription_id
func (r *UserAdProfileRepository) GetByStripeSubscriptionID(subscriptionID string) (*domain.UserAdProfile, error) {
	query := `
		SELECT
			id, user_id, is_premium, premium_until,
			ads_shown_today, ads_shown_this_hour,
			last_ad_shown_at, total_ad_impressions, total_ad_clicks,
			favorite_genres, listening_times,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		FROM user_ad_profiles
		WHERE stripe_subscription_id = $1
	`

	return r.scanProfile(r.db.DB.QueryRow(query, subscriptionID))
}

// Create crea un nuevo perfil
func (r *UserAdProfileRepository) Create(profile *domain.UserAdProfile) error {
	query := `
		INSERT INTO user_ad_profiles (
			id, user_id, is_premium, premium_until,
			ads_shown_today, ads_shown_this_hour,
			last_ad_shown_at, total_ad_impressions, total_ad_clicks,
			favorite_genres, listening_times,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err := r.db.DB.Exec(query,
		profile.ID,
		profile.UserID,
		profile.IsPremium,
		profile.PremiumExpiresAt,
		profile.AdsShownToday,
		profile.AdsShownThisHour,
		profile.LastAdShownAt,
		profile.TotalAdsShown,
		profile.TotalAdClicks,
		pq.Array(profile.PreferredGenres),
		profile.ListeningTimes,
		profile.StripeCustomerID,
		profile.StripeSubscriptionID,
		time.Now(),
		time.Now(),
	)

	return err
}

// Update actualiza un perfil existente
func (r *UserAdProfileRepository) Update(profile *domain.UserAdProfile) error {
	query := `
		UPDATE user_ad_profiles
		SET
			is_premium = $2,
			premium_until = $3,
			ads_shown_today = $4,
			ads_shown_this_hour = $5,
			last_ad_shown_at = $6,
			total_ad_impressions = $7,
			total_ad_clicks = $8,
			favorite_genres = $9,
			listening_times = $10,
			stripe_customer_id = $11,
			stripe_subscription_id = $12,
			updated_at = $13
		WHERE id = $1
	`

	_, err := r.db.DB.Exec(query,
		profile.ID,
		profile.IsPremium,
		profile.PremiumExpiresAt,
		profile.AdsShownToday,
		profile.AdsShownThisHour,
		profile.LastAdShownAt,
		profile.TotalAdsShown,
		profile.TotalAdClicks,
		pq.Array(profile.PreferredGenres),
		profile.ListeningTimes,
		profile.StripeCustomerID,
		profile.StripeSubscriptionID,
		time.Now(),
	)

	return err
}

// IncrementAdsShown incrementa el contador de anuncios mostrados
func (r *UserAdProfileRepository) IncrementAdsShown(userID uuid.UUID) error {
	query := `
		UPDATE user_ad_profiles
		SET
			ads_shown_today = ads_shown_today + 1,
			ads_shown_this_hour = ads_shown_this_hour + 1,
			total_ad_impressions = total_ad_impressions + 1,
			last_ad_shown_at = $2,
			updated_at = $2
		WHERE user_id = $1
	`

	_, err := r.db.DB.Exec(query, userID, time.Now())
	return err
}

// IncrementAdClicks incrementa el contador de clicks
func (r *UserAdProfileRepository) IncrementAdClicks(userID uuid.UUID) error {
	query := `
		UPDATE user_ad_profiles
		SET
			total_ad_clicks = total_ad_clicks + 1,
			updated_at = $2
		WHERE user_id = $1
	`

	_, err := r.db.DB.Exec(query, userID, time.Now())
	return err
}

// UpdatePremiumStatus actualiza el estado premium
func (r *UserAdProfileRepository) UpdatePremiumStatus(userID uuid.UUID, isPremium bool, expiresAt *time.Time) error {
	query := `
		UPDATE user_ad_profiles
		SET
			is_premium = $2,
			premium_until = $3,
			updated_at = $4
		WHERE user_id = $1
	`

	_, err := r.db.DB.Exec(query, userID, isPremium, expiresAt, time.Now())
	return err
}

// ResetDailyCounters resetea los contadores diarios (llamado por cron job)
func (r *UserAdProfileRepository) ResetDailyCounters() error {
	query := `
		UPDATE user_ad_profiles
		SET
			ads_shown_today = 0,
			ads_shown_today_reset_at = CURRENT_DATE,
			updated_at = NOW()
		WHERE ads_shown_today_reset_at < CURRENT_DATE
	`

	result, err := r.db.DB.Exec(query)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	fmt.Printf("Reset daily counters for %d profiles\n", rows)
	return nil
}

// ResetHourlyCounters resetea los contadores horarios (llamado por cron job)
func (r *UserAdProfileRepository) ResetHourlyCounters() error {
	query := `
		UPDATE user_ad_profiles
		SET
			ads_shown_this_hour = 0,
			ads_shown_this_hour_reset_at = date_trunc('hour', NOW()),
			updated_at = NOW()
		WHERE ads_shown_this_hour_reset_at < date_trunc('hour', NOW())
	`

	result, err := r.db.DB.Exec(query)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	fmt.Printf("Reset hourly counters for %d profiles\n", rows)
	return nil
}

// scanProfile escanea una fila de la BD a un UserAdProfile
func (r *UserAdProfileRepository) scanProfile(row *sql.Row) (*domain.UserAdProfile, error) {
	var profile domain.UserAdProfile
	var premiumUntil sql.NullTime
	var lastAdShownAt sql.NullTime
	var listeningTimes sql.NullString
	var stripeCustomerID sql.NullString
	var stripeSubscriptionID sql.NullString

	err := row.Scan(
		&profile.ID,
		&profile.UserID,
		&profile.IsPremium,
		&premiumUntil,
		&profile.AdsShownToday,
		&profile.AdsShownThisHour,
		&lastAdShownAt,
		&profile.TotalAdsShown,
		&profile.TotalAdClicks,
		pq.Array(&profile.PreferredGenres),
		&listeningTimes,
		&stripeCustomerID,
		&stripeSubscriptionID,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("profile not found: %w", err)
	}

	// Convertir nullables
	if premiumUntil.Valid {
		profile.PremiumExpiresAt = &premiumUntil.Time
	}
	if lastAdShownAt.Valid {
		profile.LastAdShownAt = &lastAdShownAt.Time
	}
	if listeningTimes.Valid {
		profile.ListeningTimes = &listeningTimes.String
	}
	if stripeCustomerID.Valid {
		profile.StripeCustomerID = &stripeCustomerID.String
	}
	if stripeSubscriptionID.Valid {
		profile.StripeSubscriptionID = &stripeSubscriptionID.String
	}

	return &profile, nil
}
