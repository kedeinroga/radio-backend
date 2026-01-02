-- ===============================================
-- Rollback 000009: Search Cache
-- ===============================================

DROP FUNCTION IF EXISTS cleanup_expired_search_cache();
DROP TABLE IF EXISTS station_search_cache CASCADE;
