-- Migration: Drop token blacklist table
-- Purpose: Rollback for 000014_create_token_blacklist.up.sql
-- Author: Kedeinroga
-- Date: 4 January 2026

-- Remove pg_cron job
SELECT cron.unschedule('cleanup-expired-tokens');

-- Drop functions
DROP FUNCTION IF EXISTS is_token_blacklisted(VARCHAR);
DROP FUNCTION IF EXISTS cleanup_expired_tokens();

-- Drop indexes (will be dropped automatically with table, but explicit for clarity)
DROP INDEX IF EXISTS idx_token_blacklist_jti_expires;
DROP INDEX IF EXISTS idx_token_blacklist_expires_at;
DROP INDEX IF EXISTS idx_token_blacklist_user_id;
DROP INDEX IF EXISTS idx_token_blacklist_jti;

-- Drop table
DROP TABLE IF EXISTS token_blacklist;
