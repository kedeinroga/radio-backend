package jwt

import (
	"crypto/rsa"
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
}

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	UserType string `json:"user_type"`
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
	}, nil
}

// GenerateAccessToken generates an access token for a user
func (tm *TokenManager) GenerateAccessToken(user *domain.User) (string, error) {
	return tm.generateToken(user, tm.accessExpiration)
}

// GenerateRefreshToken generates a refresh token for a user
func (tm *TokenManager) GenerateRefreshToken(user *domain.User) (string, error) {
	return tm.generateToken(user, tm.refreshExpiration)
}

// generateToken generates a JWT token
func (tm *TokenManager) generateToken(user *domain.User, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		Email:    user.Email,
		UserType: user.UserType.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(tm.privateKey)
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
		IssuedAt:  claims.IssuedAt.Time,
		ExpiresAt: claims.ExpiresAt.Time,
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
