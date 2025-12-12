package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserType_String(t *testing.T) {
	tests := []struct {
		name     string
		userType UserType
		want     string
	}{
		{
			name:     "guest user type",
			userType: UserTypeGuest,
			want:     "guest",
		},
		{
			name:     "premium user type",
			userType: UserTypePremium,
			want:     "premium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.userType.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		userType UserType
		want     bool
	}{
		{
			name:     "guest is valid",
			userType: UserTypeGuest,
			want:     true,
		},
		{
			name:     "premium is valid",
			userType: UserTypePremium,
			want:     true,
		},
		{
			name:     "invalid user type",
			userType: UserType("invalid"),
			want:     false,
		},
		{
			name:     "empty user type",
			userType: UserType(""),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.userType.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUser_IsPremium(t *testing.T) {
	tests := []struct {
		name string
		user *User
		want bool
	}{
		{
			name: "premium user returns true",
			user: &User{
				ID:       "user-1",
				Email:    "premium@example.com",
				UserType: UserTypePremium,
			},
			want: true,
		},
		{
			name: "guest user returns false",
			user: &User{
				ID:       "user-2",
				Email:    "guest@example.com",
				UserType: UserTypeGuest,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.IsPremium()
			assert.Equal(t, tt.want, got)
		})
	}
}
