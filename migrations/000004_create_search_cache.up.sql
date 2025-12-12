-- Create search cache table
CREATE TABLE IF NOT EXISTS station_search_cache (
    id SERIAL PRIMARY KEY,
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    query_params JSONB NOT NULL,
    station_ids TEXT[] NOT NULL,
    result_count INTEGER NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_search_cache_hash ON station_search_cache(query_hash);
CREATE INDEX idx_search_cache_expires ON station_search_cache(expires_at);
