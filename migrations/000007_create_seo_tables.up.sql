-- Tabla de agregación de tags (actualizada por job/trigger)
CREATE TABLE IF NOT EXISTS seo_tag_stats (
    tag VARCHAR(255) PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    station_count INT DEFAULT 0 CHECK (station_count >= 0),
    active_count INT DEFAULT 0 CHECK (active_count >= 0),
    last_updated TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Tabla de agregación de países
CREATE TABLE IF NOT EXISTS seo_country_stats (
    country_name VARCHAR(100) PRIMARY KEY,
    slug VARCHAR(100) NOT NULL,
    station_count INT DEFAULT 0 CHECK (station_count >= 0),
    last_updated TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_tag_stats_station_count ON seo_tag_stats(station_count DESC);
CREATE INDEX IF NOT EXISTS idx_tag_stats_active_count ON seo_tag_stats(active_count DESC);
CREATE INDEX IF NOT EXISTS idx_country_stats_count ON seo_country_stats(station_count DESC);
CREATE INDEX IF NOT EXISTS idx_tag_stats_updated ON seo_tag_stats(last_updated DESC);
CREATE INDEX IF NOT EXISTS idx_country_stats_updated ON seo_country_stats(last_updated DESC);

-- Vista materializada para top 100 tags (refresh cada hora)
CREATE MATERIALIZED VIEW IF NOT EXISTS top_tags_for_seo AS
SELECT
    tag,
    slug,
    station_count,
    active_count,
    last_updated
FROM seo_tag_stats
WHERE active_count > 0  -- Solo tags con estaciones activas
ORDER BY station_count DESC, active_count DESC
LIMIT 100;

-- Índice único en la vista materializada
CREATE UNIQUE INDEX IF NOT EXISTS idx_top_tags_tag ON top_tags_for_seo (tag);

-- Vista materializada para top 50 países
CREATE MATERIALIZED VIEW IF NOT EXISTS top_countries_for_seo AS
SELECT
    country_name,
    slug,
    station_count,
    last_updated
FROM seo_country_stats
WHERE station_count > 0  -- Solo países con estaciones
ORDER BY station_count DESC
LIMIT 50;

-- Índice único en la vista materializada
CREATE UNIQUE INDEX IF NOT EXISTS idx_top_countries_name ON top_countries_for_seo (country_name);

-- Comentarios para documentación
COMMENT ON TABLE seo_tag_stats IS 'Estadísticas agregadas de tags/géneros para SEO sitemap generation';
COMMENT ON TABLE seo_country_stats IS 'Estadísticas agregadas de países para SEO sitemap generation';
COMMENT ON MATERIALIZED VIEW top_tags_for_seo IS 'Top 100 tags con más estaciones activas - refresh cada 6 horas';
COMMENT ON MATERIALIZED VIEW top_countries_for_seo IS 'Top 50 países con más estaciones - refresh cada 6 horas';
