package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAdvertisement_Validate(t *testing.T) {
	cpmRate := 5000
	width := 300
	height := 250
	duration := 30

	tests := []struct {
		name    string
		ad      *Advertisement
		wantErr error
	}{
		{
			name: "valid banner ad with CPM",
			ad: &Advertisement{
				Title:            "Test Ad",
				AdFormat:         AdFormatBanner,
				AdType:           AdTypeImage,
				MediaURL:         "https://example.com/banner.jpg",
				ClickURL:         "https://example.com/click",
				Width:            &width,
				Height:           &height,
				CPMRateCents:     &cpmRate,
				StartDate:        time.Now(),
				EndDate:          time.Now().Add(24 * time.Hour),
				TotalBudgetCents: 100000,
			},
			wantErr: nil,
		},
		{
			name: "valid audio ad",
			ad: &Advertisement{
				Title:            "Audio Ad",
				AdFormat:         AdFormatAudio,
				AdType:           AdTypeAudio,
				MediaURL:         "https://example.com/audio.mp3",
				ClickURL:         "https://example.com/click",
				DurationSeconds:  &duration,
				CPMRateCents:     &cpmRate,
				StartDate:        time.Now(),
				EndDate:          time.Now().Add(24 * time.Hour),
				TotalBudgetCents: 100000,
			},
			wantErr: nil,
		},
		{
			name: "missing title",
			ad: &Advertisement{
				Title:        "",
				AdFormat:     AdFormatBanner,
				AdType:       AdTypeImage,
				MediaURL:     "https://example.com/banner.jpg",
				ClickURL:     "https://example.com/click",
				CPMRateCents: &cpmRate,
				StartDate:    time.Now(),
				EndDate:      time.Now().Add(24 * time.Hour),
			},
			wantErr: ErrInvalidAdTitle,
		},
		{
			name: "banner without dimensions",
			ad: &Advertisement{
				Title:        "Test",
				AdFormat:     AdFormatBanner,
				AdType:       AdTypeImage,
				MediaURL:     "https://example.com/banner.jpg",
				ClickURL:     "https://example.com/click",
				CPMRateCents: &cpmRate,
				StartDate:    time.Now(),
				EndDate:      time.Now().Add(24 * time.Hour),
			},
			wantErr: ErrInvalidDimensions,
		},
		{
			name: "audio without duration",
			ad: &Advertisement{
				Title:        "Test",
				AdFormat:     AdFormatAudio,
				AdType:       AdTypeAudio,
				MediaURL:     "https://example.com/audio.mp3",
				ClickURL:     "https://example.com/click",
				CPMRateCents: &cpmRate,
				StartDate:    time.Now(),
				EndDate:      time.Now().Add(24 * time.Hour),
			},
			wantErr: ErrInvalidDuration,
		},
		{
			name: "no pricing model",
			ad: &Advertisement{
				Title:     "Test",
				AdFormat:  AdFormatBanner,
				AdType:    AdTypeImage,
				MediaURL:  "https://example.com/banner.jpg",
				ClickURL:  "https://example.com/click",
				Width:     &width,
				Height:    &height,
				StartDate: time.Now(),
				EndDate:   time.Now().Add(24 * time.Hour),
			},
			wantErr: ErrInvalidPricingModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ad.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestAdvertisement_CTR(t *testing.T) {
	ad := &Advertisement{
		ImpressionsCount: 1000,
		ClicksCount:      50,
	}
	assert.Equal(t, 5.0, ad.CTR())

	ad.ClicksCount = 0
	assert.Equal(t, 0.0, ad.CTR())

	ad.ImpressionsCount = 0
	assert.Equal(t, 0.0, ad.CTR())
}

func TestAdvertisement_CalculateCost(t *testing.T) {
	cpmRate := 5000 // $50 per 1000 impressions
	cpcRate := 250  // $2.50 per click

	ad := &Advertisement{
		CPMRateCents: &cpmRate,
		CPCRateCents: &cpcRate,
	}

	// Costo por impresión (CPM / 1000)
	assert.Equal(t, 5, ad.CalculateCost(false))

	// Costo por click
	assert.Equal(t, 250, ad.CalculateCost(true))
}

func TestAdvertisement_MatchesTargeting(t *testing.T) {
	ad := &Advertisement{
		TargetCountries: []string{"US", "CA", "MX"},
		TargetGenres:    []string{"rock", "pop"},
		TargetDevices:   []string{"mobile", "tablet"},
	}

	// Matches all criteria
	assert.True(t, ad.MatchesTargeting("US", "rock", "en", "mobile"))

	// Doesn't match country
	assert.False(t, ad.MatchesTargeting("BR", "rock", "en", "mobile"))

	// Doesn't match genre
	assert.False(t, ad.MatchesTargeting("US", "jazz", "en", "mobile"))

	// Doesn't match device
	assert.False(t, ad.MatchesTargeting("US", "rock", "en", "desktop"))

	// Empty targeting matches all
	adNoTargeting := &Advertisement{}
	assert.True(t, adNoTargeting.MatchesTargeting("BR", "jazz", "pt", "desktop"))
}

func TestAdvertisement_IsActive(t *testing.T) {
	now := time.Now()

	ad := &Advertisement{
		Status:    AdStatusActive,
		StartDate: now.Add(-1 * time.Hour),
		EndDate:   now.Add(1 * time.Hour),
	}
	assert.True(t, ad.IsActive())

	ad.Status = AdStatusPaused
	assert.False(t, ad.IsActive())

	ad.Status = AdStatusActive
	ad.StartDate = now.Add(1 * time.Hour)
	assert.False(t, ad.IsActive())
}

func TestAdFormat_IsValid(t *testing.T) {
	assert.True(t, AdFormatBanner.IsValid())
	assert.True(t, AdFormatInterstitial.IsValid())
	assert.True(t, AdFormatAudio.IsValid())
	assert.True(t, AdFormatNative.IsValid())
	assert.False(t, AdFormat("invalid").IsValid())
}

func TestAdType_IsValid(t *testing.T) {
	assert.True(t, AdTypeImage.IsValid())
	assert.True(t, AdTypeVideo.IsValid())
	assert.True(t, AdTypeAudio.IsValid())
	assert.True(t, AdTypeHTML.IsValid())
	assert.False(t, AdType("invalid").IsValid())
}
