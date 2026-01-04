package domain

import "time"

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	UserID    string
	UserType  UserType
	Email     string
	Role      string // admin | user | guest
	IssuedAt  time.Time
	ExpiresAt time.Time
	TokenID   string // jti - Unique token identifier
	SessionID string // Unique session identifier
	Issuer    string // iss - Token issuer
}

// IsExpired checks if the token has expired
func (tc *TokenClaims) IsExpired() bool {
	return time.Now().After(tc.ExpiresAt)
}

// SessionMetadata holds information about a user session
type SessionMetadata struct {
	IPAddress    string
	UserAgent    string
	LastActivity time.Time
	Browser      string
	OS           string
	DeviceType   string // desktop | mobile | tablet
	Country      string
	City         string
}

// Session represents a user session
type Session struct {
	ID           string
	UserID       string
	SessionID    string
	TokenID      string
	DeviceInfo   map[string]string
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
	LastActivity time.Time
	ExpiresAt    time.Time
	IsActive     bool
}

// TokenValidationResponse represents the response from token validation
type TokenValidationResponse struct {
	Valid           bool
	UserID          string
	Email           string
	Role            string
	ExpiresAt       time.Time
	IssuedAt        time.Time
	TokenID         string
	SessionMetadata *SessionMetadata
}

// RevokeTokenRequest represents a request to revoke tokens
type RevokeTokenRequest struct {
	TokenID   string
	SessionID string
	RevokeAll bool
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	TokenID   string // JTI from the current token
	UserID    string // User performing logout
	IPAddress string
	UserAgent string
	Reason    string // "user_logout", "admin_revoke", "security_concern"
}

// LogoutResponse represents a logout response
type LogoutResponse struct {
	Success       bool      `json:"success"`
	Message       string    `json:"message"`
	TokenID       string    `json:"token_id,omitempty"`
	BlacklistedAt time.Time `json:"blacklisted_at,omitempty"`
}

// TokenBlacklistEntry represents a blacklisted token
type TokenBlacklistEntry struct {
	ID            int64     `json:"id"`
	TokenJTI      string    `json:"token_jti"`
	UserID        string    `json:"user_id"` // Changed from int64 to string (UUID)
	BlacklistedAt time.Time `json:"blacklisted_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Reason        string    `json:"reason"`
	IPAddress     string    `json:"ip_address,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"`
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Timestamp time.Time
	Event     string // token.issued, token.validated, token.revoked, session.suspicious
	UserID    string
	TokenID   string
	IPAddress string
	UserAgent string
	Reason    string
	Metadata  map[string]interface{}
}

// TokenGenerator defines the interface for generating authentication tokens
type TokenGenerator interface {
	GenerateAccessToken(user *User, sessionID string, ipAddress string, userAgent string) (tokenString string, tokenID string, err error)
	GenerateRefreshToken(user *User, sessionID string) (string, error)
}

// TokenValidator defines the interface for validating authentication tokens
type TokenValidator interface {
	ValidateToken(token string) (*TokenClaims, error)
}

// TokenBlacklist defines the interface for token revocation
type TokenBlacklist interface {
	IsTokenRevoked(tokenID string) (bool, error)
	RevokeToken(tokenID string, expiresAt time.Time) error
	RevokeAllUserTokens(userID string) error
	RevokeSession(sessionID string) error
	IsSessionRevoked(sessionID string) (bool, error)
	// New methods for logout functionality
	BlacklistToken(entry *TokenBlacklistEntry) error
	IsTokenBlacklisted(tokenJTI string) (bool, error)
	GetUserBlacklistedTokens(userID string) ([]*TokenBlacklistEntry, error) // Changed from int64 to string
	CleanupExpiredTokens() (int64, error)
}

// SessionRepository defines the interface for session management
type SessionRepository interface {
	Create(session *Session) error
	FindByID(sessionID string) (*Session, error)
	FindByUserID(userID string) ([]*Session, error)
	FindByTokenID(tokenID string) (*Session, error)
	UpdateLastActivity(sessionID string, lastActivity time.Time) error
	Delete(sessionID string) error
	DeleteByUserID(userID string) error
	DeleteExpired() error
}

// LoginAttempt represents a login attempt record for account lockout
type LoginAttempt struct {
	Email       string
	FailedCount int
	LastAttempt time.Time
	IsLocked    bool
	UnlockAt    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LoginAttemptRepository defines the interface for login attempt management
type LoginAttemptRepository interface {
	GetByEmail(email string) (*LoginAttempt, error)
	Create(attempt *LoginAttempt) error
	Update(attempt *LoginAttempt) error
	Reset(email string) error
	DeleteExpired() error
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	Register(email, password string) (*User, error)
	Login(email, password string, ipAddress string, userAgent string) (accessToken, refreshToken, sessionID, tokenID string, expiresAt time.Time, err error)
	RefreshToken(refreshToken string, ipAddress string, userAgent string) (accessToken, sessionID, tokenID string, expiresAt time.Time, err error)
	ValidateToken(token string, includeMetadata bool) (*TokenValidationResponse, error)
	RevokeToken(request *RevokeTokenRequest) (int, error)
	GetUserSessions(userID string) ([]*Session, error)
	DeleteSession(sessionID string, userID string) error
	LogSecurityEvent(event *SecurityEvent) error
	// New logout method
	Logout(request *LogoutRequest) (*LogoutResponse, error)
}

// PasswordHasher defines the interface for password hashing operations
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}
