package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AdRateLimiter provides rate limiting for ad-related endpoints
type AdRateLimiter struct {
	redis  *redis.Client
	logger *slog.Logger
}

// NewAdRateLimiter creates a new ad rate limiter
func NewAdRateLimiter(redis *redis.Client, logger *slog.Logger) *AdRateLimiter {
	return &AdRateLimiter{
		redis:  redis,
		logger: logger,
	}
}

// RateLimitConfig defines rate limit configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
}

// DefaultAdRateLimitConfig returns default configuration for ad endpoints
func DefaultAdRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         20,
	}
}

// LimitByIP rate limits requests by IP address
func (r *AdRateLimiter) LimitByIP(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := context.Background()

		// Redis key for this IP
		key := fmt.Sprintf("ratelimit:ad:ip:%s", ip)

		// Get current count
		count, err := r.redis.Incr(ctx, key).Result()
		if err != nil {
			r.logger.Error("Failed to increment rate limit counter", "error", err, "ip", ip)
			// Allow request on error (fail open)
			c.Next()
			return
		}

		// Set expiration on first request
		if count == 1 {
			r.redis.Expire(ctx, key, time.Minute)
		}

		// Check if limit exceeded
		if count > int64(config.RequestsPerMinute+config.BurstSize) {
			r.logger.Warn("Rate limit exceeded", "ip", ip, "count", count)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": fmt.Sprintf("Maximum %d requests per minute allowed", config.RequestsPerMinute),
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", config.RequestsPerMinute-int(count)))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

		c.Next()
	}
}

// LimitImpressionTracking provides stricter rate limiting for impression tracking
func (r *AdRateLimiter) LimitImpressionTracking() gin.HandlerFunc {
	// Stricter limits for impression tracking (prevent fraud)
	config := RateLimitConfig{
		RequestsPerMinute: 60, // Max 60 impressions per minute per IP
		BurstSize:         10,
	}
	return r.LimitByIP(config)
}

// LimitClickTracking provides stricter rate limiting for click tracking
func (r *AdRateLimiter) LimitClickTracking() gin.HandlerFunc {
	// Even stricter for clicks (more valuable)
	config := RateLimitConfig{
		RequestsPerMinute: 30, // Max 30 clicks per minute per IP
		BurstSize:         5,
	}
	return r.LimitByIP(config)
}
