package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ImpressionToken gestiona tokens HMAC para validar impresiones
type ImpressionToken struct {
	AdvertisementID uuid.UUID
	SessionID       string
	Timestamp       int64
	Signature       string
}

// GenerateImpressionToken genera un token HMAC para una impresión
func GenerateImpressionToken(adID uuid.UUID, sessionID string, secret []byte) string {
	timestamp := time.Now().Unix()
	message := fmt.Sprintf("%s:%s:%d", adID.String(), sessionID, timestamp)
	
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	
	// Format: adID:sessionID:timestamp:signature
	return fmt.Sprintf("%s:%s:%d:%s", adID.String(), sessionID, timestamp, signature)
}

// ValidateImpressionToken valida un token HMAC
func ValidateImpressionToken(token string, secret []byte, maxAge time.Duration) (*ImpressionToken, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 4 {
		return nil, ErrInvalidImpressionToken
	}
	
	adID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, ErrInvalidImpressionToken
	}
	
	sessionID := parts[1]
	
	timestamp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, ErrInvalidImpressionToken
	}
	
	signature := parts[3]
	
	// Verificar que no haya expirado
	tokenTime := time.Unix(timestamp, 0)
	if time.Since(tokenTime) > maxAge {
		return nil, ErrTokenExpired
	}
	
	// Verificar firma HMAC
	message := fmt.Sprintf("%s:%s:%d", adID.String(), sessionID, timestamp)
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, ErrInvalidToken
	}
	
	return &ImpressionToken{
		AdvertisementID: adID,
		SessionID:       sessionID,
		Timestamp:       timestamp,
		Signature:       signature,
	}, nil
}

// FraudDetectionSignals representa señales para detección de fraude
type FraudDetectionSignals struct {
	IPAddress            string
	UserAgent            string
	SessionID            string
	TimeToClick          time.Duration
	ImpressionsFromIP    int64
	ClicksFromIP         int64
	ViewableDuration     int
	DeviceType           string
	IsViewable           bool
	HasValidReferrer     bool
}

// CalculateFraudScore calcula un score de fraude (0-1, donde 1 es muy sospechoso)
func (s *FraudDetectionSignals) CalculateFraudScore() float64 {
	score := 0.0
	
	// Click muy rápido (< 100ms)
	if s.TimeToClick > 0 && s.TimeToClick < 100*time.Millisecond {
		score += 0.3
	}
	
	// Demasiados clicks de la misma IP (> 10 en 5 minutos)
	if s.ClicksFromIP > 10 {
		score += 0.25
	}
	
	// Demasiadas impresiones de la misma IP (> 50 en 5 minutos)
	if s.ImpressionsFromIP > 50 {
		score += 0.2
	}
	
	// No viewable o duración muy corta
	if !s.IsViewable || s.ViewableDuration < 500 {
		score += 0.15
	}
	
	// User agent sospechoso o vacío
	if s.UserAgent == "" || len(s.UserAgent) < 20 {
		score += 0.1
	}
	
	// Limitar a 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// IsSuspicious determina si las señales son sospechosas (score > 0.7)
func (s *FraudDetectionSignals) IsSuspicious() bool {
	return s.CalculateFraudScore() > 0.7
}

// AdSecurityService define servicios de seguridad para publicidad
type AdSecurityService interface {
	GenerateToken(adID uuid.UUID, sessionID string) (string, error)
	ValidateToken(token string) (*ImpressionToken, error)
	CheckFraud(signals *FraudDetectionSignals) (fraudScore float64, isSuspicious bool, err error)
	RateLimitCheck(key string, limit int, window time.Duration) (allowed bool, remaining int, err error)
}
