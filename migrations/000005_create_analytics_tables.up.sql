-- ===============================================
-- Migración 000005: Analytics Tables (Partitioned)
-- ===============================================
-- Tablas de analytics particionadas por mes para mejor performance

-- ============================================
-- 1. STATION PLAYS (Particionada)
-- ============================================
CREATE TABLE station_plays (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    station_id VARCHAR(255) NOT NULL,
    user_id UUID,
    user_type user_type_enum NOT NULL,
    duration_ms BIGINT NOT NULL,
    ip_address INET,
    country_code CHAR(2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Crear particiones para 2026 (6 meses)
CREATE TABLE station_plays_2026_01 PARTITION OF station_plays
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE station_plays_2026_02 PARTITION OF station_plays
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE station_plays_2026_03 PARTITION OF station_plays
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE station_plays_2026_04 PARTITION OF station_plays
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE station_plays_2026_05 PARTITION OF station_plays
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE station_plays_2026_06 PARTITION OF station_plays
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Índices en tabla padre (se propagan a particiones)
CREATE INDEX idx_station_plays_station_created ON station_plays(station_id, created_at DESC);
CREATE INDEX idx_station_plays_user_created ON station_plays(user_id, created_at DESC) 
    WHERE user_id IS NOT NULL;
CREATE INDEX idx_station_plays_created ON station_plays(created_at DESC);
CREATE INDEX idx_station_plays_country ON station_plays(country_code, created_at DESC) 
    WHERE country_code IS NOT NULL;

-- ============================================
-- 2. REQUEST LOGS (Particionada)
-- ============================================
CREATE TABLE request_logs (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(255) NOT NULL,
    user_id UUID,
    user_type user_type_enum NOT NULL,
    status_code SMALLINT NOT NULL,
    duration_ms BIGINT NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Crear particiones para 2026
CREATE TABLE request_logs_2026_01 PARTITION OF request_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE request_logs_2026_02 PARTITION OF request_logs
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE request_logs_2026_03 PARTITION OF request_logs
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE request_logs_2026_04 PARTITION OF request_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE request_logs_2026_05 PARTITION OF request_logs
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE request_logs_2026_06 PARTITION OF request_logs
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Índices
CREATE INDEX idx_request_logs_user_created ON request_logs(user_id, created_at DESC) 
    WHERE user_id IS NOT NULL;
CREATE INDEX idx_request_logs_path_created ON request_logs(path, created_at DESC);
CREATE INDEX idx_request_logs_status_created ON request_logs(status_code, created_at DESC) 
    WHERE status_code >= 400;
CREATE INDEX idx_request_logs_created ON request_logs(created_at DESC);

-- ============================================
-- 3. SEARCH QUERIES (Particionada)
-- ============================================
CREATE TABLE search_queries (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    query VARCHAR(255) NOT NULL,
    query_normalized VARCHAR(255),
    results_count INTEGER NOT NULL,
    user_id UUID,
    user_type user_type_enum NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Crear particiones para 2026
CREATE TABLE search_queries_2026_01 PARTITION OF search_queries
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE search_queries_2026_02 PARTITION OF search_queries
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE search_queries_2026_03 PARTITION OF search_queries
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE search_queries_2026_04 PARTITION OF search_queries
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE search_queries_2026_05 PARTITION OF search_queries
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE search_queries_2026_06 PARTITION OF search_queries
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Índices
CREATE INDEX idx_search_queries_normalized ON search_queries(query_normalized, created_at DESC);
CREATE INDEX idx_search_queries_user_created ON search_queries(user_id, created_at DESC) 
    WHERE user_id IS NOT NULL;
CREATE INDEX idx_search_queries_created ON search_queries(created_at DESC);

-- ============================================
-- 4. STATION PLAY STATS (Agregaciones diarias)
-- ============================================
CREATE TABLE station_play_stats (
    station_id VARCHAR(255) NOT NULL,
    stats_date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_plays INTEGER NOT NULL DEFAULT 0,
    total_duration_ms BIGINT NOT NULL DEFAULT 0,
    unique_users INTEGER NOT NULL DEFAULT 0,
    unique_ips INTEGER NOT NULL DEFAULT 0,
    countries JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (station_id, stats_date)
);

-- Índices
CREATE INDEX idx_station_play_stats_date ON station_play_stats(stats_date DESC);
CREATE INDEX idx_station_play_stats_plays ON station_play_stats(stats_date DESC, total_plays DESC);
CREATE INDEX idx_station_play_stats_station ON station_play_stats(station_id, stats_date DESC);

-- Vista materializada para últimos 7 días
CREATE MATERIALIZED VIEW mv_station_stats_7d AS
SELECT 
    station_id,
    SUM(total_plays) as plays_7d,
    SUM(total_duration_ms) as duration_7d,
    SUM(unique_users) as listeners_7d,
    COUNT(DISTINCT stats_date) as active_days,
    MAX(stats_date) as last_active
FROM station_play_stats
WHERE stats_date >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY station_id;

CREATE UNIQUE INDEX ON mv_station_stats_7d(station_id);
CREATE INDEX ON mv_station_stats_7d(plays_7d DESC);

-- Comentarios
COMMENT ON TABLE station_plays IS 'Station play events - partitioned by month for performance';
COMMENT ON TABLE request_logs IS 'HTTP request logs - partitioned by month';
COMMENT ON TABLE search_queries IS 'Search query logs - partitioned by month';
COMMENT ON TABLE station_play_stats IS 'Daily aggregated station play statistics';
COMMENT ON MATERIALIZED VIEW mv_station_stats_7d IS 'Last 7 days station stats - refresh hourly';
