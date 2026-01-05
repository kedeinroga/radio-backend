package fixtures

import (
	"fmt"
	"time"

	"radio-backend/internal/domain"

	"github.com/google/uuid"
)

// UserFactory crea usuarios de prueba con el patrón builder
type UserFactory struct {
	email        string
	passwordHash string
	userType     domain.UserType
}

// NewUserFactory crea una nueva fábrica de usuarios
func NewUserFactory() *UserFactory {
	return &UserFactory{
		email:        "test@example.com",
		passwordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // "password"
		userType:     domain.UserTypeGuest,
	}
}

// WithEmail establece el email del usuario
func (f *UserFactory) WithEmail(email string) *UserFactory {
	f.email = email
	return f
}

// WithPasswordHash establece el hash de la contraseña
func (f *UserFactory) WithPasswordHash(passwordHash string) *UserFactory {
	f.passwordHash = passwordHash
	return f
}

// WithUserType establece el tipo de usuario
func (f *UserFactory) WithUserType(userType domain.UserType) *UserFactory {
	f.userType = userType
	return f
}

// Build construye el usuario
func (f *UserFactory) Build() *domain.User {
	now := time.Now()
	return &domain.User{
		ID:           uuid.New().String(),
		Email:        f.email,
		PasswordHash: f.passwordHash,
		UserType:     f.userType,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateTestUser crea un usuario de prueba con valores por defecto
func CreateTestUser() *domain.User {
	return NewUserFactory().Build()
}

// CreateTestUsers crea múltiples usuarios de prueba
func CreateTestUsers(count int) []*domain.User {
	users := make([]*domain.User, count)
	for i := 0; i < count; i++ {
		users[i] = NewUserFactory().
			WithEmail(fmt.Sprintf("user%d@example.com", i)).
			Build()
	}
	return users
}

// CreateAdminUser crea un usuario administrador
func CreateAdminUser() *domain.User {
	return NewUserFactory().
		WithEmail("admin@example.com").
		WithUserType(domain.UserTypeAdmin).
		Build()
}
