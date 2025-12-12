package services

import (
	"fmt"
	"regexp"
	"strings"

	"radio-backend/internal/domain"
)

// AuthService implements domain.AuthService
type AuthService struct {
	userRepo       domain.UserRepository
	passwordHasher domain.PasswordHasher
	tokenGenerator domain.TokenGenerator
	tokenValidator domain.TokenValidator
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo domain.UserRepository,
	passwordHasher domain.PasswordHasher,
	tokenGenerator domain.TokenGenerator,
	tokenValidator domain.TokenValidator,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		tokenGenerator: tokenGenerator,
		tokenValidator: tokenValidator,
	}
}

// Register registers a new user
func (s *AuthService) Register(email, password string) (*domain.User, error) {
	// Validate email
	if err := s.validateEmail(email); err != nil {
		return nil, err
	}

	// Validate password
	if err := s.validatePassword(password); err != nil {
		return nil, err
	}

	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		Email:        email,
		PasswordHash: hashedPassword,
		UserType:     domain.UserTypeGuest, // Default to guest
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(email, password string) (string, string, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("failed to find user: %w", err)
	}

	// Compare password
	if err := s.passwordHasher.Compare(user.PasswordHash, password); err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// RefreshToken generates a new access token from a refresh token
func (s *AuthService) RefreshToken(refreshToken string) (string, error) {
	// Validate refresh token
	claims, err := s.tokenValidator.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Get user
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new access token
	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return accessToken, nil
}

// ValidateToken validates a token and returns claims
func (s *AuthService) ValidateToken(token string) (*domain.TokenClaims, error) {
	return s.tokenValidator.ValidateToken(token)
}

// validateEmail validates an email address
func (s *AuthService) validateEmail(email string) error {
	if email == "" {
		return domain.NewValidationError("email", "email is required")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return domain.NewValidationError("email", "invalid email format")
	}

	return nil
}

// validatePassword validates a password
func (s *AuthService) validatePassword(password string) error {
	if password == "" {
		return domain.NewValidationError("password", "password is required")
	}

	if len(password) < 8 {
		return domain.NewValidationError("password", "password must be at least 8 characters")
	}

	// Check for at least one uppercase, one lowercase, and one number
	hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasDigit := strings.ContainsAny(password, "0123456789")

	if !hasUpper || !hasLower || !hasDigit {
		return domain.NewValidationError("password", "password must contain uppercase, lowercase, and digit")
	}

	return nil
}
