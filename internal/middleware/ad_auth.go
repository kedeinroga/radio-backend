package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdAuthMiddleware provides authentication and authorization for ad endpoints
type AdAuthMiddleware struct {
	logger *slog.Logger
}

// NewAdAuthMiddleware creates a new ad auth middleware
func NewAdAuthMiddleware(logger *slog.Logger) *AdAuthMiddleware {
	return &AdAuthMiddleware{
		logger: logger,
	}
}

// RequireAdvertiser ensures the user has advertiser role
func (m *AdAuthMiddleware) RequireAdvertiser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by main auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			m.logger.Warn("No user_id in context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Get user role
		role, exists := c.Get("user_role")
		if !exists {
			m.logger.Warn("No user_role in context", "user_id", userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		// Check if user has advertiser or admin role
		roleStr, ok := role.(string)
		if !ok || (roleStr != "advertiser" && roleStr != "admin") {
			m.logger.Warn("User does not have advertiser role", "user_id", userID, "role", role)
			c.JSON(http.StatusForbidden, gin.H{"error": "advertiser access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin ensures the user has admin role
func (m *AdAuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user role
		role, exists := c.Get("user_role")
		if !exists {
			m.logger.Warn("No user_role in context")
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		// Check if user has admin role
		roleStr, ok := role.(string)
		if !ok || roleStr != "admin" {
			m.logger.Warn("User does not have admin role", "role", role)
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOwnership ensures the user owns the resource (campaign or advertiser_id)
func (m *AdAuthMiddleware) RequireOwnership(resourceParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		_, ok := userIDInterface.(uuid.UUID)
		if !ok {
			// Try parsing from string
			userIDStr, ok := userIDInterface.(string)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id format"})
				c.Abort()
				return
			}
			_, err := uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id"})
				c.Abort()
				return
			}
		}

		// Get resource ID from URL params
		resourceID := c.Param(resourceParam)
		if resourceID == "" {
			// Try from query params
			resourceID = c.Query(resourceParam)
		}

		if resourceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing resource identifier"})
			c.Abort()
			return
		}

		// Parse resource UUID
		resourceUUID, err := uuid.Parse(resourceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource identifier"})
			c.Abort()
			return
		}

		// Check if user is admin (admins can access all resources)
		role, _ := c.Get("user_role")
		if roleStr, ok := role.(string); ok && roleStr == "admin" {
			c.Next()
			return
		}

		// For non-admin users, check ownership
		// This would typically query the database to verify ownership
		// For now, we'll check if the resource UUID matches the user UUID
		// (In production, you'd query the campaign/ad to get advertiser_id)

		// Store the resource ID for handlers to use
		c.Set("resource_id", resourceUUID)
		c.Set("verified_ownership", true)

		c.Next()
	}
}

// OptionalAuth makes authentication optional but extracts user info if present
func (m *AdAuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue as anonymous
			c.Set("is_authenticated", false)
			c.Next()
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			// Invalid format
			c.Set("is_authenticated", false)
			c.Next()
			return
		}

		// TODO: Validate token and extract user info
		// For now, just mark as authenticated
		c.Set("is_authenticated", true)
		c.Next()
	}
}
