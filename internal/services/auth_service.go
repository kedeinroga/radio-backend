package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"radio-backend/internal/domain"
)

// AuthService implements domain.AuthService
type AuthService struct {
	userRepo          domain.UserRepository
	sessionRepo       domain.SessionRepository
	securityEventRepo SecurityEventRepository
	loginAttemptRepo  domain.LoginAttemptRepository
	passwordHasher    domain.PasswordHasher
	tokenGenerator    domain.TokenGenerator
	tokenValidator    domain.TokenValidator
	tokenBlacklist    domain.TokenBlacklist
	dummyHash         string // Pre-calculated hash for timing attack prevention
}

// SecurityEventRepository interface for logging security events
type SecurityEventRepository interface {
	Create(event *domain.SecurityEvent) error
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	securityEventRepo SecurityEventRepository,
	loginAttemptRepo domain.LoginAttemptRepository,
	passwordHasher domain.PasswordHasher,
	tokenGenerator domain.TokenGenerator,
	tokenValidator domain.TokenValidator,
	tokenBlacklist domain.TokenBlacklist,
) *AuthService {
	// Pre-calculate dummy hash for timing attack prevention
	// This is a bcrypt hash of "dummy_password_for_timing_attack_prevention"
	dummyHash, _ := passwordHasher.Hash("dummy_password_for_timing_attack_prevention_12345")

	return &AuthService{
		userRepo:          userRepo,
		sessionRepo:       sessionRepo,
		securityEventRepo: securityEventRepo,
		loginAttemptRepo:  loginAttemptRepo,
		passwordHasher:    passwordHasher,
		tokenGenerator:    tokenGenerator,
		tokenValidator:    tokenValidator,
		tokenBlacklist:    tokenBlacklist,
		dummyHash:         dummyHash,
	}
}

// generateSessionID generates a unique session ID
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "session_" + hex.EncodeToString(b), nil
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

// Login authenticates a user and returns tokens with session info
func (s *AuthService) Login(email, password string, ipAddress string, userAgent string) (string, string, string, string, time.Time, error) {
	// Check if account is locked
	if err := s.checkAccountLockout(email); err != nil {
		return "", "", "", "", time.Time{}, err
	}

	// Find user by email
	user, err := s.userRepo.FindByEmail(email)

	// Timing attack prevention: Always execute bcrypt.Compare to maintain constant time
	// even when user doesn't exist
	if err != nil {
		if err == domain.ErrUserNotFound {
			// Execute bcrypt with dummy hash to prevent timing leak
			s.passwordHasher.Compare(s.dummyHash, password)

			// Increment failed attempts
			s.incrementFailedAttempts(email)

			// Log failed login attempt
			s.LogSecurityEvent(&domain.SecurityEvent{
				Timestamp: time.Now(),
				Event:     "login.failed",
				UserID:    "",
				IPAddress: ipAddress,
				UserAgent: userAgent,
				Reason:    "user_not_found",
				Metadata: map[string]interface{}{
					"email": email,
				},
			})

			return "", "", "", "", time.Time{}, domain.ErrInvalidCredentials
		}
		return "", "", "", "", time.Time{}, fmt.Errorf("failed to find user: %w", err)
	}

	// Compare password
	if err := s.passwordHasher.Compare(user.PasswordHash, password); err != nil {
		// Increment failed attempts
		s.incrementFailedAttempts(email)

		// Log failed login attempt
		s.LogSecurityEvent(&domain.SecurityEvent{
			Timestamp: time.Now(),
			Event:     "login.failed",
			UserID:    user.ID,
			IPAddress: ipAddress,
			UserAgent: userAgent,
			Reason:    "invalid_password",
			Metadata: map[string]interface{}{
				"email": email,
			},
		})

		return "", "", "", "", time.Time{}, domain.ErrInvalidCredentials
	}

	// Reset failed attempts on successful login
	s.resetFailedAttempts(email)

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return "", "", "", "", time.Time{}, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Generate tokens
	accessToken, tokenID, err := s.tokenGenerator.GenerateAccessToken(user, sessionID, ipAddress, userAgent)
	if err != nil {
		return "", "", "", "", time.Time{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user, sessionID)
	if err != nil {
		return "", "", "", "", time.Time{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session record
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7 days
	session := &domain.Session{
		UserID:       user.ID,
		SessionID:    sessionID,
		TokenID:      tokenID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		DeviceInfo:   parseUserAgent(userAgent),
		CreatedAt:    now,
		LastActivity: now,
		ExpiresAt:    expiresAt,
		IsActive:     true,
	}

	if err := s.sessionRepo.Create(session); err != nil {
		// Log error but don't fail login
		fmt.Printf("failed to create session: %v\n", err)
	}

	// Log security event
	s.LogSecurityEvent(&domain.SecurityEvent{
		Timestamp: now,
		Event:     "token.issued",
		UserID:    user.ID,
		TokenID:   tokenID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Reason:    "login",
		Metadata: map[string]interface{}{
			"session_id": sessionID,
		},
	})

	return accessToken, refreshToken, sessionID, tokenID, expiresAt, nil
}

// RefreshToken generates a new access token from a refresh token
func (s *AuthService) RefreshToken(refreshToken string, ipAddress string, userAgent string) (string, string, string, time.Time, error) {
	// Validate refresh token
	claims, err := s.tokenValidator.ValidateToken(refreshToken)
	if err != nil {
		return "", "", "", time.Time{}, err
	}

	// Check if token is revoked
	revoked, err := s.tokenBlacklist.IsTokenRevoked(claims.TokenID)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("failed to check token revocation: %w", err)
	}
	if revoked {
		return "", "", "", time.Time{}, domain.ErrInvalidToken
	}

	// Check if session is revoked
	sessionRevoked, err := s.tokenBlacklist.IsSessionRevoked(claims.SessionID)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("failed to check session revocation: %w", err)
	}
	if sessionRevoked {
		return "", "", "", time.Time{}, domain.ErrInvalidToken
	}

	// Get user
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new access token with same session ID
	accessToken, tokenID, err := s.tokenGenerator.GenerateAccessToken(user, claims.SessionID, ipAddress, userAgent)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Update session last activity
	now := time.Now()
	if err := s.sessionRepo.UpdateLastActivity(claims.SessionID, now); err != nil {
		// Log error but don't fail refresh
		fmt.Printf("failed to update session activity: %v\n", err)
	}

	expiresAt := now.Add(15 * time.Minute) // 15 minutes for access token

	// Log security event
	s.LogSecurityEvent(&domain.SecurityEvent{
		Timestamp: now,
		Event:     "token.refreshed",
		UserID:    user.ID,
		TokenID:   tokenID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Reason:    "refresh",
		Metadata: map[string]interface{}{
			"session_id":   claims.SessionID,
			"old_token_id": claims.TokenID,
		},
	})

	return accessToken, claims.SessionID, tokenID, expiresAt, nil
}

// ValidateToken validates a token and returns claims
func (s *AuthService) ValidateToken(token string, includeMetadata bool) (*domain.TokenValidationResponse, error) {
	// Validate token structure and signature
	claims, err := s.tokenValidator.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// Check if token is expired
	if claims.IsExpired() {
		return nil, domain.ErrInvalidToken
	}

	// Check if token is revoked
	revoked, err := s.tokenBlacklist.IsTokenRevoked(claims.TokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to check token revocation: %w", err)
	}
	if revoked {
		return nil, domain.ErrInvalidToken
	}

	// Check if session is revoked
	sessionRevoked, err := s.tokenBlacklist.IsSessionRevoked(claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session revocation: %w", err)
	}
	if sessionRevoked {
		return nil, domain.ErrInvalidToken
	}

	response := &domain.TokenValidationResponse{
		Valid:     true,
		UserID:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt,
		IssuedAt:  claims.IssuedAt,
		TokenID:   claims.TokenID,
	}

	// Include session metadata if requested
	if includeMetadata {
		session, err := s.sessionRepo.FindByID(claims.SessionID)
		if err == nil && session != nil {
			response.SessionMetadata = &domain.SessionMetadata{
				IPAddress:    session.IPAddress,
				UserAgent:    session.UserAgent,
				LastActivity: session.LastActivity,
				Browser:      session.DeviceInfo["browser"],
				OS:           session.DeviceInfo["os"],
				DeviceType:   session.DeviceInfo["device_type"],
				Country:      session.DeviceInfo["country"],
				City:         session.DeviceInfo["city"],
			}
		}
	}

	// Log validation event
	s.LogSecurityEvent(&domain.SecurityEvent{
		Timestamp: time.Now(),
		Event:     "token.validated",
		UserID:    claims.UserID,
		TokenID:   claims.TokenID,
		Reason:    "validation_success",
	})

	return response, nil
}

// RevokeToken revokes tokens based on the request
func (s *AuthService) RevokeToken(request *domain.RevokeTokenRequest) (int, error) {
	count := 0

	if request.RevokeAll {
		// Revoke all user tokens - need to get user ID from one of the tokens
		// This should be called with user ID in context
		return 0, fmt.Errorf("revoke_all requires user_id in context")
	}

	if request.TokenID != "" {
		// Revoke specific token
		// Need to find token expiration
		session, err := s.sessionRepo.FindByTokenID(request.TokenID)
		if err == nil && session != nil {
			if err := s.tokenBlacklist.RevokeToken(request.TokenID, session.ExpiresAt); err != nil {
				return 0, fmt.Errorf("failed to revoke token: %w", err)
			}
			count++

			// Log security event
			s.LogSecurityEvent(&domain.SecurityEvent{
				Timestamp: time.Now(),
				Event:     "token.revoked",
				UserID:    session.UserID,
				TokenID:   request.TokenID,
				Reason:    "token_id_revocation",
			})
		}
	}

	if request.SessionID != "" {
		// Revoke entire session
		if err := s.tokenBlacklist.RevokeSession(request.SessionID); err != nil {
			return count, fmt.Errorf("failed to revoke session: %w", err)
		}

		// Also mark session as inactive in database
		if err := s.sessionRepo.Delete(request.SessionID); err != nil {
			fmt.Printf("failed to delete session: %v\n", err)
		}

		count++

		// Log security event
		session, _ := s.sessionRepo.FindByID(request.SessionID)
		if session != nil {
			s.LogSecurityEvent(&domain.SecurityEvent{
				Timestamp: time.Now(),
				Event:     "token.revoked",
				UserID:    session.UserID,
				TokenID:   session.TokenID,
				Reason:    "session_revocation",
				Metadata: map[string]interface{}{
					"session_id": request.SessionID,
				},
			})
		}
	}

	return count, nil
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (s *AuthService) RevokeAllUserTokens(userID string) error {
	// Revoke all tokens in Redis
	if err := s.tokenBlacklist.RevokeAllUserTokens(userID); err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	// Mark all sessions as inactive
	if err := s.sessionRepo.DeleteByUserID(userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	// Log security event
	s.LogSecurityEvent(&domain.SecurityEvent{
		Timestamp: time.Now(),
		Event:     "token.revoked",
		UserID:    userID,
		Reason:    "revoke_all",
	})

	return nil
}

// GetUserSessions returns all active sessions for a user
func (s *AuthService) GetUserSessions(userID string) ([]*domain.Session, error) {
	return s.sessionRepo.FindByUserID(userID)
}

// DeleteSession deletes a specific session
func (s *AuthService) DeleteSession(sessionID string, userID string) error {
	// Verify session belongs to user
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	if session.UserID != userID {
		return fmt.Errorf("unauthorized: session belongs to different user")
	}

	// Revoke session
	if err := s.tokenBlacklist.RevokeSession(sessionID); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Delete from database
	if err := s.sessionRepo.Delete(sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Log security event
	s.LogSecurityEvent(&domain.SecurityEvent{
		Timestamp: time.Now(),
		Event:     "token.revoked",
		UserID:    userID,
		TokenID:   session.TokenID,
		Reason:    "session_deleted",
		Metadata: map[string]interface{}{
			"session_id": sessionID,
		},
	})

	return nil
}

// LogSecurityEvent logs a security event
func (s *AuthService) LogSecurityEvent(event *domain.SecurityEvent) error {
	if s.securityEventRepo == nil {
		return nil
	}
	return s.securityEventRepo.Create(event)
}

// parseUserAgent parses user agent string into device info
func parseUserAgent(userAgent string) map[string]string {
	info := make(map[string]string)

	// Simple parsing - in production, use a library like ua-parser
	ua := strings.ToLower(userAgent)

	// Detect browser
	if strings.Contains(ua, "chrome") {
		info["browser"] = "Chrome"
	} else if strings.Contains(ua, "safari") {
		info["browser"] = "Safari"
	} else if strings.Contains(ua, "firefox") {
		info["browser"] = "Firefox"
	} else if strings.Contains(ua, "edge") {
		info["browser"] = "Edge"
	} else {
		info["browser"] = "Unknown"
	}

	// Detect OS
	if strings.Contains(ua, "windows") {
		info["os"] = "Windows"
	} else if strings.Contains(ua, "mac") || strings.Contains(ua, "macintosh") {
		info["os"] = "macOS"
	} else if strings.Contains(ua, "linux") {
		info["os"] = "Linux"
	} else if strings.Contains(ua, "android") {
		info["os"] = "Android"
	} else if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		info["os"] = "iOS"
	} else {
		info["os"] = "Unknown"
	}

	// Detect device type
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		info["device_type"] = "mobile"
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		info["device_type"] = "tablet"
	} else {
		info["device_type"] = "desktop"
	}

	return info
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

// checkAccountLockout checks if an account is locked
func (s *AuthService) checkAccountLockout(email string) error {
	attempt, err := s.loginAttemptRepo.GetByEmail(email)
	if err != nil {
		if err == domain.ErrNotFound {
			// No previous attempts, account is not locked
			return nil
		}
		// Log error but don't fail login
		fmt.Printf("failed to check account lockout: %v\n", err)
		return nil
	}

	// Check if account is locked
	if attempt.IsLocked && attempt.UnlockAt != nil {
		if time.Now().Before(*attempt.UnlockAt) {
			// Account is still locked
			s.LogSecurityEvent(&domain.SecurityEvent{
				Timestamp: time.Now(),
				Event:     "login.blocked",
				UserID:    "",
				IPAddress: "",
				UserAgent: "",
				Reason:    "account_locked",
				Metadata: map[string]interface{}{
					"email":     email,
					"unlock_at": attempt.UnlockAt,
				},
			})
			return domain.ErrAccountLocked
		}

		// Lock period has expired, reset the attempt
		s.resetFailedAttempts(email)
	}

	return nil
}

// incrementFailedAttempts increments the failed login attempts count
func (s *AuthService) incrementFailedAttempts(email string) {
	attempt, err := s.loginAttemptRepo.GetByEmail(email)
	if err != nil {
		if err == domain.ErrNotFound {
			// Create new attempt record
			attempt = &domain.LoginAttempt{
				Email:       email,
				FailedCount: 1,
				LastAttempt: time.Now(),
				IsLocked:    false,
			}
			if err := s.loginAttemptRepo.Create(attempt); err != nil {
				fmt.Printf("failed to create login attempt: %v\n", err)
			}
			return
		}
		fmt.Printf("failed to get login attempt: %v\n", err)
		return
	}

	// Increment failed count
	attempt.FailedCount++
	attempt.LastAttempt = time.Now()

	// Lock account after 10 failed attempts
	if attempt.FailedCount >= 10 {
		attempt.IsLocked = true
		unlockTime := time.Now().Add(30 * time.Minute)
		attempt.UnlockAt = &unlockTime

		// Log account lockout
		s.LogSecurityEvent(&domain.SecurityEvent{
			Timestamp: time.Now(),
			Event:     "account.locked",
			UserID:    "",
			IPAddress: "",
			UserAgent: "",
			Reason:    "too_many_failed_attempts",
			Metadata: map[string]interface{}{
				"email":        email,
				"failed_count": attempt.FailedCount,
				"unlock_at":    unlockTime,
			},
		})
	}

	if err := s.loginAttemptRepo.Update(attempt); err != nil {
		fmt.Printf("failed to update login attempt: %v\n", err)
	}
}

// resetFailedAttempts resets the failed login attempts count
func (s *AuthService) resetFailedAttempts(email string) {
	if err := s.loginAttemptRepo.Reset(email); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("failed to reset login attempts: %v\n", err)
	}
}

// validatePassword validates a password
func (s *AuthService) validatePassword(password string) error {
	if password == "" {
		return domain.NewValidationError("password", "password is required")
	}

	if len(password) < 8 {
		return domain.NewValidationError("password", "password must be at least 8 characters")
	}

	// Check for at least one uppercase, one lowercase, one number, and one special character
	hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasDigit := strings.ContainsAny(password, "0123456789")
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`).MatchString(password)

	if !hasUpper || !hasLower || !hasDigit {
		return domain.NewValidationError("password", "password must contain uppercase, lowercase, and digit")
	}

	if !hasSpecial {
		return domain.NewValidationError("password", "password must contain at least one special character (!@#$%^&*()_+-=[]{}...)")
	}

	// Check against common passwords
	if isCommonPassword(password) {
		return domain.NewValidationError("password", "password is too common, please choose a stronger one")
	}

	return nil
}

// Common passwords list (top 100 most common)
var commonPasswords = []string{
	"password", "123456", "12345678", "qwerty", "abc123", "monkey", "1234567", "letmein",
	"trustno1", "dragon", "baseball", "111111", "iloveyou", "master", "sunshine", "ashley",
	"bailey", "passw0rd", "shadow", "123123", "654321", "superman", "qazwsx", "michael",
	"football", "password1", "welcome", "jesus", "ninja", "mustang", "password123", "admin",
	"hello", "charlie", "aa123456", "donald", "password1234", "qwerty123", "1q2w3e4r",
	"welcome1", "monkey1", "dragon1", "princess", "qwerty1", "1qaz2wsx", "password12",
}

// isCommonPassword checks if the password is in the common passwords list
func isCommonPassword(password string) bool {
	lowerPass := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lowerPass == strings.ToLower(common) {
			return true
		}
	}
	return false
}
