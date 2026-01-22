package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AdFraudDetector provides middleware for detecting fraudulent ad requests
type AdFraudDetector struct {
	redis  *redis.Client
	logger *slog.Logger
}

// NewAdFraudDetector creates a new fraud detector
func NewAdFraudDetector(redis *redis.Client, logger *slog.Logger) *AdFraudDetector {
	return &AdFraudDetector{
		redis:  redis,
		logger: logger,
	}
}

// DetectFraud analyzes requests for suspicious patterns
func (f *AdFraudDetector) DetectFraud() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ip := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		// Check for suspicious patterns
		suspiciousScore := 0

		// 1. Check for bot user agents
		if f.isBotUserAgent(userAgent) {
			suspiciousScore += 30
			f.logger.Warn("Bot user agent detected", "ip", ip, "user_agent", userAgent)
		}

		// 2. Check for missing user agent
		if userAgent == "" {
			suspiciousScore += 20
			f.logger.Warn("Missing user agent", "ip", ip)
		}

		// 3. Check for excessive requests from same IP (last 5 minutes)
		requestCount, err := f.getRecentRequestCount(ctx, ip)
		if err == nil {
			if requestCount > 100 {
				suspiciousScore += 40
				f.logger.Warn("Excessive requests from IP", "ip", ip, "count", requestCount)
			} else if requestCount > 50 {
				suspiciousScore += 20
			}
		}

		// 4. Track this request
		if err := f.trackRequest(ctx, ip); err != nil {
			f.logger.Error("Failed to track request", "error", err, "ip", ip)
		}

		// Block if suspicious score is too high
		if suspiciousScore >= 50 {
			f.logger.Warn("Blocking suspicious request", "ip", ip, "score", suspiciousScore)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "suspicious activity detected",
			})
			c.Abort()
			return
		}

		// Add suspicious score to context for services to use
		c.Set("fraud_score", suspiciousScore)
		c.Next()
	}
}

// isBotUserAgent checks if the user agent is a known bot
func (f *AdFraudDetector) isBotUserAgent(userAgent string) bool {
	userAgentLower := strings.ToLower(userAgent)

	botPatterns := []string{
		"bot", "crawler", "spider", "scraper",
		"curl", "wget", "python", "java",
		"headless", "phantom", "selenium",
	}

	for _, pattern := range botPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}

	return false
}

// getRecentRequestCount gets the number of recent requests from an IP
func (f *AdFraudDetector) getRecentRequestCount(ctx context.Context, ip string) (int64, error) {
	key := fmt.Sprintf("fraud:requests:%s", ip)
	count, err := f.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// trackRequest tracks a request from an IP
func (f *AdFraudDetector) trackRequest(ctx context.Context, ip string) error {
	key := fmt.Sprintf("fraud:requests:%s", ip)
	pipe := f.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// CheckIPBlacklist checks if an IP is blacklisted
func (f *AdFraudDetector) CheckIPBlacklist() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ip := c.ClientIP()

		// Check if IP is blacklisted
		key := fmt.Sprintf("fraud:blacklist:ip:%s", ip)
		exists, err := f.redis.Exists(ctx, key).Result()
		if err != nil {
			f.logger.Error("Failed to check IP blacklist", "error", err, "ip", ip)
			c.Next()
			return
		}

		if exists > 0 {
			f.logger.Warn("Blocked blacklisted IP", "ip", ip)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "access denied",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
