package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related HTTP headers to responses
func SecurityHeaders(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Enable browser's XSS protection (legacy, but still useful)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Only enable HSTS in production with HTTPS
		if isProduction && c.Request.TLS != nil {
			// Force HTTPS for 1 year, including subdomains
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Content Security Policy - restrict resource loading
		// Adjust based on your frontend needs
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline'; " + // Allow inline scripts (adjust as needed)
			"style-src 'self' 'unsafe-inline'; " + // Allow inline styles
			"img-src 'self' data: https:; " + // Allow images from HTTPS sources
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"media-src 'self' https:; " + // Allow media from HTTPS sources (for radio streams)
			"object-src 'none'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"

		c.Header("Content-Security-Policy", csp)

		// Permissions Policy - restrict browser features
		permissionsPolicy := "geolocation=(), " +
			"microphone=(), " +
			"camera=(), " +
			"payment=(), " +
			"usb=(), " +
			"magnetometer=(), " +
			"accelerometer=(), " +
			"gyroscope=(), " +
			"picture-in-picture=()"

		c.Header("Permissions-Policy", permissionsPolicy)

		// Remove Server header to avoid leaking implementation details
		c.Header("Server", "")

		c.Next()
	}
}
