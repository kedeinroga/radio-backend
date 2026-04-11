package middleware

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"radio-backend/internal/infrastructure/cache"

	"github.com/gin-gonic/gin"
)

const (
	GuestIPRateLimit  = 150
	GuestIPRateWindow = time.Hour

	guestRateEnabledKey = "guest:rate:enabled"
	guestRateSyncPeriod = 5 * time.Second
)

// GuestIPRateLimiter limits guest requests per IP to GuestIPRateLimit per hour using Redis,
// so the limit is shared across all server instances.
// It is disabled by default; the toggle state is persisted in Redis and synced every 5 seconds.
type GuestIPRateLimiter struct {
	redis   *cache.RedisClient
	enabled int32 // atomic local cache of the Redis toggle state
}

// NewGuestIPRateLimiter creates a new GuestIPRateLimiter backed by Redis.
// It reads the current toggle state from Redis on startup and keeps it in sync.
func NewGuestIPRateLimiter(redisClient *cache.RedisClient) *GuestIPRateLimiter {
	g := &GuestIPRateLimiter{redis: redisClient}
	g.syncEnabled()
	go g.syncLoop()
	return g
}

// IsEnabled returns true if the limiter is currently active.
func (g *GuestIPRateLimiter) IsEnabled() bool {
	return atomic.LoadInt32(&g.enabled) == 1
}

// Toggle flips the enabled state in Redis and updates the local cache immediately.
// Returns the new state.
func (g *GuestIPRateLimiter) Toggle() (bool, error) {
	val, err := g.redis.Get(guestRateEnabledKey)
	newEnabled := true
	if err == nil && val == "1" {
		newEnabled = false
	}

	newVal := "0"
	if newEnabled {
		newVal = "1"
	}

	if err := g.redis.Set(guestRateEnabledKey, newVal, 0); err != nil {
		return false, fmt.Errorf("failed to persist toggle state: %w", err)
	}

	if newEnabled {
		atomic.StoreInt32(&g.enabled, 1)
	} else {
		atomic.StoreInt32(&g.enabled, 0)
	}

	return newEnabled, nil
}

// syncEnabled reads the toggle state from Redis and updates the local atomic cache.
func (g *GuestIPRateLimiter) syncEnabled() {
	val, err := g.redis.Get(guestRateEnabledKey)
	if err != nil {
		// Key absent or Redis error → treat as disabled
		atomic.StoreInt32(&g.enabled, 0)
		return
	}
	if val == "1" {
		atomic.StoreInt32(&g.enabled, 1)
	} else {
		atomic.StoreInt32(&g.enabled, 0)
	}
}

// syncLoop keeps the local enabled cache in sync with Redis every 5 seconds,
// so all instances converge within guestRateSyncPeriod of a toggle.
func (g *GuestIPRateLimiter) syncLoop() {
	ticker := time.NewTicker(guestRateSyncPeriod)
	defer ticker.Stop()
	for range ticker.C {
		g.syncEnabled()
	}
}

// allow increments the Redis counter for the current hour bucket of the given IP.
// Returns false if the counter exceeds GuestIPRateLimit.
// On Redis failure it fails open (allows the request) to avoid blocking real users.
func (g *GuestIPRateLimiter) allow(ip string) bool {
	hourBucket := time.Now().Unix() / 3600
	key := fmt.Sprintf("guest:rate:%s:%d", ip, hourBucket)

	count, err := g.redis.Increment(key)
	if err != nil {
		// Redis unavailable → fail open
		return true
	}

	if count == 1 {
		// First request in this window: set TTL to 2 hours so old keys auto-expire
		_ = g.redis.Expire(key, 2*time.Hour)
	}

	return count <= GuestIPRateLimit
}

// Middleware returns the Gin handler. It:
//   - skips if the feature is disabled
//   - skips /health
//   - skips requests with an Authorization header (authenticated users)
//   - returns 429 when the per-IP hourly limit is exceeded
func (g *GuestIPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !g.IsEnabled() {
			c.Next()
			return
		}

		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Skip authenticated requests — they carry a Bearer token
		if c.GetHeader("Authorization") != "" {
			c.Next()
			return
		}

		if !g.allow(c.ClientIP()) {
			c.Header("Retry-After", "3600")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "guest_rate_limit_exceeded",
					"message": "Too many requests from this IP. Please try again in an hour.",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
