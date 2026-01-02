-- ===============================================
-- Rollback 000007: SEO Tables
-- ===============================================

DROP MATERIALIZED VIEW IF EXISTS mv_top_countries_seo;
DROP MATERIALIZED VIEW IF EXISTS mv_top_tags_seo;
DROP TABLE IF EXISTS seo_country_stats CASCADE;
DROP TABLE IF EXISTS seo_tag_stats CASCADE;
