package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UserAgentFilter blocks requests whose User-Agent contains "axios" with a 403
// Forbidden response. Requests whose User-Agent contains "node" are allowed
// through unchanged. All other User-Agents also pass through; adjust the logic
// below if you want to adopt a stricter allowlist approach.
func UserAgentFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ua := strings.ToLower(c.GetHeader("User-Agent"))

		if strings.Contains(ua, "axios") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Client not allowed",
			})
			return
		}

		c.Next()
	}
}
