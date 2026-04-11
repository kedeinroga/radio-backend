package handlers

import (
	"fmt"
	"net/http"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/middleware"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService      *services.AuthService
	emailRateLimiter *middleware.EmailRateLimiter
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, emailRateLimiter *middleware.EmailRateLimiter) *AuthHandler {
	return &AuthHandler{
		authService:      authService,
		emailRateLimiter: emailRateLimiter,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents a refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ValidateTokenRequest represents a token validation request
type ValidateTokenRequest struct {
	IncludeMetadata bool `json:"include_metadata"`
}

// RevokeTokenRequest represents a token revocation request
type RevokeTokenRequest struct {
	TokenID   string `json:"token_id"`
	SessionID string `json:"session_id"`
	RevokeAll bool   `json:"revoke_all"`
}

// Response structs for Swagger documentation

// LoginResponse represents a successful login response
type LoginResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int    `json:"expires_in" example:"900"`
	ExpiresAt    string `json:"expires_at" example:"2024-01-15T12:30:00Z"`
	SessionID    string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	JTI          string `json:"jti" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// RefreshTokenResponse represents a successful token refresh response
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int    `json:"expires_in" example:"900"`
	ExpiresAt   string `json:"expires_at" example:"2024-01-15T12:30:00Z"`
	SessionID   string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	JTI         string `json:"jti" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// ValidateTokenResponse represents a token validation response
type ValidateTokenResponse struct {
	Valid           bool             `json:"valid" example:"true"`
	UserID          string           `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Email           string           `json:"email" example:"user@example.com"`
	Role            string           `json:"role" example:"guest"`
	ExpiresAt       string           `json:"expires_at" example:"2024-01-15T12:30:00Z"`
	IssuedAt        string           `json:"issued_at" example:"2024-01-15T12:15:00Z"`
	TokenID         string           `json:"token_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	SessionMetadata *SessionMetadata `json:"session_metadata,omitempty"`
}

// SessionMetadata represents session metadata in validation response
type SessionMetadata struct {
	IPAddress    string `json:"ip_address" example:"192.168.1.1"`
	UserAgent    string `json:"user_agent" example:"Mozilla/5.0..."`
	LastActivity string `json:"last_activity" example:"2024-01-15T12:15:00Z"`
	Browser      string `json:"browser" example:"Chrome"`
	OS           string `json:"os" example:"macOS"`
	DeviceType   string `json:"device_type" example:"desktop"`
	Country      string `json:"country" example:"US"`
	City         string `json:"city" example:"New York"`
}

// RevokeTokenResponse represents a token revocation response
type RevokeTokenResponse struct {
	RevokedCount int    `json:"revoked_count" example:"1"`
	Message      string `json:"message" example:"Token revoked successfully"`
}

// SessionInfo represents a single session in the sessions list
type SessionInfo struct {
	SessionID    string     `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TokenID      string     `json:"token_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	DeviceInfo   DeviceInfo `json:"device_info"`
	Location     Location   `json:"location"`
	CreatedAt    string     `json:"created_at" example:"2024-01-15T12:00:00Z"`
	LastActivity string     `json:"last_activity" example:"2024-01-15T12:15:00Z"`
	ExpiresAt    string     `json:"expires_at" example:"2024-01-15T20:00:00Z"`
	IsCurrent    bool       `json:"is_current" example:"true"`
}

// DeviceInfo represents device information
type DeviceInfo struct {
	UserAgent  string `json:"user_agent" example:"Mozilla/5.0..."`
	Browser    string `json:"browser" example:"Chrome"`
	OS         string `json:"os" example:"macOS"`
	DeviceType string `json:"device_type" example:"desktop"`
}

// Location represents location information
type Location struct {
	IP      string `json:"ip" example:"192.168.1.1"`
	Country string `json:"country" example:"US"`
	City    string `json:"city" example:"New York"`
}

// GetSessionsResponse represents a sessions list response
type GetSessionsResponse struct {
	Sessions      []SessionInfo `json:"sessions"`
	TotalSessions int           `json:"total_sessions" example:"3"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool        `json:"success" example:"false"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string `json:"code" example:"invalid_token"`
	Message string `json:"message" example:"Token is invalid, expired or revoked"`
}

// Register handles user registration
// @Summary Register new user
// @Description Creates a new guest user account.
// @Description
// @Description **Password Requirements:**
// @Description - Minimum 8 characters
// @Description - At least one special character (!@#$%^&*(),.?":{}|<>)
// @Description - Cannot be one of 47 common passwords (e.g., 'password', '12345678', 'qwerty')
// @Description - Passwords are hashed with bcrypt (cost factor 12)
// @Description
// @Description **Security Features:**
// @Description - Email uniqueness validation (generic error to prevent enumeration)
// @Description - Rate limiting inherited from IP-based middleware
// @Description - HTTPS enforcement in production
// @Description
// @Description **Error Codes:**
// @Description - `WEAK_PASSWORD`: Password does not meet strength requirements
// @Description - `COMMON_PASSWORD`: Password is in the common passwords blacklist
// @Description - `EMAIL_EXISTS`: Email already registered (generic message)
// @Description - `VALIDATION_ERROR`: Invalid input format
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or password does not meet requirements"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	user, err := h.authService.Register(req.Email, req.Password)
	if err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			RespondWithDomainError(c, domainErr)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "registration_failed", "Failed to register user")
		return
	}

	RespondWithSuccess(c, http.StatusCreated, gin.H{
		"user": gin.H{
			"id":        user.ID,
			"email":     user.Email,
			"user_type": user.UserType.String(),
		},
	})
}

// Login handles user login
// @Summary User login
// @Description Authenticates a user and returns JWT tokens with session information.
// @Description Includes access token, refresh token, session ID, and token metadata.
// @Description
// @Description **Security Features:**
// @Description - Password hashing: bcrypt with cost factor 12
// @Description - Timing attack prevention: Constant-time operations
// @Description - Rate limiting: 5 attempts per IP per 15 minutes
// @Description - Email rate limiting: 10 attempts per email per hour
// @Description - Account lockout: 10 failed attempts = 30 minute lockout
// @Description - Session hijacking protection: User-Agent validation
// @Description - Security event logging: All failed login attempts logged
// @Description - HTTPS enforcement: Production environment only
// @Description
// @Description **Error Codes:**
// @Description - `INVALID_CREDENTIALS`: Invalid email or password
// @Description - `ACCOUNT_LOCKED`: Account temporarily locked due to multiple failed attempts
// @Description - `TOO_MANY_ATTEMPTS`: Rate limit exceeded (IP or email)
// @Description - `VALIDATION_ERROR`: Invalid input format
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials (email and password)"
// @Success 200 {object} LoginResponse "Authentication tokens with session metadata"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Invalid credentials or account locked"
// @Failure 429 {object} ErrorResponse "Too many login attempts (rate limit exceeded)"
// @Header 429 {integer} Retry-After "Number of seconds to wait before retrying"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Check email rate limit
	if err := h.emailRateLimiter.CheckEmailRateLimit(req.Email); err != nil {
		ttl, _ := h.emailRateLimiter.GetTTL(req.Email)
		c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
		RespondWithError(c, http.StatusTooManyRequests, "email_rate_limit_exceeded",
			"Too many login attempts for this email. Please try again later.")
		return
	}

	// Get IP address and user agent
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	accessToken, refreshToken, sessionID, tokenID, expiresAt, err := h.authService.Login(req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			RespondWithDomainError(c, domainErr)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "login_failed", "Failed to login")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900, // 15 minutes in seconds
		"expires_at":    expiresAt.Format(time.RFC3339),
		"session_id":    sessionID,
		"jti":           tokenID,
	})
}

// RefreshToken handles token refresh
// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Generates a new access token using a valid refresh token
// @Description Returns new access token with session ID and token metadata
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} RefreshTokenResponse "New access token with metadata"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Get IP address and user agent
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	accessToken, sessionID, tokenID, expiresAt, err := h.authService.RefreshToken(req.RefreshToken, ipAddress, userAgent)
	if err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			RespondWithDomainError(c, domainErr)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "refresh_failed", "Failed to refresh token")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   900, // 15 minutes in seconds
		"expires_at":   expiresAt.Format(time.RFC3339),
		"session_id":   sessionID,
		"jti":          tokenID,
	})
}

// Me returns the current user info
// @Summary Get current user information
// @Description Returns the authenticated user information
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User information"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == nil {
		RespondWithError(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}

	userType := middleware.GetUserType(c)
	email, _ := c.Get("user_email")

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":        *userID,
			"email":     email,
			"user_type": userType.String(),
		},
	})
}

// ValidateToken validates a token and returns its claims
// ValidateToken validates a JWT token and returns its claims
// @Summary Validate access token
// @Description Validates a JWT token and returns its claims and metadata
// @Description Checks if token is valid, not expired, and not revoked
// @Description Optional session metadata includes browser, OS, IP, and location
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ValidateTokenRequest false "Validation options (include_metadata)"
// @Success 200 {object} ValidateTokenResponse "Token validation result with claims"
// @Failure 401 {object} ErrorResponse "Invalid, expired or revoked token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/validate [post]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		RespondWithError(c, http.StatusUnauthorized, "missing_token", "Authorization header is required")
		return
	}

	// Extract token (remove "Bearer " prefix)
	token := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	var req ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.IncludeMetadata = false
	}

	response, err := h.authService.ValidateToken(token, req.IncludeMetadata)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "invalid_token", "Token is invalid, expired or revoked")
		return
	}

	data := gin.H{
		"valid":      response.Valid,
		"user_id":    response.UserID,
		"email":      response.Email,
		"role":       response.Role,
		"expires_at": response.ExpiresAt.Format(time.RFC3339),
		"issued_at":  response.IssuedAt.Format(time.RFC3339),
		"token_id":   response.TokenID,
	}

	if req.IncludeMetadata && response.SessionMetadata != nil {
		data["session_metadata"] = gin.H{
			"ip_address":    response.SessionMetadata.IPAddress,
			"user_agent":    response.SessionMetadata.UserAgent,
			"last_activity": response.SessionMetadata.LastActivity.Format(time.RFC3339),
			"browser":       response.SessionMetadata.Browser,
			"os":            response.SessionMetadata.OS,
			"device_type":   response.SessionMetadata.DeviceType,
			"country":       response.SessionMetadata.Country,
			"city":          response.SessionMetadata.City,
		}
	}

	RespondWithSuccess(c, http.StatusOK, data)
}

// RevokeToken revokes tokens
// RevokeToken handles token revocation
// @Summary Revoke tokens
// @Description Revokes tokens by token ID, session ID, or all user tokens
// @Description Use token_id to revoke specific token, session_id for all session tokens, or revoke_all for all user tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RevokeTokenRequest true "Revocation request (token_id, session_id, or revoke_all)"
// @Success 200 {object} RevokeTokenResponse "Number of tokens revoked"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Failed to revoke token"
// @Router /auth/revoke [post]
func (h *AuthHandler) RevokeToken(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == nil {
		RespondWithError(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}

	var req RevokeTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	var count int
	var err error

	if req.RevokeAll {
		// Revoke all user tokens
		err = h.authService.RevokeAllUserTokens(*userID)
		if err == nil {
			count = 1 // Indicate success
		}
	} else {
		// Revoke specific token or session
		revokeRequest := &domain.RevokeTokenRequest{
			TokenID:   req.TokenID,
			SessionID: req.SessionID,
			RevokeAll: false,
		}
		count, err = h.authService.RevokeToken(revokeRequest)
	}

	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "revocation_failed", "Failed to revoke token")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"revoked_count": count,
		"message":       "Token revoked successfully",
	})
}

// GetSessions returns all active sessions for the current user
// @Summary Get user sessions
// @Description Returns all active sessions for the authenticated user
// @Description Includes device info, location, and activity timestamps for each session
// @Description Current session is marked with is_current flag
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetSessionsResponse "List of active user sessions"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Failed to retrieve sessions"
// @Router /auth/sessions [get]
func (h *AuthHandler) GetSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == nil {
		RespondWithError(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}

	sessions, err := h.authService.GetUserSessions(*userID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed_to_get_sessions", "Failed to retrieve sessions")
		return
	}

	// Get current session ID from token
	currentSessionID := ""
	if sid, exists := c.Get("session_id"); exists {
		if sidStr, ok := sid.(string); ok {
			currentSessionID = sidStr
		}
	}

	sessionList := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		sessionList = append(sessionList, gin.H{
			"session_id": session.SessionID,
			"token_id":   session.TokenID,
			"device_info": gin.H{
				"user_agent":  session.UserAgent,
				"browser":     session.DeviceInfo["browser"],
				"os":          session.DeviceInfo["os"],
				"device_type": session.DeviceInfo["device_type"],
			},
			"location": gin.H{
				"ip":      session.IPAddress,
				"country": session.DeviceInfo["country"],
				"city":    session.DeviceInfo["city"],
			},
			"created_at":    session.CreatedAt.Format(time.RFC3339),
			"last_activity": session.LastActivity.Format(time.RFC3339),
			"expires_at":    session.ExpiresAt.Format(time.RFC3339),
			"is_current":    session.SessionID == currentSessionID,
		})
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"sessions":       sessionList,
		"total_sessions": len(sessionList),
	})
}

// DeleteSession deletes a specific session
// @Summary Delete session
// @Description Terminates a specific user session
// @Description Revokes the session and all associated tokens
// @Description Cannot delete current session
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param sessionId path string true "Session ID to delete"
// @Success 200 {object} SuccessResponse "Session terminated successfully"
// @Failure 400 {object} ErrorResponse "Invalid or missing session ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Failed to delete session"
// @Router /auth/sessions/{sessionId} [delete]
func (h *AuthHandler) DeleteSession(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == nil {
		RespondWithError(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}

	sessionID := c.Param("sessionId")
	if sessionID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Session ID is required")
		return
	}

	err := h.authService.DeleteSession(sessionID, *userID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed_to_delete_session", "Failed to delete session")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "Session terminated successfully",
	})
}

// Logout logs out the current user by blacklisting their token
// @Summary User logout
// @Description Logs out the current user by blacklisting their JWT token
// @Description The token will be invalidated and cannot be used again
// @Description The user must login again to get a new token
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.LogoutResponse "Successfully logged out"
// @Failure 400 {object} ErrorResponse "Invalid request or missing token"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Failed to logout"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID := middleware.GetUserID(c)
	if userID == nil {
		RespondWithError(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}

	// Get token JTI from context (set by auth middleware)
	tokenID := middleware.GetTokenID(c)
	if tokenID == nil || *tokenID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_token", "Token ID not found")
		return
	}

	// Get IP address and user agent
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Create logout request
	logoutReq := &domain.LogoutRequest{
		TokenID:   *tokenID,
		UserID:    *userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Reason:    "user_logout",
	}

	// Perform logout
	response, err := h.authService.Logout(logoutReq)
	if err != nil {
		// Check if it's a validation error
		if domainErr, ok := err.(*domain.DomainError); ok {
			RespondWithError(c, http.StatusBadRequest, domainErr.Code, domainErr.Message)
			return
		}

		RespondWithError(c, http.StatusInternalServerError, "logout_failed", "Failed to logout")
		return
	}

	RespondWithSuccess(c, http.StatusOK, response)
}
