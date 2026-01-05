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

// ClickService handles click tracking, fraud detection, and conversion attribution
type ClickService struct {
	clickRepo      domain.AdClickRepository
	impressionRepo domain.AdImpressionRepository
	adRepo         domain.AdvertisementRepository
	adCache        *cache.AdCache
	logger         *slog.Logger
}

// NewClickService creates a new click service
func NewClickService(
	clickRepo domain.AdClickRepository,
	impressionRepo domain.AdImpressionRepository,
	adRepo domain.AdvertisementRepository,
	adCache *cache.AdCache,
	logger *slog.Logger,
) *ClickService {
	return &ClickService{
		clickRepo:      clickRepo,
		impressionRepo: impressionRepo,
		adRepo:         adRepo,
		adCache:        adCache,
		logger:         logger,
	}
}

// RecordClick records an ad click with impression validation and fraud detection
func (s *ClickService) RecordClick(ctx context.Context, click *domain.AdClick) error {
	s.logger.Info("Recording click", "impression_id", click.ImpressionID, "ad_id", click.AdvertisementID)

	// 1. Validate click data
	if err := click.Validate(); err != nil {
		s.logger.Warn("Invalid click data", "error", err)
		return err
	}

	// 2. Verify impression exists and is recent (within 24 hours)
	impression, err := s.impressionRepo.GetByID(click.ImpressionID)
	if err != nil {
		s.logger.Error("Failed to fetch impression for click", "error", err, "impression_id", click.ImpressionID)
		return fmt.Errorf("impression not found: %w", err)
	}

	// Check impression age (must be < 24 hours)
	if time.Since(impression.CreatedAt) > 24*time.Hour {
		s.logger.Warn("Click on expired impression", "impression_id", click.ImpressionID, "age", time.Since(impression.CreatedAt))
		return domain.ErrInvalidClickData
	}

	// Verify impression and click match on ad_id and user_id
	if impression.AdvertisementID != click.AdvertisementID {
		s.logger.Warn("Click ad_id mismatch", "click_ad", click.AdvertisementID, "impression_ad", impression.AdvertisementID)
		return domain.ErrInvalidClickData
	}

	if impression.UserID != nil && click.UserID != nil && *impression.UserID != *click.UserID {
		s.logger.Warn("Click user_id mismatch", "click_user", *click.UserID, "impression_user", *impression.UserID)
		return domain.ErrInvalidClickData
	}

	// 3. Verify advertisement exists and is active
	ad, err := s.adRepo.GetByID(click.AdvertisementID)
	if err != nil {
		s.logger.Error("Failed to fetch advertisement for click", "error", err, "ad_id", click.AdvertisementID)
		return err
	}

	if ad.Status != domain.AdStatusActive {
		s.logger.Warn("Click on inactive advertisement", "ad_id", click.AdvertisementID, "status", ad.Status)
		return domain.ErrAdvertisementNotActive
	}

	// 4. Check for suspicious click patterns using domain method
	if click.IsSuspicious(impression) {
		s.logger.Warn("Suspicious click detected", "impression_id", click.ImpressionID)
		return domain.ErrSuspiciousActivity
	}

	// 5. Check IP-based fraud patterns
	ipClicks, err := s.adCache.CountIPClicks(ctx, click.IPAddress, 5*time.Minute)
	if err != nil {
		s.logger.Warn("Failed to get IP click count", "error", err)
	} else if ipClicks > 10 {
		s.logger.Warn("Excessive clicks from IP", "ip", click.IPAddress, "count", ipClicks)
		return domain.ErrSuspiciousActivity
	}

	// 6. Generate ID if not present
	if click.ID == uuid.Nil {
		click.ID = uuid.New()
	}

	// 7. Save click
	if err := s.clickRepo.Create(click); err != nil {
		s.logger.Error("Failed to save click", "error", err)
		return fmt.Errorf("failed to save click: %w", err)
	}

	s.logger.Info("Click recorded successfully", "click_id", click.ID, "ad_id", click.AdvertisementID)

	// 8. Update counters asynchronously
	go s.updateCounters(ctx, click)

	return nil
}

// updateCounters updates various counters after recording a click (async)
func (s *ClickService) updateCounters(ctx context.Context, click *domain.AdClick) {
	// Increment advertisement click count in cache
	if err := s.adCache.IncrementClicks(ctx, click.AdvertisementID); err != nil {
		s.logger.Error("Failed to increment ad click count in cache", "error", err, "ad_id", click.AdvertisementID)
	}

	// Track IP click in cache (5-minute window for fraud detection)
	if err := s.adCache.TrackIPClick(ctx, click.IPAddress, click.ID, 5*time.Minute); err != nil {
		s.logger.Error("Failed to track IP click", "error", err, "ip", click.IPAddress)
	}
}

// RecordConversion records a conversion for a click (for attribution tracking)
func (s *ClickService) RecordConversion(ctx context.Context, clickID uuid.UUID, conversionValueCents int) error {
	s.logger.Info("Recording conversion", "click_id", clickID, "value_cents", conversionValueCents)

	// Update click with conversion data using repository method
	if err := s.clickRepo.UpdateConversion(clickID, conversionValueCents); err != nil {
		s.logger.Error("Failed to update click with conversion", "error", err, "click_id", clickID)
		return fmt.Errorf("failed to record conversion: %w", err)
	}

	s.logger.Info("Conversion recorded successfully", "click_id", clickID, "value_cents", conversionValueCents)
	return nil
}

// GetClickByImpression retrieves the click associated with an impression
func (s *ClickService) GetClickByImpression(ctx context.Context, impressionID uuid.UUID) (*domain.AdClick, error) {
	click, err := s.clickRepo.GetByImpressionID(impressionID)
	if err != nil {
		s.logger.Error("Failed to fetch click by impression", "error", err, "impression_id", impressionID)
		return nil, err
	}

	return click, nil
}

// GetClicksByAdvertisement retrieves clicks for a specific advertisement
func (s *ClickService) GetClicksByAdvertisement(ctx context.Context, adID uuid.UUID, limit int) ([]*domain.AdClick, error) {
	clicks, err := s.clickRepo.GetByAdvertisementID(adID, limit)
	if err != nil {
		s.logger.Error("Failed to fetch clicks by advertisement", "error", err, "ad_id", adID)
		return nil, err
	}

	return clicks, nil
}

// GetRecentClicksByIP retrieves recent clicks from an IP address
func (s *ClickService) GetRecentClicksByIP(ctx context.Context, ipAddress string, since time.Time) ([]*domain.AdClick, error) {
	clicks, err := s.clickRepo.GetRecentByIPAddress(ipAddress, since)
	if err != nil {
		s.logger.Error("Failed to fetch recent clicks by IP", "error", err, "ip", ipAddress)
		return nil, err
	}

	return clicks, nil
}

// ClickStats represents click statistics for an advertisement
type ClickStats struct {
	AdvertisementID      uuid.UUID `json:"advertisement_id"`
	TotalClicks          int64     `json:"total_clicks"`
	TotalConversionValue int       `json:"total_conversion_value_cents"`
	AverageClickDelay    float64   `json:"average_click_delay_seconds"`
	ConversionRate       float64   `json:"conversion_rate"`
	Since                time.Time `json:"since"`
}

// GetClickStats retrieves click statistics for an advertisement
func (s *ClickService) GetClickStats(ctx context.Context, adID uuid.UUID, since time.Time) (*ClickStats, error) {
	s.logger.Info("Getting click stats", "ad_id", adID, "since", since)

	// Get click count
	clickCount, err := s.clickRepo.CountByAdvertisementID(adID, since)
	if err != nil {
		s.logger.Error("Failed to count clicks", "error", err, "ad_id", adID)
		return nil, err
	}

	// Get recent clicks for detailed stats
	clicks, err := s.clickRepo.GetByAdvertisementID(adID, 1000) // Limit to last 1000 clicks
	if err != nil {
		s.logger.Error("Failed to fetch clicks for stats", "error", err, "ad_id", adID)
		return nil, err
	}

	stats := &ClickStats{
		AdvertisementID: adID,
		TotalClicks:     clickCount,
		Since:           since,
	}

	if len(clicks) == 0 {
		return stats, nil
	}

	// Calculate statistics
	var totalDelay time.Duration
	var conversionValue int
	var conversionsCount int64

	for _, click := range clicks {
		if click.Converted && click.ConversionValueCents != nil {
			conversionsCount++
			conversionValue += *click.ConversionValueCents
		}

		// Get corresponding impression for delay calculation
		impression, err := s.impressionRepo.GetByID(click.ImpressionID)
		if err == nil {
			delay := click.TimeToClick(impression)
			totalDelay += delay
		}
	}

	stats.TotalConversionValue = conversionValue

	if len(clicks) > 0 {
		stats.AverageClickDelay = totalDelay.Seconds() / float64(len(clicks))
	}

	if clickCount > 0 {
		stats.ConversionRate = (float64(conversionsCount) / float64(clickCount)) * 100.0
	}

	s.logger.Info("Click stats calculated",
		"ad_id", adID,
		"total_clicks", stats.TotalClicks,
		"conversion_rate", stats.ConversionRate,
	)

	return stats, nil
}
