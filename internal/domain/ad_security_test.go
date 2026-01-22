package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateImpressionToken(t *testing.T) {
	adID := uuid.New()
	sessionID := "session123"
	secret := []byte("test-secret-key-32-bytes-long!!")

	token := GenerateImpressionToken(adID, sessionID, secret)
	assert.NotEmpty(t, token)

	// Token should have 4 parts separated by ':'
	assert.Contains(t, token, ":")
}

func TestValidateImpressionToken(t *testing.T) {
	adID := uuid.New()
	sessionID := "session123"
	secret := []byte("test-secret-key-32-bytes-long!!")

	// Generate valid token
	token := GenerateImpressionToken(adID, sessionID, secret)

	// Validate immediately (should pass)
	validatedToken, err := ValidateImpressionToken(token, secret, 5*time.Minute)
	assert.NoError(t, err)
	assert.NotNil(t, validatedToken)
	assert.Equal(t, adID, validatedToken.AdvertisementID)
	assert.Equal(t, sessionID, validatedToken.SessionID)

	// Invalid token format
	_, err = ValidateImpressionToken("invalid-token", secret, 5*time.Minute)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidImpressionToken, err)

	// Wrong secret
	wrongSecret := []byte("wrong-secret-key")
	_, err = ValidateImpressionToken(token, wrongSecret, 5*time.Minute)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestFraudDetectionSignals_CalculateFraudScore(t *testing.T) {
	tests := []struct {
		name     string
		signals  *FraudDetectionSignals
		minScore float64
		maxScore float64
	}{
		{
			name: "legitimate traffic",
			signals: &FraudDetectionSignals{
				IPAddress:         "192.168.1.1",
				UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				TimeToClick:       2 * time.Second,
				ImpressionsFromIP: 5,
				ClicksFromIP:      1,
				ViewableDuration:  2000,
				IsViewable:        true,
			},
			minScore: 0.0,
			maxScore: 0.2,
		},
		{
			name: "suspicious - fast click",
			signals: &FraudDetectionSignals{
				IPAddress:         "192.168.1.1",
				UserAgent:         "Mozilla/5.0",
				TimeToClick:       50 * time.Millisecond,
				ImpressionsFromIP: 10,
				ClicksFromIP:      5,
				ViewableDuration:  1000,
				IsViewable:        true,
			},
			minScore: 0.3,
			maxScore: 0.5,
		},
		{
			name: "very suspicious - multiple red flags",
			signals: &FraudDetectionSignals{
				IPAddress:         "192.168.1.1",
				UserAgent:         "",
				TimeToClick:       20 * time.Millisecond,
				ImpressionsFromIP: 100,
				ClicksFromIP:      50,
				ViewableDuration:  100,
				IsViewable:        false,
			},
			minScore: 0.8,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.signals.CalculateFraudScore()
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestFraudDetectionSignals_IsSuspicious(t *testing.T) {
	// Low fraud score - not suspicious
	signals := &FraudDetectionSignals{
		TimeToClick:       2 * time.Second,
		ImpressionsFromIP: 5,
		ClicksFromIP:      1,
		ViewableDuration:  2000,
		IsViewable:        true,
		UserAgent:         "Mozilla/5.0 (legitimate)",
	}
	assert.False(t, signals.IsSuspicious())

	// High fraud score - suspicious
	signals = &FraudDetectionSignals{
		TimeToClick:       20 * time.Millisecond,
		ImpressionsFromIP: 100,
		ClicksFromIP:      50,
		ViewableDuration:  100,
		IsViewable:        false,
		UserAgent:         "",
	}
	assert.True(t, signals.IsSuspicious())
}
