package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"radio-backend/internal/domain"
)

// TokenBlacklistRepository implements domain.TokenBlacklist
type TokenBlacklistRepository struct {
	db *sql.DB
}

// NewTokenBlacklistRepository creates a new token blacklist repository
func NewTokenBlacklistRepository(db *sql.DB) *TokenBlacklistRepository {
	return &TokenBlacklistRepository{db: db}
}

// BlacklistToken adds a token to the blacklist
func (r *TokenBlacklistRepository) BlacklistToken(entry *domain.TokenBlacklistEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO token_blacklist (
			token_jti, user_id, blacklisted_at, expires_at,
			reason, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (token_jti) DO NOTHING
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		entry.TokenJTI,
		entry.UserID,
		entry.BlacklistedAt,
		entry.ExpiresAt,
		entry.Reason,
		entry.IPAddress,
		entry.UserAgent,
	).Scan(&entry.ID)

	if err == sql.ErrNoRows {
		// Token already blacklisted, not an error
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// IsTokenBlacklisted checks if a token JTI is in the blacklist
func (r *TokenBlacklistRepository) IsTokenBlacklisted(tokenJTI string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use the optimized SQL function
	var isBlacklisted bool
	query := `SELECT is_token_blacklisted($1)`

	err := r.db.QueryRowContext(ctx, query, tokenJTI).Scan(&isBlacklisted)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return isBlacklisted, nil
}

// GetUserBlacklistedTokens retrieves all blacklisted tokens for a user
func (r *TokenBlacklistRepository) GetUserBlacklistedTokens(userID string) ([]*domain.TokenBlacklistEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token_jti, user_id, blacklisted_at, expires_at,
		       reason, ip_address, user_agent
		FROM token_blacklist
		WHERE user_id = $1
		AND expires_at > NOW()
		ORDER BY blacklisted_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user blacklisted tokens: %w", err)
	}
	defer rows.Close()

	var entries []*domain.TokenBlacklistEntry
	for rows.Next() {
		var entry domain.TokenBlacklistEntry
		var ipAddress, userAgent sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.TokenJTI,
			&entry.UserID,
			&entry.BlacklistedAt,
			&entry.ExpiresAt,
			&entry.Reason,
			&ipAddress,
			&userAgent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan blacklisted token: %w", err)
		}

		if ipAddress.Valid {
			entry.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			entry.UserAgent = userAgent.String
		}

		entries = append(entries, &entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating blacklisted tokens: %w", err)
	}

	return entries, nil
}

// CleanupExpiredTokens removes expired tokens from the blacklist
func (r *TokenBlacklistRepository) CleanupExpiredTokens() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the optimized SQL function
	var deletedCount int64
	query := `SELECT cleanup_expired_tokens()`

	err := r.db.QueryRowContext(ctx, query).Scan(&deletedCount)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	return deletedCount, nil
}

// IsTokenRevoked checks if a token is revoked (legacy method, uses IsTokenBlacklisted)
func (r *TokenBlacklistRepository) IsTokenRevoked(tokenID string) (bool, error) {
	return r.IsTokenBlacklisted(tokenID)
}

// RevokeToken revokes a single token (legacy method)
func (r *TokenBlacklistRepository) RevokeToken(tokenID string, expiresAt time.Time) error {
	entry := &domain.TokenBlacklistEntry{
		TokenJTI:      tokenID,
		UserID:        "00000000-0000-0000-0000-000000000000", // Unknown UUID for legacy calls
		BlacklistedAt: time.Now(),
		ExpiresAt:     expiresAt,
		Reason:        "revoked",
	}
	return r.BlacklistToken(entry)
}

// RevokeAllUserTokens revokes all active tokens for a user
func (r *TokenBlacklistRepository) RevokeAllUserTokens(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This would require additional logic to track all active tokens
	// For now, we'll mark sessions as revoked
	query := `
		UPDATE sessions
		SET is_active = false
		WHERE user_id = $1 AND is_active = true
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// RevokeSession revokes all tokens associated with a session
func (r *TokenBlacklistRepository) RevokeSession(sessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE sessions
		SET is_active = false
		WHERE session_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

// IsSessionRevoked checks if a session is revoked
func (r *TokenBlacklistRepository) IsSessionRevoked(sessionID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var isActive bool
	query := `SELECT is_active FROM sessions WHERE session_id = $1`

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&isActive)
	if err == sql.ErrNoRows {
		// Session doesn't exist, consider it revoked
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check session revocation: %w", err)
	}

	// If not active, it's revoked
	return !isActive, nil
}
