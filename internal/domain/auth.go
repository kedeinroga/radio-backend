package domain

import "time"

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	UserID    string
	UserType  UserType
	Email     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// IsExpired checks if the token has expired
func (tc *TokenClaims) IsExpired() bool {
	return time.Now().After(tc.ExpiresAt)
}

// TokenGenerator defines the interface for generating authentication tokens
type TokenGenerator interface {
	GenerateAccessToken(user *User) (string, error)
	GenerateRefreshToken(user *User) (string, error)
}

// TokenValidator defines the interface for validating authentication tokens
type TokenValidator interface {
	ValidateToken(token string) (*TokenClaims, error)
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	Register(email, password string) (*User, error)
	Login(email, password string) (accessToken, refreshToken string, err error)
	RefreshToken(refreshToken string) (accessToken string, err error)
	ValidateToken(token string) (*TokenClaims, error)
}

// PasswordHasher defines the interface for password hashing operations
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}
