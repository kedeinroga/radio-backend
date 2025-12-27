package handlers

import (
	"net/http"

	"radio-backend/internal/domain"
	"radio-backend/internal/middleware"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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

// Register handles user registration
// @Summary Register new user
// @Description Creates a new guest user account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
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
// @Description Authenticates a user and returns JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Authentication tokens"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	accessToken, refreshToken, err := h.authService.Login(req.Email, req.Password)
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
	})
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Generates a new access token using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} map[string]interface{} "New access token"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid or expired refresh token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	accessToken, err := h.authService.RefreshToken(req.RefreshToken)
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
