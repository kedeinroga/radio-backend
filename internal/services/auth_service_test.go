package services

import (
	"errors"
	"testing"
	"time"

	"radio-backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

type MockPasswordHasher struct {
	mock.Mock
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordHasher) Compare(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

type MockTokenGenerator struct {
	mock.Mock
}

func (m *MockTokenGenerator) GenerateAccessToken(user *domain.User, sessionID string, ipAddress string, userAgent string) (string, string, error) {
	args := m.Called(user, sessionID, ipAddress, userAgent)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockTokenGenerator) GenerateRefreshToken(user *domain.User, sessionID string) (string, error) {
	args := m.Called(user, sessionID)
	return args.String(0), args.Error(1)
}

type MockTokenValidator struct {
	mock.Mock
}

func (m *MockTokenValidator) ValidateToken(token string) (*domain.TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenClaims), args.Error(1)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(session *domain.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByID(id string) (*domain.Session, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) FindByUserID(userID string) ([]*domain.Session, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) FindByTokenID(tokenID string) (*domain.Session, error) {
	args := m.Called(tokenID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) Update(session *domain.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateLastActivity(sessionID string, lastActivity time.Time) error {
	args := m.Called(sessionID, lastActivity)
	return args.Error(0)
}

func (m *MockSessionRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteByUserID(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpired() error {
	args := m.Called()
	return args.Error(0)
}

type MockSecurityEventRepository struct {
	mock.Mock
}

func (m *MockSecurityEventRepository) Create(event *domain.SecurityEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

type MockTokenBlacklist struct {
	mock.Mock
}

func (m *MockTokenBlacklist) IsTokenRevoked(tokenID string) (bool, error) {
	args := m.Called(tokenID)
	return args.Bool(0), args.Error(1)
}

func (m *MockTokenBlacklist) RevokeToken(tokenID string, expiresAt time.Time) error {
	args := m.Called(tokenID, expiresAt)
	return args.Error(0)
}

func (m *MockTokenBlacklist) RevokeAllUserTokens(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockTokenBlacklist) RevokeSession(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockTokenBlacklist) IsSessionRevoked(sessionID string) (bool, error) {
	args := m.Called(sessionID)
	return args.Bool(0), args.Error(1)
}

// MockLoginAttemptRepository is a mock implementation of domain.LoginAttemptRepository
type MockLoginAttemptRepository struct {
	mock.Mock
}

func (m *MockLoginAttemptRepository) GetByEmail(email string) (*domain.LoginAttempt, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LoginAttempt), args.Error(1)
}

func (m *MockLoginAttemptRepository) Create(attempt *domain.LoginAttempt) error {
	args := m.Called(attempt)
	return args.Error(0)
}

func (m *MockLoginAttemptRepository) Update(attempt *domain.LoginAttempt) error {
	args := m.Called(attempt)
	return args.Error(0)
}

func (m *MockLoginAttemptRepository) Reset(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockLoginAttemptRepository) DeleteExpired() error {
	args := m.Called()
	return args.Error(0)
}

// Helper function to create a complete service with all mocks
func setupAuthService() (*AuthService, *MockUserRepository, *MockSessionRepository, *MockSecurityEventRepository, *MockLoginAttemptRepository, *MockPasswordHasher, *MockTokenGenerator, *MockTokenValidator, *MockTokenBlacklist) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockSecurityEventRepo := new(MockSecurityEventRepository)
	mockLoginAttemptRepo := new(MockLoginAttemptRepository)
	mockHasher := new(MockPasswordHasher)
	mockTokenGen := new(MockTokenGenerator)
	mockTokenVal := new(MockTokenValidator)
	mockBlacklist := new(MockTokenBlacklist)

	// Mock the dummy hash generation in NewAuthService
	mockHasher.On("Hash", "dummy_password_for_timing_attack_prevention_12345").Return("$2a$12$dummy_hash", nil)

	service := NewAuthService(mockUserRepo, mockSessionRepo, mockSecurityEventRepo, mockLoginAttemptRepo, mockHasher, mockTokenGen, mockTokenVal, mockBlacklist)
	return service, mockUserRepo, mockSessionRepo, mockSecurityEventRepo, mockLoginAttemptRepo, mockHasher, mockTokenGen, mockTokenVal, mockBlacklist
}

// Tests
func TestAuthService_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		service, mockUserRepo, _, _, _, mockHasher, _, _, _ := setupAuthService()

		email := "test@example.com"
		password := "Password123!@#"
		hashedPassword := "hashed_password"

		mockUserRepo.On("FindByEmail", email).Return(nil, domain.ErrUserNotFound)
		mockHasher.On("Hash", password).Return(hashedPassword, nil)
		mockUserRepo.On("Create", mock.AnythingOfType("*domain.User")).Return(nil)

		user, err := service.Register(email, password)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, hashedPassword, user.PasswordHash)
		assert.Equal(t, domain.UserTypeGuest, user.UserType)
		mockUserRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})

	t.Run("registration with existing email", func(t *testing.T) {
		service, mockUserRepo, _, _, _, _, _, _, _ := setupAuthService()

		email := "existing@example.com"
		password := "Password123!@#"

		existingUser := &domain.User{
			ID:    "user-1",
			Email: email,
		}

		mockUserRepo.On("FindByEmail", email).Return(existingUser, nil)

		user, err := service.Register(email, password)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, domain.ErrUserAlreadyExists, err)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_Login_Basic(t *testing.T) {
	t.Run("login with non-existent user", func(t *testing.T) {
		service, mockUserRepo, _, mockSecurityEventRepo, mockLoginAttemptRepo, mockHasher, _, _, _ := setupAuthService()

		email := "notfound@example.com"
		password := "Password123!@#"

		mockLoginAttemptRepo.On("GetByEmail", email).Return(nil, domain.ErrNotFound)
		mockUserRepo.On("FindByEmail", email).Return(nil, domain.ErrUserNotFound)
		mockHasher.On("Compare", mock.Anything, password).Return(domain.ErrInvalidCredentials)
		mockLoginAttemptRepo.On("Create", mock.AnythingOfType("*domain.LoginAttempt")).Return(nil)
		mockSecurityEventRepo.On("Create", mock.AnythingOfType("*domain.SecurityEvent")).Return(nil)

		_, _, _, _, _, err := service.Login(email, password, "127.0.0.1", "Test User Agent")

		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidCredentials, err)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("login with invalid credentials", func(t *testing.T) {
		service, mockUserRepo, _, mockSecurityEventRepo, mockLoginAttemptRepo, mockHasher, _, _, _ := setupAuthService()

		email := "test@example.com"
		password := "WrongPassword123!@#"
		hashedPassword := "hashed_password"

		user := &domain.User{
			ID:           "user-1",
			Email:        email,
			PasswordHash: hashedPassword,
		}

		mockLoginAttemptRepo.On("GetByEmail", email).Return(nil, domain.ErrNotFound)
		mockUserRepo.On("FindByEmail", email).Return(user, nil)
		mockHasher.On("Compare", hashedPassword, password).Return(errors.New("password mismatch"))
		mockLoginAttemptRepo.On("Create", mock.AnythingOfType("*domain.LoginAttempt")).Return(nil)
		mockSecurityEventRepo.On("Create", mock.AnythingOfType("*domain.SecurityEvent")).Return(nil)

		_, _, _, _, _, err := service.Login(email, password, "127.0.0.1", "Test User Agent")

		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidCredentials, err)
		mockUserRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})
}
