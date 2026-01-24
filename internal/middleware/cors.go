package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware handles CORS
func CORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" {
				allowed = true
				break
			}
			if allowedOrigin == origin && origin != "" {
				allowed = true
				break
			}
		}

		// Only set CORS headers if origin is allowed
		if allowed {
			// If wildcard, use it; otherwise use the specific origin
			if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			}

			c.Header("Access-Control-Allow-Methods", joinStrings(allowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", joinStrings(allowedHeaders, ", "))
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}

		c.Next()
	}
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}
