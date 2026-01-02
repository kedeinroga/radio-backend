-- ===============================================
-- Rollback 000005: Analytics Tables
-- ===============================================

DROP MATERIALIZED VIEW IF EXISTS mv_station_stats_7d;
DROP TABLE IF EXISTS station_play_stats CASCADE;
DROP TABLE IF EXISTS search_queries CASCADE;
DROP TABLE IF EXISTS request_logs CASCADE;
DROP TABLE IF EXISTS station_plays CASCADE;
