-- ===============================================
-- Migración 000009: Search Cache
-- ===============================================
-- Tabla de caché de búsquedas de estaciones

CREATE TABLE station_search_cache (
    id SERIAL PRIMARY KEY,
    query_hash CHAR(64) UNIQUE NOT NULL, -- SHA256 hash
    query_params JSONB NOT NULL,
    station_ids TEXT[] NOT NULL,
    result_count INTEGER NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices optimizados
CREATE INDEX idx_search_cache_hash ON station_search_cache(query_hash);
CREATE INDEX idx_search_cache_expires ON station_search_cache(expires_at);

-- Función de limpieza automática
CREATE OR REPLACE FUNCTION cleanup_expired_search_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM station_search_cache 
    WHERE expires_at < NOW() - INTERVAL '1 hour';
END;
$$ LANGUAGE plpgsql;

-- Comentarios
COMMENT ON TABLE station_search_cache IS 'Cache for station search results';
COMMENT ON COLUMN station_search_cache.query_hash IS 'SHA256 hash of query parameters for fast lookup';
COMMENT ON FUNCTION cleanup_expired_search_cache() IS 'Removes expired cache entries older than 1 hour';
