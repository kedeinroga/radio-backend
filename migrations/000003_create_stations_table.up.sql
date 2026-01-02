-- ===============================================
-- Migración 000003: Stations Table
-- ===============================================
-- Tabla de estaciones optimizada con full-text search y campos desnormalizados

CREATE TABLE stations (
    id VARCHAR(255) PRIMARY KEY, -- External ID from RadioBrowser or similar
    name VARCHAR(500) NOT NULL,
    stream_url TEXT NOT NULL,
    stream_url_resolved TEXT,
    image_url TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    country VARCHAR(100),
    country_code CHAR(2), -- ISO 3166-1 alpha-2
    votes INTEGER NOT NULL DEFAULT 0,
    
    -- Campos desnormalizados para performance (actualizados por triggers/jobs)
    play_count INTEGER NOT NULL DEFAULT 0,
    unique_listeners INTEGER NOT NULL DEFAULT 0,
    last_played_at TIMESTAMPTZ,
    
    is_premium_only BOOLEAN NOT NULL DEFAULT false,
    
    -- Cache metadata
    source VARCHAR(20) NOT NULL DEFAULT 'radio_browser',
    last_synced_at TIMESTAMPTZ,
    sync_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Full-text search vector
    search_vector TSVECTOR,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices optimizados con parciales y compuestos
CREATE INDEX idx_stations_active_votes ON stations(votes DESC) 
    WHERE is_active = true;

CREATE INDEX idx_stations_country_votes ON stations(country, votes DESC) 
    WHERE is_active = true;

CREATE INDEX idx_stations_country_code_votes ON stations(country_code, votes DESC) 
    WHERE is_active = true AND country_code IS NOT NULL;

CREATE INDEX idx_stations_tags_gin ON stations USING GIN(tags);

CREATE INDEX idx_stations_search_vector_gin ON stations USING GIN(search_vector);

CREATE INDEX idx_stations_play_count ON stations(play_count DESC) 
    WHERE is_active = true;

CREATE INDEX idx_stations_last_played ON stations(last_played_at DESC NULLS LAST) 
    WHERE is_active = true;

CREATE INDEX idx_stations_is_active ON stations(is_active, updated_at DESC);

-- Covering index para búsquedas populares (Index-Only Scan)
CREATE INDEX idx_stations_active_votes_covering ON stations(is_active, votes DESC)
    INCLUDE (id, name, stream_url, image_url, country, country_code, tags);

-- Trigger para actualizar search_vector automáticamente
CREATE OR REPLACE FUNCTION stations_search_vector_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('simple', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.country, '')), 'B') ||
        setweight(to_tsvector('simple', array_to_string(NEW.tags, ' ')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stations_search_vector_update_trigger
    BEFORE INSERT OR UPDATE ON stations
    FOR EACH ROW
    EXECUTE FUNCTION stations_search_vector_update();

-- Trigger para updated_at
CREATE TRIGGER stations_updated_at
    BEFORE UPDATE ON stations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Comentarios
COMMENT ON TABLE stations IS 'Radio stations cache with full-text search';
COMMENT ON COLUMN stations.search_vector IS 'Auto-generated tsvector for full-text search';
COMMENT ON COLUMN stations.play_count IS 'Denormalized play count - updated by analytics jobs';
COMMENT ON COLUMN stations.unique_listeners IS 'Denormalized unique listeners count';
COMMENT ON COLUMN stations.country_code IS 'ISO 3166-1 alpha-2 country code for efficient joins';
