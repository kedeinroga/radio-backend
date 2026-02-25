package middleware

import (
	"crypto/subtle"
	"log"

	"github.com/gin-gonic/gin"
)

const sharedSecretHeader = "X-Rradio-Secret"

// SharedSecretAuth returns a Gin middleware that validates the X-Rradio-Secret header
// against the provided apiSecret. If apiSecret is empty, the middleware is disabled
// (pass-through), which is useful for local development and CI environments.
//
// Usage in router:
//
//	stations.Use(middleware.SharedSecretAuth(cfg.Security.APISecretKey))
func SharedSecretAuth(apiSecret string) gin.HandlerFunc {
	if apiSecret == "" {
		// Disabled: let all requests through (warn logged at startup in main.go)
		return func(c *gin.Context) {
			c.Next()
		}
	}

	secretBytes := []byte(apiSecret)

	return func(c *gin.Context) {
		incoming := c.GetHeader(sharedSecretHeader)
		if incoming == "" {
			log.Printf("[WARN] shared secret header missing path=%s ip=%s", c.Request.URL.Path, c.ClientIP())
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		// constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(incoming), secretBytes) != 1 {
			log.Printf("[WARN] shared secret mismatch path=%s ip=%s", c.Request.URL.Path, c.ClientIP())
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}
