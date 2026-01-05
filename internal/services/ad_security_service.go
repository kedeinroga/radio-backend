package services

import (
	"fmt"
	"strings"
	"time"

	"radio-backend/internal/domain"

	"github.com/google/uuid"
)

// AdSecurityServiceImpl implementa domain.AdSecurityService
type AdSecurityServiceImpl struct {
	hmacSecret          []byte
	fraudScoreThreshold float64
}

// NewAdSecurityService crea una nueva instancia del servicio de seguridad
func NewAdSecurityService(hmacSecret []byte, fraudScoreThreshold float64) *AdSecurityServiceImpl {
	return &AdSecurityServiceImpl{
		hmacSecret:          hmacSecret,
		fraudScoreThreshold: fraudScoreThreshold,
	}
}

// GenerateToken genera un token HMAC para una impresión
func (s *AdSecurityServiceImpl) GenerateToken(adID uuid.UUID, sessionID string) (string, error) {
	if adID == uuid.Nil {
		return "", fmt.Errorf("invalid ad ID")
	}
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}

	token := domain.GenerateImpressionToken(adID, sessionID, s.hmacSecret)
	return token, nil
}

// ValidateToken valida un token HMAC
func (s *AdSecurityServiceImpl) ValidateToken(token string) (*domain.ImpressionToken, error) {
	if token == "" {
		return nil, domain.ErrInvalidImpressionToken
	}

	// Tokens válidos por 24 horas
	maxAge := 24 * time.Hour

	impressionToken, err := domain.ValidateImpressionToken(token, s.hmacSecret, maxAge)
	if err != nil {
		return nil, err
	}

	return impressionToken, nil
}

// CheckFraud analiza señales de fraude y retorna un score
func (s *AdSecurityServiceImpl) CheckFraud(signals *domain.FraudDetectionSignals) (fraudScore float64, isSuspicious bool, err error) {
	if signals == nil {
		return 0, false, fmt.Errorf("fraud signals are required")
	}

	score := 0.0

	// 1. Bot Detection (30 puntos)
	if s.isBotUserAgent(signals.UserAgent) {
		score += 30.0
	}

	// 2. Missing User Agent (20 puntos)
	if signals.UserAgent == "" {
		score += 20.0
	}

	// 3. IP Impressions Count (40 puntos si > 100, 20 si > 50)
	if signals.ImpressionsFromIP > 100 {
		score += 40.0
	} else if signals.ImpressionsFromIP > 50 {
		score += 20.0
	}

	// 4. IP Clicks Count (30 puntos si > 20)
	if signals.ClicksFromIP > 20 {
		score += 30.0
	}

	// 5. Suspicious Timing (30 puntos si < 100ms)
	if signals.TimeToClick > 0 && signals.TimeToClick < 100*time.Millisecond {
		score += 30.0
	}

	// 6. Invalid Referrer (10 puntos)
	if !signals.HasValidReferrer {
		score += 10.0
	}

	// 7. Not Viewable (15 puntos)
	if !signals.IsViewable {
		score += 15.0
	}

	// Score normalizado (0-100 convertido a 0-1)
	fraudScore = score / 100.0

	// Es sospechoso si el score >= threshold
	isSuspicious = fraudScore >= s.fraudScoreThreshold

	return fraudScore, isSuspicious, nil
}

// RateLimitCheck verifica límites de tasa (implementación básica)
func (s *AdSecurityServiceImpl) RateLimitCheck(key string, limit int, window time.Duration) (allowed bool, remaining int, err error) {
	// Esta es una implementación simplificada
	// En producción, deberías usar Redis con sliding window

	// Por ahora, siempre permitimos (el middleware maneja el rate limiting real)
	return true, limit, nil
}

// isBotUserAgent detecta user agents de bots conocidos
func (s *AdSecurityServiceImpl) isBotUserAgent(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	userAgentLower := strings.ToLower(userAgent)

	botPatterns := []string{
		"bot", "crawler", "spider", "scraper",
		"curl", "wget", "python", "go-http-client",
		"headless", "phantom", "selenium", "puppeteer",
		"scrapy", "mechanize", "httpclient",
	}

	for _, pattern := range botPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}

	return false
}

// Verify interface implementation
var _ domain.AdSecurityService = (*AdSecurityServiceImpl)(nil)
