package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxRequestSize limits the size of request bodies to prevent memory exhaustion attacks
func MaxRequestSize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set maximum request body size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		// Continue to next middleware/handler
		c.Next()

		// Check if request body was too large
		if c.Writer.Status() == http.StatusRequestEntityTooLarge {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{
					"code":    "request_too_large",
					"message": "Request body is too large. Maximum size is 10MB.",
				},
			})
			c.Abort()
		}
	}
}
