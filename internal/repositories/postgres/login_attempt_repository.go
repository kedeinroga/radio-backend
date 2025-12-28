package postgres

import (
	"database/sql"
	"time"

	"radio-backend/internal/domain"
)

// LoginAttemptRepository implements domain.LoginAttemptRepository
type LoginAttemptRepository struct {
	db *sql.DB
}

// NewLoginAttemptRepository creates a new login attempt repository
func NewLoginAttemptRepository(db *sql.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db}
}

// GetByEmail retrieves a login attempt by email
func (r *LoginAttemptRepository) GetByEmail(email string) (*domain.LoginAttempt, error) {
	query := `
		SELECT email, failed_count, last_attempt, is_locked, unlock_at, created_at, updated_at
		FROM login_attempts
		WHERE email = $1
	`

	attempt := &domain.LoginAttempt{}
	var unlockAt sql.NullTime

	err := r.db.QueryRow(query, email).Scan(
		&attempt.Email,
		&attempt.FailedCount,
		&attempt.LastAttempt,
		&attempt.IsLocked,
		&unlockAt,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	if unlockAt.Valid {
		attempt.UnlockAt = &unlockAt.Time
	}

	return attempt, nil
}

// Create creates a new login attempt record
func (r *LoginAttemptRepository) Create(attempt *domain.LoginAttempt) error {
	query := `
		INSERT INTO login_attempts (email, failed_count, last_attempt, is_locked, unlock_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	attempt.CreatedAt = now
	attempt.UpdatedAt = now

	_, err := r.db.Exec(query,
		attempt.Email,
		attempt.FailedCount,
		attempt.LastAttempt,
		attempt.IsLocked,
		attempt.UnlockAt,
		attempt.CreatedAt,
		attempt.UpdatedAt,
	)

	return err
}

// Update updates an existing login attempt record
func (r *LoginAttemptRepository) Update(attempt *domain.LoginAttempt) error {
	query := `
		UPDATE login_attempts
		SET failed_count = $2,
		    last_attempt = $3,
		    is_locked = $4,
		    unlock_at = $5,
		    updated_at = $6
		WHERE email = $1
	`

	attempt.UpdatedAt = time.Now()

	_, err := r.db.Exec(query,
		attempt.Email,
		attempt.FailedCount,
		attempt.LastAttempt,
		attempt.IsLocked,
		attempt.UnlockAt,
		attempt.UpdatedAt,
	)

	return err
}

// Reset resets the failed count and unlocks the account
func (r *LoginAttemptRepository) Reset(email string) error {
	query := `
		UPDATE login_attempts
		SET failed_count = 0,
		    is_locked = FALSE,
		    unlock_at = NULL,
		    updated_at = $2
		WHERE email = $1
	`

	_, err := r.db.Exec(query, email, time.Now())
	return err
}

// DeleteExpired deletes expired login attempt records (older than 30 days)
func (r *LoginAttemptRepository) DeleteExpired() error {
	query := `
		DELETE FROM login_attempts
		WHERE updated_at < $1
	`

	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	_, err := r.db.Exec(query, thirtyDaysAgo)
	return err
}
