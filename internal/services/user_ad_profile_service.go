package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/cache"

	"github.com/google/uuid"
)

// UserAdProfileService handles user ad profile management
type UserAdProfileService struct {
	profileRepo domain.UserAdProfileRepository
	adCache     *cache.AdCache
	logger      *slog.Logger
}

// NewUserAdProfileService creates a new user ad profile service
func NewUserAdProfileService(
	profileRepo domain.UserAdProfileRepository,
	adCache *cache.AdCache,
	logger *slog.Logger,
) *UserAdProfileService {
	return &UserAdProfileService{
		profileRepo: profileRepo,
		adCache:     adCache,
		logger:      logger,
	}
}

// GetOrCreateProfile retrieves an existing user ad profile or creates a new one (idempotent)
func (s *UserAdProfileService) GetOrCreateProfile(ctx context.Context, userID uuid.UUID) (*domain.UserAdProfile, error) {
	s.logger.Info("Getting or creating ad profile", "user_id", userID)

	profile, err := s.profileRepo.GetOrCreate(userID)
	if err != nil {
		s.logger.Error("Failed to get or create profile", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get or create profile: %w", err)
	}

	s.logger.Info("Profile retrieved/created", "user_id", userID, "profile_id", profile.ID)
	return profile, nil
}

// GetProfile retrieves a user ad profile by user ID
func (s *UserAdProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.UserAdProfile, error) {
	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil {
		s.logger.Error("Failed to get profile", "error", err, "user_id", userID)
		return nil, err
	}

	return profile, nil
}

// UpdatePremiumStatus updates the premium status of a user
func (s *UserAdProfileService) UpdatePremiumStatus(ctx context.Context, userID uuid.UUID, isPremium bool, expiresAt *time.Time) error {
	s.logger.Info("Updating premium status", "user_id", userID, "is_premium", isPremium, "expires_at", expiresAt)

	if err := s.profileRepo.UpdatePremiumStatus(userID, isPremium, expiresAt); err != nil {
		s.logger.Error("Failed to update premium status", "error", err, "user_id", userID)
		return fmt.Errorf("failed to update premium status: %w", err)
	}

	s.logger.Info("Premium status updated successfully", "user_id", userID)
	return nil
}

// CanShowAd checks if an ad can be shown to a user (frequency capping check)
func (s *UserAdProfileService) CanShowAd(ctx context.Context, userID uuid.UUID) (bool, error) {
	// Get user profile
	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil {
		// If profile doesn't exist, user can see ads
		s.logger.Warn("Profile not found, allowing ad", "user_id", userID)
		return true, nil
	}

	// Check if user has premium
	if profile.HasPremium() {
		s.logger.Info("User has premium, no ads", "user_id", userID)
		return false, nil
	}

	// Get frequency caps from cache (real-time counters)
	hourlyCount, err := s.adCache.GetUserAdCountHourly(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get hourly ad count from cache", "error", err)
		hourlyCount = int64(profile.AdsShownThisHour) // Fallback to DB
	}

	dailyCount, err := s.adCache.GetUserAdCountDaily(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get daily ad count from cache", "error", err)
		dailyCount = int64(profile.AdsShownToday) // Fallback to DB
	}

	caps := domain.DefaultFrequencyCaps()

	// Check frequency caps
	if hourlyCount >= int64(caps.MaxAdsPerHour) {
		s.logger.Info("User reached hourly ad cap", "user_id", userID, "count", hourlyCount, "cap", caps.MaxAdsPerHour)
		return false, nil
	}

	if dailyCount >= int64(caps.MaxAdsPerDay) {
		s.logger.Info("User reached daily ad cap", "user_id", userID, "count", dailyCount, "cap", caps.MaxAdsPerDay)
		return false, nil
	}

	return true, nil
}

// IncrementAdsShown increments the ads shown counters for a user
func (s *UserAdProfileService) IncrementAdsShown(ctx context.Context, userID uuid.UUID) error {
	s.logger.Info("Incrementing ads shown", "user_id", userID)

	// Increment in database
	if err := s.profileRepo.IncrementAdsShown(userID); err != nil {
		s.logger.Error("Failed to increment ads shown", "error", err, "user_id", userID)
		return fmt.Errorf("failed to increment ads shown: %w", err)
	}

	// Increment in cache asynchronously
	go func() {
		if _, err := s.adCache.IncrementUserAdCountHourly(ctx, userID); err != nil {
			s.logger.Error("Failed to increment hourly ad count in cache", "error", err, "user_id", userID)
		}

		if _, err := s.adCache.IncrementUserAdCountDaily(ctx, userID); err != nil {
			s.logger.Error("Failed to increment daily ad count in cache", "error", err, "user_id", userID)
		}
	}()

	return nil
}

// IncrementAdClicks increments the ad click counter for a user
func (s *UserAdProfileService) IncrementAdClicks(ctx context.Context, userID uuid.UUID) error {
	s.logger.Info("Incrementing ad clicks", "user_id", userID)

	if err := s.profileRepo.IncrementAdClicks(userID); err != nil {
		s.logger.Error("Failed to increment ad clicks", "error", err, "user_id", userID)
		return fmt.Errorf("failed to increment ad clicks: %w", err)
	}

	return nil
}

// ResetDailyCounters resets daily ad counters for all users (cron job)
func (s *UserAdProfileService) ResetDailyCounters(ctx context.Context) (int64, error) {
	s.logger.Info("Resetting daily ad counters")

	if err := s.profileRepo.ResetDailyCounters(); err != nil {
		s.logger.Error("Failed to reset daily counters", "error", err)
		return 0, fmt.Errorf("failed to reset daily counters: %w", err)
	}

	// Note: Redis cache keys will expire automatically (24h TTL)
	s.logger.Info("Daily ad counters reset successfully")
	return 0, nil // Repository doesn't return count, so return 0
}

// ResetHourlyCounters resets hourly ad counters for all users (cron job)
func (s *UserAdProfileService) ResetHourlyCounters(ctx context.Context) (int64, error) {
	s.logger.Info("Resetting hourly ad counters")

	if err := s.profileRepo.ResetHourlyCounters(); err != nil {
		s.logger.Error("Failed to reset hourly counters", "error", err)
		return 0, fmt.Errorf("failed to reset hourly counters: %w", err)
	}

	// Note: Redis cache keys will expire automatically (1h TTL)
	s.logger.Info("Hourly ad counters reset successfully")
	return 0, nil // Repository doesn't return count, so return 0
}

// UserAdStats represents user ad statistics
type UserAdStats struct {
	UserID           uuid.UUID  `json:"user_id"`
	IsPremium        bool       `json:"is_premium"`
	TotalAdsShown    int        `json:"total_ads_shown"`
	TotalAdClicks    int        `json:"total_ad_clicks"`
	CTR              float64    `json:"ctr"`
	AdsShownToday    int        `json:"ads_shown_today"`
	AdsShownThisHour int        `json:"ads_shown_this_hour"`
	LastAdShownAt    *time.Time `json:"last_ad_shown_at,omitempty"`
	IsActive         bool       `json:"is_active"`
}

// GetProfileStats retrieves user ad statistics
func (s *UserAdProfileService) GetProfileStats(ctx context.Context, userID uuid.UUID) (*UserAdStats, error) {
	s.logger.Info("Getting profile stats", "user_id", userID)

	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil {
		s.logger.Error("Failed to get profile for stats", "error", err, "user_id", userID)
		return nil, err
	}

	stats := &UserAdStats{
		UserID:           userID,
		IsPremium:        profile.HasPremium(),
		TotalAdsShown:    profile.TotalAdsShown,
		TotalAdClicks:    profile.TotalAdClicks,
		CTR:              profile.CTR(),
		AdsShownToday:    profile.AdsShownToday,
		AdsShownThisHour: profile.AdsShownThisHour,
		LastAdShownAt:    profile.LastAdShownAt,
		IsActive:         profile.IsActive(),
	}

	return stats, nil
}

// UpdateProfile updates a user ad profile
func (s *UserAdProfileService) UpdateProfile(ctx context.Context, profile *domain.UserAdProfile) error {
	s.logger.Info("Updating profile", "user_id", profile.UserID, "profile_id", profile.ID)

	if err := s.profileRepo.Update(profile); err != nil {
		s.logger.Error("Failed to update profile", "error", err, "user_id", profile.UserID)
		return fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("Profile updated successfully", "user_id", profile.UserID)
	return nil
}

// PremiumSubscriptionParams holds parameters for premium subscription
type PremiumSubscriptionParams struct {
	StripeCustomerID     string
	StripeSubscriptionID string
	SubscriptionStatus   string
	ExpiresAt            *time.Time
}

// ActivatePremium activates premium subscription for a user (Stripe integration)
func (s *UserAdProfileService) ActivatePremium(ctx context.Context, userID uuid.UUID, params PremiumSubscriptionParams) error {
	s.logger.Info("Activating premium subscription", "user_id", userID, "stripe_customer", params.StripeCustomerID)

	// Get or create profile
	profile, err := s.profileRepo.GetOrCreate(userID)
	if err != nil {
		s.logger.Error("Failed to get profile for premium activation", "error", err, "user_id", userID)
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Update premium status
	profile.IsPremium = true
	profile.PremiumExpiresAt = params.ExpiresAt
	profile.StripeCustomerID = &params.StripeCustomerID
	profile.StripeSubscriptionID = &params.StripeSubscriptionID
	profile.SubscriptionStatus = &params.SubscriptionStatus
	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(profile); err != nil {
		s.logger.Error("Failed to update profile for premium activation", "error", err, "user_id", userID)
		return fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("Premium subscription activated", "user_id", userID)
	return nil
}

// DeactivatePremium deactivates premium subscription for a user
func (s *UserAdProfileService) DeactivatePremium(ctx context.Context, userID uuid.UUID) error {
	s.logger.Info("Deactivating premium subscription", "user_id", userID)

	if err := s.profileRepo.UpdatePremiumStatus(userID, false, nil); err != nil {
		s.logger.Error("Failed to deactivate premium", "error", err, "user_id", userID)
		return fmt.Errorf("failed to deactivate premium: %w", err)
	}

	s.logger.Info("Premium subscription deactivated", "user_id", userID)
	return nil
}

// CheckFrequencyCap checks if a user has exceeded frequency caps (detailed check)
func (s *UserAdProfileService) CheckFrequencyCap(ctx context.Context, userID uuid.UUID) (bool, string, error) {
	profile, err := s.profileRepo.GetByUserID(userID)
	if err != nil {
		// If profile doesn't exist, allow ads
		return false, "", nil
	}

	if profile.HasPremium() {
		return true, "user_has_premium", nil
	}

	hourlyCount, _ := s.adCache.GetUserAdCountHourly(ctx, userID)
	dailyCount, _ := s.adCache.GetUserAdCountDaily(ctx, userID)

	caps := domain.DefaultFrequencyCaps()

	if hourlyCount >= int64(caps.MaxAdsPerHour) {
		return true, "hourly_cap_exceeded", nil
	}

	if dailyCount >= int64(caps.MaxAdsPerDay) {
		return true, "daily_cap_exceeded", nil
	}

	return false, "", nil
}
