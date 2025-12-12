package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStation_IsAccessibleBy(t *testing.T) {
	tests := []struct {
		name     string
		station  *Station
		userType UserType
		want     bool
	}{
		{
			name: "premium-only station accessible by premium user",
			station: &Station{
				ID:            "station-1",
				Name:          "Premium Radio",
				IsPremiumOnly: true,
			},
			userType: UserTypePremium,
			want:     true,
		},
		{
			name: "premium-only station not accessible by guest user",
			station: &Station{
				ID:            "station-2",
				Name:          "Premium Radio",
				IsPremiumOnly: true,
			},
			userType: UserTypeGuest,
			want:     false,
		},
		{
			name: "regular station accessible by premium user",
			station: &Station{
				ID:            "station-3",
				Name:          "Free Radio",
				IsPremiumOnly: false,
			},
			userType: UserTypePremium,
			want:     true,
		},
		{
			name: "regular station accessible by guest user",
			station: &Station{
				ID:            "station-4",
				Name:          "Free Radio",
				IsPremiumOnly: false,
			},
			userType: UserTypeGuest,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.station.IsAccessibleBy(tt.userType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStation_NeedsSync(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	tests := []struct {
		name    string
		station *Station
		maxAge  time.Duration
		want    bool
	}{
		{
			name: "station with nil LastSyncedAt needs sync",
			station: &Station{
				ID:           "station-1",
				LastSyncedAt: nil,
			},
			maxAge: 1 * time.Hour,
			want:   true,
		},
		{
			name: "station synced 2 hours ago needs sync (maxAge 1 hour)",
			station: &Station{
				ID:           "station-2",
				LastSyncedAt: &twoHoursAgo,
			},
			maxAge: 1 * time.Hour,
			want:   true,
		},
		{
			name: "station synced 1 hour ago does not need sync (maxAge 2 hours)",
			station: &Station{
				ID:           "station-3",
				LastSyncedAt: &oneHourAgo,
			},
			maxAge: 2 * time.Hour,
			want:   false,
		},
		{
			name: "freshly synced station does not need sync",
			station: &Station{
				ID:           "station-4",
				LastSyncedAt: &now,
			},
			maxAge: 1 * time.Hour,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.station.NeedsSync(tt.maxAge)
			assert.Equal(t, tt.want, got)
		})
	}
}
