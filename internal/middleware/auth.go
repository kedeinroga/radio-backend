package middleware

import (
	"strings"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware handles authentication
type AuthMiddleware struct {
	tokenValidator domain.TokenValidator
	tokenBlacklist domain.TokenBlacklist
	sessionRepo    domain.SessionRepository
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(tokenValidator domain.TokenValidator, tokenBlacklist domain.TokenBlacklist, sessionRepo domain.SessionRepository) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: tokenValidator,
		tokenBlacklist: tokenBlacklist,
		sessionRepo:    sessionRepo,
	}
}

// Optional is middleware that optionally authenticates users (non-blocking)
func (m *AuthMiddleware) Optional() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := domain.UserTypeGuest
		var userID *string

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != authHeader {
				// Validate token
				claims, err := m.tokenValidator.ValidateToken(token)
				if err == nil && !claims.IsExpired() {
					// Check if token is revoked
					revoked, err := m.tokenBlacklist.IsTokenRevoked(claims.TokenID)
					if err == nil && !revoked {
						// Check if session is revoked
						sessionRevoked, err := m.tokenBlacklist.IsSessionRevoked(claims.SessionID)
						if err == nil && !sessionRevoked {
							userType = claims.UserType
							userID = &claims.UserID

							// Set user context
							c.Set("user_id", claims.UserID)
							c.Set("user_type", claims.UserType)
							c.Set("user_email", claims.Email)
							c.Set("session_id", claims.SessionID)
							c.Set("token_id", claims.TokenID)

							// Add user ID to logger context
							ctx := logger.WithUserID(c.Request.Context(), claims.UserID)
							c.Request = c.Request.WithContext(ctx)
						}
					}
				}
			}
		}

		// Set default user type if not authenticated
		if userID == nil {
			c.Set("user_type", userType)
		}

		c.Next()
	}
}

// Required is middleware that requires authentication
func (m *AuthMiddleware) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.JSON(401, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := m.tokenValidator.ValidateToken(token)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Check if token is expired
		if claims.IsExpired() {
			c.JSON(401, gin.H{"error": "token expired"})
			c.Abort()
			return
		}

		// Check if token is revoked
		revoked, err := m.tokenBlacklist.IsTokenRevoked(claims.TokenID)
		if err != nil || revoked {
			c.JSON(401, gin.H{"error": "token revoked"})
			c.Abort()
			return
		}

		// Check if session is revoked
		sessionRevoked, err := m.tokenBlacklist.IsSessionRevoked(claims.SessionID)
		if err != nil || sessionRevoked {
			c.JSON(401, gin.H{"error": "session revoked"})
			c.Abort()
			return
		}

		// Validate session hijacking (User-Agent)
		if err := m.validateSessionSecurity(claims.SessionID, c.Request.UserAgent()); err != nil {
			logger.Warn("Session validation failed", 
				"session_id", claims.SessionID,
				"user_id", claims.UserID,
				"error", err.Error())
			c.JSON(401, gin.H{"error": "session validation failed"})
			c.Abort()
			return
		}

		// Update session last activity
		go func() {
			if err := m.sessionRepo.UpdateLastActivity(claims.SessionID, time.Now()); err != nil {
				logger.Error("Failed to update session activity", "error", err)
			}
		}()

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("user_type", claims.UserType)
		c.Set("user_email", claims.Email)
		c.Set("session_id", claims.SessionID)
		c.Set("token_id", claims.TokenID)

		// Add user ID to logger context
		ctx := logger.WithUserID(c.Request.Context(), claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// PremiumOnly is middleware that requires premium user
func (m *AuthMiddleware) PremiumOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(403, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Type assertion with safety check
		ut, ok := userType.(domain.UserType)
		if !ok || ut != domain.UserTypePremium {
			c.JSON(403, gin.H{"error": "premium access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOnly is middleware that requires admin user
func (m *AuthMiddleware) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(403, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Type assertion with safety check
		ut, ok := userType.(domain.UserType)
		if !ok || ut != domain.UserTypeAdmin {
			c.JSON(403, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserType returns the user type from context
func GetUserType(c *gin.Context) domain.UserType {
	userType, exists := c.Get("user_type")
	if !exists {
		return domain.UserTypeGuest
	}

	ut, ok := userType.(domain.UserType)
	if !ok {
		return domain.UserTypeGuest
	}

	return ut
}

// GetUserID returns the user ID from context
func GetUserID(c *gin.Context) *string {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil
	}

	id, ok := userID.(string)
	if !ok {
		return nil
	}

	return &id
}

// validateSessionSecurity validates session security by checking User-Agent
func (m *AuthMiddleware) validateSessionSecurity(sessionID string, currentUserAgent string) error {
	session, err := m.sessionRepo.FindByID(sessionID)
	if err != nil {
		return err
	}

	// Validate User-Agent hasn't changed (session hijacking detection)
	if session.UserAgent != currentUserAgent {
		return domain.NewValidationError("session", "user agent mismatch - possible session hijacking")
	}

	// Optional: You can add IP validation here as well
	// if !sameIPRange(session.IPAddress, currentIP, 24) {
	//     return domain.NewValidationError("session", "IP address changed significantly")
	// }

	return nil
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Add request ID to logger context
		ctx := logger.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Log request
		logger.InfoContext(ctx, "incoming request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)

		c.Next()

		// Log response
		duration := time.Since(start)
		logger.InfoContext(ctx, "request completed",
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"size", c.Writer.Size(),
		)
	}
}
