package middleware

import (
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AnalyticsMiddleware tracks analytics
type AnalyticsMiddleware struct {
	analyticsService *services.AnalyticsService
}

// NewAnalyticsMiddleware creates a new analytics middleware
func NewAnalyticsMiddleware(analyticsService *services.AnalyticsService) *AnalyticsMiddleware {
	return &AnalyticsMiddleware{analyticsService: analyticsService}
}

// Track tracks request analytics
func (m *AnalyticsMiddleware) Track() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// Get request details
		requestID, _ := c.Get("request_id")
		userType := GetUserType(c)
		userID := GetUserID(c)

		// Create request log
		log := &domain.RequestLog{
			RequestID:  requestID.(string),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			UserID:     userID,
			UserType:   userType,
			StatusCode: c.Writer.Status(),
			Duration:   time.Since(start),
			IPAddress:  c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
		}

		// Track asynchronously (don't block the response)
		go func() {
			_ = m.analyticsService.TrackRequest(log)
		}()
	}
}
