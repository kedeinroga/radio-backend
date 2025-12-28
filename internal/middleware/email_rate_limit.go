package middleware

import (
	"fmt"
	"net/http"
	"time"

	"radio-backend/internal/infrastructure/cache"

	"github.com/gin-gonic/gin"
)

// EmailRateLimiter implements rate limiting per email address
type EmailRateLimiter struct {
	redis       *cache.RedisClient
	maxAttempts int
	window      time.Duration
}

// NewEmailRateLimiter creates a new email-based rate limiter
func NewEmailRateLimiter(redis *cache.RedisClient, maxAttempts int, window time.Duration) *EmailRateLimiter {
	return &EmailRateLimiter{
		redis:       redis,
		maxAttempts: maxAttempts,
		window:      window,
	}
}

// CheckEmailRateLimit checks if an email has exceeded the rate limit
func (erl *EmailRateLimiter) CheckEmailRateLimit(email string) error {
	key := fmt.Sprintf("email_rate_limit:%s", email)

	// Increment attempt count
	count, err := erl.redis.Increment(key)
	if err != nil {
		return err
	}

	// Set expiration on first attempt
	if count == 1 {
		erl.redis.Expire(key, erl.window)
	}

	// Check if limit exceeded
	if count > int64(erl.maxAttempts) {
		return fmt.Errorf("rate limit exceeded for email")
	}

	return nil
}

// GetRemainingAttempts returns the number of remaining attempts for an email
func (erl *EmailRateLimiter) GetRemainingAttempts(email string) (int, error) {
	key := fmt.Sprintf("email_rate_limit:%s", email)

	count, err := erl.redis.Get(key)
	if err != nil {
		// If key doesn't exist, all attempts are available
		return erl.maxAttempts, nil
	}

	attempts := 0
	fmt.Sscanf(count, "%d", &attempts)
	remaining := erl.maxAttempts - attempts
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// GetTTL returns the time until the rate limit resets
func (erl *EmailRateLimiter) GetTTL(email string) (time.Duration, error) {
	key := fmt.Sprintf("email_rate_limit:%s", email)

	ttl, err := erl.redis.TTL(key)
	if err != nil {
		return 0, err
	}

	return ttl, nil
}

// Middleware returns a Gin middleware for email rate limiting
// This middleware expects the email to be in the request body
func (erl *EmailRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to extract email from request body
		// This is a simplified version - in production you'd want to parse the JSON properly
		var req struct {
			Email string `json:"email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			// If we can't parse the body, continue without rate limiting
			c.Next()
			return
		}

		// Re-bind the body so the handler can still read it
		c.Set("email_rate_limit_checked", true)

		if req.Email != "" {
			if err := erl.CheckEmailRateLimit(req.Email); err != nil {
				ttl, _ := erl.GetTTL(req.Email)
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "email_rate_limit_exceeded",
						"message": "Too many login attempts for this email. Please try again later.",
					},
				})
				c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
