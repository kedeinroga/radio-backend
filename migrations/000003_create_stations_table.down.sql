-- ===============================================
-- Rollback 000003: Stations Table
-- ===============================================

DROP TRIGGER IF EXISTS stations_updated_at ON stations;
DROP TRIGGER IF EXISTS stations_search_vector_update_trigger ON stations;
DROP FUNCTION IF EXISTS stations_search_vector_update();
DROP TABLE IF EXISTS stations CASCADE;
