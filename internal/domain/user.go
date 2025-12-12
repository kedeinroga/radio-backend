package domain

import "time"

// UserType represents the type of user in the system
type UserType string

const (
	UserTypeGuest   UserType = "guest"
	UserTypePremium UserType = "premium"
)

// String returns the string representation of UserType
func (ut UserType) String() string {
	return string(ut)
}

// IsValid checks if the user type is valid
func (ut UserType) IsValid() bool {
	return ut == UserTypeGuest || ut == UserTypePremium
}

// User represents a user in the system
type User struct {
	ID           string
	Email        string
	PasswordHash string
	UserType     UserType
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsPremium checks if the user has premium access
func (u *User) IsPremium() bool {
	return u.UserType == UserTypePremium
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(user *User) error
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	Update(user *User) error
	Delete(id string) error
}
