package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter implements a per-IP rate limiter using token bucket algorithm
type RateLimiter struct {
	limiters map[string]*rateLimiterClient
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

type rateLimiterClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
// requestsPerMinute: maximum number of requests allowed per minute per IP
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rateLimiterClient),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0), // Convert to per-second rate
		burst:    requestsPerMinute,
		cleanup:  5 * time.Minute,
	}

	// Start cleanup goroutine to remove old limiters
	go rl.cleanupRoutine()

	return rl
}

// getLimiter returns the rate limiter for a given IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = &rateLimiterClient{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	client.lastSeen = time.Now()
	return client.limiter
}

// cleanupRoutine removes inactive rate limiters to prevent memory leaks
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, client := range rl.limiters {
			if now.Sub(client.lastSeen) > rl.cleanup {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns a Gin middleware handler
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := rl.getLimiter(c.ClientIP())

		if !limiter.Allow() {
			c.JSON(429, gin.H{
				"error": gin.H{
					"code":    "rate_limit_exceeded",
					"message": "Too many requests. Please try again later.",
				},
			})
			c.Header("Retry-After", "60") // Suggest retry after 60 seconds
			c.Abort()
			return
		}

		c.Next()
	}
}

// StrictMiddleware returns a stricter rate limiter for sensitive endpoints
func (rl *RateLimiter) StrictMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := rl.getLimiter(c.ClientIP())

		if !limiter.Allow() {
			c.JSON(429, gin.H{
				"error": gin.H{
					"code":    "rate_limit_exceeded",
					"message": "Too many login attempts. Please try again in a few minutes.",
				},
			})
			c.Header("Retry-After", "300") // Suggest retry after 5 minutes
			c.Abort()
			return
		}

		c.Next()
	}
}
