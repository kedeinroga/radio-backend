package bcrypt

import (
	"radio-backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher implements domain.PasswordHasher using bcrypt
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new bcrypt password hasher
func NewPasswordHasher(cost int) *PasswordHasher {
	return &PasswordHasher{cost: cost}
}

// Hash hashes a password using bcrypt
func (h *PasswordHasher) Hash(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// Compare compares a hashed password with a plain password
func (h *PasswordHasher) Compare(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}
