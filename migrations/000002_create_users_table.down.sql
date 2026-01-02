-- ===============================================
-- Rollback 000002: Users Table
-- ===============================================

DROP TRIGGER IF EXISTS users_updated_at ON users;
DROP TABLE IF EXISTS users CASCADE;
