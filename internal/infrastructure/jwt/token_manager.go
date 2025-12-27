package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"radio-backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

// TokenManager handles JWT token generation and validation
type TokenManager struct {
	privateKey        *rsa.PrivateKey
	publicKey         *rsa.PublicKey
	accessExpiration  time.Duration
	refreshExpiration time.Duration
	issuer            string
}

// Claims represents JWT claims with RFC 7519 standard fields
type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	UserType  string `json:"user_type"`
	Role      string `json:"role"` // admin | user | guest
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

// NewTokenManager creates a new JWT token manager
func NewTokenManager(privateKeyPath, publicKeyPath string, accessExp, refreshExp time.Duration) (*TokenManager, error) {
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	publicKey, err := loadPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return &TokenManager{
		privateKey:        privateKey,
		publicKey:         publicKey,
		accessExpiration:  accessExp,
		refreshExpiration: refreshExp,
		issuer:            "radioapp-backend",
	}, nil
}

// GenerateAccessToken generates an access token for a user with session info
func (tm *TokenManager) GenerateAccessToken(user *domain.User, sessionID string, ipAddress string, userAgent string) (string, string, error) {
	return tm.generateToken(user, sessionID, tm.accessExpiration)
}

// GenerateRefreshToken generates a refresh token for a user
func (tm *TokenManager) GenerateRefreshToken(user *domain.User, sessionID string) (string, error) {
	token, _, err := tm.generateToken(user, sessionID, tm.refreshExpiration)
	return token, err
}

// generateUniqueID generates a unique ID for JWT tokens
func generateUniqueID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generateToken generates a JWT token with RFC 7519 compliant claims
func (tm *TokenManager) generateToken(user *domain.User, sessionID string, expiration time.Duration) (string, string, error) {
	now := time.Now()
	jti, err := generateUniqueID()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token ID: %w", err)
	}

	// Map UserType to role
	role := "user"
	switch user.UserType {
	case domain.UserTypeAdmin:
		role = "admin"
	case domain.UserTypeGuest:
		role = "guest"
	}

	claims := Claims{
		UserID:    user.ID,
		Email:     user.Email,
		UserType:  user.UserType.String(),
		Role:      role,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,                              // sub - standard claim
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)), // exp - standard claim
			IssuedAt:  jwt.NewNumericDate(now),              // iat - standard claim
			NotBefore: jwt.NewNumericDate(now),              // nbf - standard claim
			ID:        jti,                                  // jti - standard claim
			Issuer:    tm.issuer,                            // iss - standard claim
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(tm.privateKey)
	if err != nil {
		return "", "", err
	}

	return tokenString, jti, nil
}

// ValidateToken validates a JWT token and returns the claims
func (tm *TokenManager) ValidateToken(tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.publicKey, nil
	})

	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	userType := domain.UserType(claims.UserType)
	if !userType.IsValid() {
		return nil, domain.ErrInvalidUserType
	}

	return &domain.TokenClaims{
		UserID:    claims.UserID,
		UserType:  userType,
		Email:     claims.Email,
		Role:      claims.Role,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiresAt: claims.ExpiresAt.Time,
		TokenID:   claims.ID,
		SessionID: claims.SessionID,
		Issuer:    claims.Issuer,
	}, nil
}

// loadPrivateKey loads an RSA private key from a file
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return jwt.ParseRSAPrivateKeyFromPEM(keyData)
}

// loadPublicKey loads an RSA public key from a file
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return jwt.ParseRSAPublicKeyFromPEM(keyData)
}
