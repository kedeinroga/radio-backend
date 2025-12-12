package services

import (
	"errors"
	"testing"

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

func (m *MockTokenGenerator) GenerateAccessToken(user *domain.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *MockTokenGenerator) GenerateRefreshToken(user *domain.User) (string, error) {
	args := m.Called(user)
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

// Tests
func TestAuthService_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		email := "test@example.com"
		password := "Password123"
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
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		email := "existing@example.com"
		password := "Password123"

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

	t.Run("registration with invalid email", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		invalidEmails := []string{"", "invalid", "invalid@", "@example.com", "invalid@.com"}

		for _, email := range invalidEmails {
			user, err := service.Register(email, "Password123")
			assert.Error(t, err)
			assert.Nil(t, user)
		}
	})

	t.Run("registration with weak password", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		weakPasswords := []string{
			"",          // empty
			"short",     // too short
			"password",  // no uppercase or digit
			"PASSWORD",  // no lowercase or digit
			"Password",  // no digit
			"password1", // no uppercase
			"PASSWORD1", // no lowercase
		}

		for _, password := range weakPasswords {
			user, err := service.Register("test@example.com", password)
			assert.Error(t, err)
			assert.Nil(t, user)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		email := "test@example.com"
		password := "Password123"
		hashedPassword := "hashed_password"
		accessToken := "access_token"
		refreshToken := "refresh_token"

		user := &domain.User{
			ID:           "user-1",
			Email:        email,
			PasswordHash: hashedPassword,
			UserType:     domain.UserTypeGuest,
		}

		mockUserRepo.On("FindByEmail", email).Return(user, nil)
		mockHasher.On("Compare", hashedPassword, password).Return(nil)
		mockTokenGen.On("GenerateAccessToken", user).Return(accessToken, nil)
		mockTokenGen.On("GenerateRefreshToken", user).Return(refreshToken, nil)

		gotAccess, gotRefresh, err := service.Login(email, password)

		assert.NoError(t, err)
		assert.Equal(t, accessToken, gotAccess)
		assert.Equal(t, refreshToken, gotRefresh)
		mockUserRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
		mockTokenGen.AssertExpectations(t)
	})

	t.Run("login with invalid credentials", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		email := "test@example.com"
		password := "WrongPassword123"
		hashedPassword := "hashed_password"

		user := &domain.User{
			ID:           "user-1",
			Email:        email,
			PasswordHash: hashedPassword,
		}

		mockUserRepo.On("FindByEmail", email).Return(user, nil)
		mockHasher.On("Compare", hashedPassword, password).Return(errors.New("password mismatch"))

		gotAccess, gotRefresh, err := service.Login(email, password)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidCredentials, err)
		assert.Empty(t, gotAccess)
		assert.Empty(t, gotRefresh)
		mockUserRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})

	t.Run("login with non-existent user", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		email := "nonexistent@example.com"
		password := "Password123"

		mockUserRepo.On("FindByEmail", email).Return(nil, domain.ErrUserNotFound)

		gotAccess, gotRefresh, err := service.Login(email, password)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidCredentials, err)
		assert.Empty(t, gotAccess)
		assert.Empty(t, gotRefresh)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("successful token refresh", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		refreshToken := "valid_refresh_token"
		accessToken := "new_access_token"
		userID := "user-1"

		claims := &domain.TokenClaims{
			UserID: userID,
		}

		user := &domain.User{
			ID:       userID,
			Email:    "test@example.com",
			UserType: domain.UserTypeGuest,
		}

		mockTokenVal.On("ValidateToken", refreshToken).Return(claims, nil)
		mockUserRepo.On("FindByID", userID).Return(user, nil)
		mockTokenGen.On("GenerateAccessToken", user).Return(accessToken, nil)

		gotAccess, err := service.RefreshToken(refreshToken)

		assert.NoError(t, err)
		assert.Equal(t, accessToken, gotAccess)
		mockTokenVal.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockTokenGen.AssertExpectations(t)
	})

	t.Run("refresh with invalid token", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		refreshToken := "invalid_refresh_token"

		mockTokenVal.On("ValidateToken", refreshToken).Return(nil, domain.ErrInvalidToken)

		gotAccess, err := service.RefreshToken(refreshToken)

		assert.Error(t, err)
		assert.Empty(t, gotAccess)
		mockTokenVal.AssertExpectations(t)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	t.Run("validate valid token", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		token := "valid_token"
		claims := &domain.TokenClaims{
			UserID: "user-1",
			Email:  "test@example.com",
		}

		mockTokenVal.On("ValidateToken", token).Return(claims, nil)

		gotClaims, err := service.ValidateToken(token)

		assert.NoError(t, err)
		assert.Equal(t, claims, gotClaims)
		mockTokenVal.AssertExpectations(t)
	})

	t.Run("validate invalid token", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockHasher := new(MockPasswordHasher)
		mockTokenGen := new(MockTokenGenerator)
		mockTokenVal := new(MockTokenValidator)

		service := NewAuthService(mockUserRepo, mockHasher, mockTokenGen, mockTokenVal)

		token := "invalid_token"

		mockTokenVal.On("ValidateToken", token).Return(nil, domain.ErrInvalidToken)

		gotClaims, err := service.ValidateToken(token)

		assert.Error(t, err)
		assert.Nil(t, gotClaims)
		mockTokenVal.AssertExpectations(t)
	})
}
