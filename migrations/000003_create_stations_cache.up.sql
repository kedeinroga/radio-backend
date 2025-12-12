-- Enable trigram extension for fuzzy search (must be first)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create stations cache table
CREATE TABLE IF NOT EXISTS stations (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    stream_url TEXT NOT NULL,
    stream_url_resolved TEXT,
    image_url TEXT,
    tags TEXT[],
    country VARCHAR(100),
    votes INTEGER DEFAULT 0,
    is_premium_only BOOLEAN DEFAULT false,
    
    -- Cache metadata
    source VARCHAR(20) DEFAULT 'radio_browser',
    last_synced_at TIMESTAMP,
    sync_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for fast queries
CREATE INDEX idx_stations_name_trgm ON stations USING gin(name gin_trgm_ops);
CREATE INDEX idx_stations_country ON stations(country) WHERE is_active = true;
CREATE INDEX idx_stations_votes ON stations(votes DESC) WHERE is_active = true;
CREATE INDEX idx_stations_tags ON stations USING gin(tags);
CREATE INDEX idx_stations_last_synced ON stations(last_synced_at);
CREATE INDEX idx_stations_is_active ON stations(is_active);
