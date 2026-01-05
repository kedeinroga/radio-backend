//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"

	"radio-backend/internal/domain"
	"radio-backend/internal/repositories/postgres"
	"radio-backend/tests/fixtures"
	"radio-backend/tests/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserRepository_Create tests user creation in database
func TestUserRepository_Create(t *testing.T) {
	// Setup
	db := helpers.SetupTestDB(t)
	defer helpers.TeardownTestDB(t, db)

	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := fixtures.CreateTestUser()

	// Execute
	err := repo.Create(user)
	require.NoError(t, err)

	// Verify
	assert.NotEmpty(t, user.ID, "User ID should be set after creation")

	// Verify in database
	retrieved, err := repo.FindByEmail(user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.Email, retrieved.Email)
	assert.Equal(t, user.UserType, retrieved.UserType)
}

// TestUserRepository_FindByEmail tests finding user by email
func TestUserRepository_FindByEmail(t *testing.T) {
	// Setup
	db := helpers.SetupTestDB(t)
	defer helpers.TeardownTestDB(t, db)

	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := fixtures.CreateTestUser()
	err := repo.Create(user)
	require.NoError(t, err)

	// Test: Find existing user
	t.Run("find existing user", func(t *testing.T) {
		found, err := repo.FindByEmail(user.Email)
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.ID, found.ID)
	})

	// Test: User not found
	t.Run("user not found", func(t *testing.T) {
		_, err := repo.FindByEmail("nonexistent@example.com")
		assert.Error(t, err)
		assert.Equal(t, domain.ErrUserNotFound, err)
	})
}

// TestUserRepository_Update tests updating user
func TestUserRepository_Update(t *testing.T) {
	// Setup
	db := helpers.SetupTestDB(t)
	defer helpers.TeardownTestDB(t, db)

	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := fixtures.CreateTestUser()
	err := repo.Create(user)
	require.NoError(t, err)

	// Update user
	user.UserType = domain.UserTypePremium
	err = repo.Update(user)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.UserTypePremium, updated.UserType)
}

// TestUserRepository_Delete tests deleting user
func TestUserRepository_Delete(t *testing.T) {
	// Setup
	db := helpers.SetupTestDB(t)
	defer helpers.TeardownTestDB(t, db)

	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := fixtures.CreateTestUser()
	err := repo.Create(user)
	require.NoError(t, err)

	// Delete user
	err = repo.Delete(user.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.FindByID(user.ID)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrUserNotFound, err)
}
