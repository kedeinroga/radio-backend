-- ===============================================
-- Migración 000007: SEO Tables
-- ===============================================
-- Tablas de estadísticas SEO y vistas materializadas

-- ============================================
-- 1. SEO TAG STATS
-- ============================================
CREATE TABLE seo_tag_stats (
    tag VARCHAR(255) PRIMARY KEY,
    slug VARCHAR(255) NOT NULL UNIQUE,
    station_count INTEGER NOT NULL DEFAULT 0 CHECK (station_count >= 0),
    active_count INTEGER NOT NULL DEFAULT 0 CHECK (active_count >= 0),
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices
CREATE INDEX idx_seo_tag_stats_active ON seo_tag_stats(active_count DESC);
CREATE INDEX idx_seo_tag_stats_total ON seo_tag_stats(station_count DESC);
CREATE INDEX idx_seo_tag_stats_updated ON seo_tag_stats(last_updated DESC);

-- ============================================
-- 2. SEO COUNTRY STATS
-- ============================================
CREATE TABLE seo_country_stats (
    country_code CHAR(2) PRIMARY KEY, -- ISO 3166-1 alpha-2
    country_name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    station_count INTEGER NOT NULL DEFAULT 0 CHECK (station_count >= 0),
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices
CREATE INDEX idx_seo_country_stats_count ON seo_country_stats(station_count DESC);
CREATE INDEX idx_seo_country_stats_updated ON seo_country_stats(last_updated DESC);

-- ============================================
-- 3. VISTA MATERIALIZADA: TOP TAGS
-- ============================================
CREATE MATERIALIZED VIEW mv_top_tags_seo AS
SELECT
    tag,
    slug,
    station_count,
    active_count,
    last_updated
FROM seo_tag_stats
WHERE active_count > 0
ORDER BY station_count DESC, active_count DESC
LIMIT 100;

CREATE UNIQUE INDEX ON mv_top_tags_seo(tag);
CREATE INDEX ON mv_top_tags_seo(station_count DESC);

-- ============================================
-- 4. VISTA MATERIALIZADA: TOP COUNTRIES
-- ============================================
CREATE MATERIALIZED VIEW mv_top_countries_seo AS
SELECT
    country_code,
    country_name,
    slug,
    station_count,
    last_updated
FROM seo_country_stats
WHERE station_count > 0
ORDER BY station_count DESC
LIMIT 50;

CREATE UNIQUE INDEX ON mv_top_countries_seo(country_code);
CREATE INDEX ON mv_top_countries_seo(station_count DESC);

-- Comentarios
COMMENT ON TABLE seo_tag_stats IS 'Aggregated tag statistics for SEO sitemap generation';
COMMENT ON TABLE seo_country_stats IS 'Aggregated country statistics for SEO';
COMMENT ON MATERIALIZED VIEW mv_top_tags_seo IS 'Top 100 tags by station count - refresh every 6 hours';
COMMENT ON MATERIALIZED VIEW mv_top_countries_seo IS 'Top 50 countries by station count - refresh every 6 hours';
