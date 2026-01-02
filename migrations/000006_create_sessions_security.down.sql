-- ===============================================
-- Rollback 000006: Sessions and Security
-- ===============================================

DROP FUNCTION IF EXISTS unlock_expired_accounts();
DROP FUNCTION IF EXISTS cleanup_expired_sessions();
DROP TABLE IF EXISTS login_attempts CASCADE;
DROP TABLE IF EXISTS security_events CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
