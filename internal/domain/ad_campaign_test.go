package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAdCampaign_Validate(t *testing.T) {
	tests := []struct {
		name     string
		campaign *AdCampaign
		wantErr  error
	}{
		{
			name: "valid campaign",
			campaign: &AdCampaign{
				ID:               uuid.New(),
				AdvertiserID:     uuid.New(),
				Name:             "Test Campaign",
				TotalBudgetCents: 100000,
				StartDate:        time.Now(),
				EndDate:          time.Now().Add(24 * time.Hour),
				Status:           CampaignStatusDraft,
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			campaign: &AdCampaign{
				ID:               uuid.New(),
				AdvertiserID:     uuid.New(),
				Name:             "",
				TotalBudgetCents: 100000,
				StartDate:        time.Now(),
				EndDate:          time.Now().Add(24 * time.Hour),
			},
			wantErr: ErrInvalidCampaignName,
		},
		{
			name: "invalid budget",
			campaign: &AdCampaign{
				ID:               uuid.New(),
				AdvertiserID:     uuid.New(),
				Name:             "Test",
				TotalBudgetCents: -100,
				StartDate:        time.Now(),
				EndDate:          time.Now().Add(24 * time.Hour),
			},
			wantErr: ErrInvalidBudget,
		},
		{
			name: "invalid dates",
			campaign: &AdCampaign{
				ID:               uuid.New(),
				AdvertiserID:     uuid.New(),
				Name:             "Test",
				TotalBudgetCents: 100000,
				StartDate:        time.Now().Add(24 * time.Hour),
				EndDate:          time.Now(),
			},
			wantErr: ErrInvalidCampaignDates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.campaign.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestAdCampaign_HasBudget(t *testing.T) {
	campaign := &AdCampaign{
		TotalBudgetCents: 10000,
		SpentCents:       5000,
	}
	assert.True(t, campaign.HasBudget())

	campaign.SpentCents = 10000
	assert.False(t, campaign.HasBudget())

	campaign.SpentCents = 15000
	assert.False(t, campaign.HasBudget())
}

func TestAdCampaign_RemainingBudget(t *testing.T) {
	campaign := &AdCampaign{
		TotalBudgetCents: 10000,
		SpentCents:       3000,
	}
	assert.Equal(t, 7000, campaign.RemainingBudget())

	campaign.SpentCents = 10000
	assert.Equal(t, 0, campaign.RemainingBudget())

	campaign.SpentCents = 12000
	assert.Equal(t, 0, campaign.RemainingBudget())
}

func TestAdCampaign_BudgetUtilization(t *testing.T) {
	campaign := &AdCampaign{
		TotalBudgetCents: 10000,
		SpentCents:       5000,
	}
	assert.Equal(t, 50.0, campaign.BudgetUtilization())

	campaign.SpentCents = 2500
	assert.Equal(t, 25.0, campaign.BudgetUtilization())

	campaign.SpentCents = 10000
	assert.Equal(t, 100.0, campaign.BudgetUtilization())
}

func TestAdCampaign_IsActive(t *testing.T) {
	now := time.Now()

	campaign := &AdCampaign{
		Status:    CampaignStatusActive,
		StartDate: now.Add(-1 * time.Hour),
		EndDate:   now.Add(1 * time.Hour),
	}
	assert.True(t, campaign.IsActive())

	campaign.Status = CampaignStatusPaused
	assert.False(t, campaign.IsActive())

	campaign.Status = CampaignStatusActive
	campaign.StartDate = now.Add(1 * time.Hour)
	assert.False(t, campaign.IsActive())

	campaign.StartDate = now.Add(-1 * time.Hour)
	campaign.EndDate = now.Add(-30 * time.Minute)
	assert.False(t, campaign.IsActive())
}

func TestAdCampaign_ShouldExpire(t *testing.T) {
	now := time.Now()

	campaign := &AdCampaign{
		Status:  CampaignStatusActive,
		EndDate: now.Add(-1 * time.Hour),
	}
	assert.True(t, campaign.ShouldExpire())

	campaign.EndDate = now.Add(1 * time.Hour)
	assert.False(t, campaign.ShouldExpire())

	campaign.Status = CampaignStatusCompleted
	campaign.EndDate = now.Add(-1 * time.Hour)
	assert.False(t, campaign.ShouldExpire())
}
