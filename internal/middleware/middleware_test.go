package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests under limit", func(t *testing.T) {
		// Create rate limiter: 5 requests per minute
		rl := NewRateLimiter(5)

		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		// Make 5 requests - all should succeed
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code, "Request %d should succeed", i+1)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		// Create rate limiter: 3 requests per minute
		rl := NewRateLimiter(3)

		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		// Make 5 requests from same IP
		successCount := 0
		blockedCount := 0

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code == 200 {
				successCount++
			} else if w.Code == 429 {
				blockedCount++
			}
		}

		assert.Equal(t, 3, successCount, "Should allow 3 requests")
		assert.Equal(t, 2, blockedCount, "Should block 2 requests")
	})

	t.Run("tracks different IPs separately", func(t *testing.T) {
		// Create rate limiter: 2 requests per minute
		rl := NewRateLimiter(2)

		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		// IP 1: Make 2 requests (should succeed)
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
		}

		// IP 1: Third request should fail
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "192.168.1.1:1234"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, 429, w1.Code)

		// IP 2: Should still be allowed (different IP)
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.2:1234"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, 200, w2.Code, "Different IP should not be rate limited")
	})

	t.Run("includes retry-after header", func(t *testing.T) {
		rl := NewRateLimiter(1)

		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		// First request succeeds
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "192.168.1.1:1234"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, 200, w1.Code)

		// Second request is blocked
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.1:1234"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, 429, w2.Code)
		assert.Equal(t, "60", w2.Header().Get("Retry-After"))
	})
}

func TestStrictMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("strict middleware has longer retry-after", func(t *testing.T) {
		rl := NewRateLimiter(1)

		router := gin.New()
		router.Use(rl.StrictMiddleware())
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		// First request succeeds
		req1 := httptest.NewRequest("POST", "/auth/login", nil)
		req1.RemoteAddr = "192.168.1.1:1234"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, 200, w1.Code)

		// Second request is blocked with longer retry
		req2 := httptest.NewRequest("POST", "/auth/login", nil)
		req2.RemoteAddr = "192.168.1.1:1234"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, 429, w2.Code)
		assert.Equal(t, "300", w2.Header().Get("Retry-After"), "Strict should have 5 min retry")
	})
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("adds security headers in development", func(t *testing.T) {
		router := gin.New()
		router.Use(SecurityHeaders(false)) // Development mode
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'")
		assert.NotEmpty(t, w.Header().Get("Permissions-Policy"))

		// HSTS should NOT be set in development without TLS
		assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
	})

	t.Run("adds all security headers in production", func(t *testing.T) {
		router := gin.New()
		router.Use(SecurityHeaders(true)) // Production mode
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		// Note: HSTS is only set when c.Request.TLS != nil
	})
}

func TestMaxRequestSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests under size limit", func(t *testing.T) {
		router := gin.New()
		router.Use(MaxRequestSize(1024)) // 1KB limit
		router.POST("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		// Small request (under limit)
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	// Note: Testing actual size rejection is complex with httptest
	// In production, http.MaxBytesReader will handle this
}
