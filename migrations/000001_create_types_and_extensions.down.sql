-- ===============================================
-- Rollback 000001: Types and Extensions
-- ===============================================

DROP FUNCTION IF EXISTS update_updated_at() CASCADE;
DROP TYPE IF EXISTS user_type_enum CASCADE;
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS "uuid-ossp";
