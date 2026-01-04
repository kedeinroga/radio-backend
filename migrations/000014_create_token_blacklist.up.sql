-- Migration: Create token blacklist table
-- Purpose: Store invalidated JWT tokens for logout functionality
-- Author: Kedeinroga
-- Date: 4 January 2026

-- =====================================================
-- Token Blacklist Table
-- =====================================================

CREATE TABLE IF NOT EXISTS token_blacklist (
    id BIGSERIAL PRIMARY KEY,
    token_jti VARCHAR(255) NOT NULL UNIQUE, -- JWT ID (jti claim)
    user_id UUID NOT NULL, -- Changed from BIGINT to UUID to match users table
    blacklisted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL, -- Token expiration time
    reason VARCHAR(100) DEFAULT 'logout', -- logout, revoked, expired, etc.
    ip_address INET,
    user_agent TEXT,

    CONSTRAINT fk_token_blacklist_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- =====================================================
-- Indexes
-- =====================================================

-- Index for fast lookup by JTI (most common query)
CREATE INDEX idx_token_blacklist_jti ON token_blacklist(token_jti);

-- Index for user-based queries
CREATE INDEX idx_token_blacklist_user_id ON token_blacklist(user_id);

-- Index for cleanup queries (find expired tokens)
CREATE INDEX idx_token_blacklist_expires_at ON token_blacklist(expires_at);

-- Note: Partial index with NOW() removed as NOW() is not IMMUTABLE
-- Query optimization will rely on the expires_at index
-- The is_token_blacklisted() function handles the expiry check

-- =====================================================
-- Automatic Cleanup Function
-- =====================================================

-- Function to delete expired blacklisted tokens
-- Keeps table size manageable by removing tokens that are already expired
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS TABLE(deleted_count BIGINT) AS $$
DECLARE
    v_deleted_count BIGINT;
BEGIN
    DELETE FROM token_blacklist
    WHERE expires_at < NOW();

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;

    RETURN QUERY SELECT v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Function to check if token is blacklisted
-- =====================================================

CREATE OR REPLACE FUNCTION is_token_blacklisted(p_jti VARCHAR(255))
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM token_blacklist
        WHERE token_jti = p_jti
        AND expires_at > NOW()
    );
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Scheduled Cleanup (pg_cron) - OPTIONAL
-- =====================================================

-- Clean up expired tokens daily at 3:00 AM
-- This prevents the table from growing indefinitely
-- Note: Requires pg_cron extension.
-- Uncomment and manually run if pg_cron is available:
/*
SELECT cron.schedule(
    'cleanup-expired-tokens',
    '0 3 * * *',
    'SELECT cleanup_expired_tokens();'
);
*/

-- =====================================================
-- Comments
-- =====================================================

COMMENT ON TABLE token_blacklist IS 'Stores invalidated JWT tokens for logout and revocation';
COMMENT ON COLUMN token_blacklist.token_jti IS 'JWT ID claim - unique identifier for the token';
COMMENT ON COLUMN token_blacklist.user_id IS 'User who owns the token';
COMMENT ON COLUMN token_blacklist.blacklisted_at IS 'When the token was blacklisted';
COMMENT ON COLUMN token_blacklist.expires_at IS 'When the token naturally expires';
COMMENT ON COLUMN token_blacklist.reason IS 'Why the token was blacklisted: logout, revoked, expired';
COMMENT ON COLUMN token_blacklist.ip_address IS 'IP address from which logout was requested';
COMMENT ON COLUMN token_blacklist.user_agent IS 'User agent from which logout was requested';

COMMENT ON FUNCTION cleanup_expired_tokens() IS 'Removes expired tokens from blacklist to keep table size manageable';
COMMENT ON FUNCTION is_token_blacklisted(VARCHAR) IS 'Fast check if a token JTI is blacklisted and not expired';
