-- Eliminar vistas materializadas
DROP MATERIALIZED VIEW IF EXISTS top_countries_for_seo;
DROP MATERIALIZED VIEW IF EXISTS top_tags_for_seo;

-- Eliminar índices
DROP INDEX IF EXISTS idx_country_stats_updated;
DROP INDEX IF EXISTS idx_tag_stats_updated;
DROP INDEX IF EXISTS idx_country_stats_count;
DROP INDEX IF EXISTS idx_tag_stats_active_count;
DROP INDEX IF EXISTS idx_tag_stats_station_count;

-- Eliminar tablas
DROP TABLE IF EXISTS seo_country_stats;
DROP TABLE IF EXISTS seo_tag_stats;
