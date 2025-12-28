package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ForceHTTPS redirects HTTP requests to HTTPS in production
func ForceHTTPS(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isProduction {
			c.Next()
			return
		}

		// Check X-Forwarded-Proto header (for reverse proxies like nginx, ALB, etc.)
		proto := c.Request.Header.Get("X-Forwarded-Proto")
		if proto == "" {
			// Check if direct TLS connection
			if c.Request.TLS == nil {
				proto = "http"
			} else {
				proto = "https"
			}
		}

		// Redirect HTTP to HTTPS
		if proto != "https" {
			httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(http.StatusMovedPermanently, httpsURL)
			c.Abort()
			return
		}

		c.Next()
	}
}
