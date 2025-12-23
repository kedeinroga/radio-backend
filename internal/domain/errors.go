package domain

// DomainError represents a business logic error
type DomainError struct {
	Code    string
	Message string
	Field   string
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}

// Common domain errors
var (
	ErrUserNotFound       = &DomainError{Code: "USER_NOT_FOUND", Message: "user not found"}
	ErrUserAlreadyExists  = &DomainError{Code: "USER_ALREADY_EXISTS", Message: "user already exists"}
	ErrInvalidCredentials = &DomainError{Code: "INVALID_CREDENTIALS", Message: "invalid email or password"}
	ErrInvalidToken       = &DomainError{Code: "INVALID_TOKEN", Message: "invalid or expired token"}
	ErrUnauthorized       = &DomainError{Code: "UNAUTHORIZED", Message: "unauthorized access"}
	ErrInvalidUserType    = &DomainError{Code: "INVALID_USER_TYPE", Message: "invalid user type"}
	ErrStationNotFound    = &DomainError{Code: "STATION_NOT_FOUND", Message: "station not found"}
	ErrInvalidQuery       = &DomainError{Code: "INVALID_QUERY", Message: "invalid search query"}

	// Favorite errors
	ErrFavoriteAlreadyExists = &DomainError{Code: "FAVORITE_ALREADY_EXISTS", Message: "station is already in favorites"}
	ErrFavoriteNotFound      = &DomainError{Code: "FAVORITE_NOT_FOUND", Message: "favorite not found"}
)

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *DomainError {
	return &DomainError{
		Code:    "VALIDATION_ERROR",
		Message: message,
		Field:   field,
	}
}
