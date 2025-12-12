-- Create request_logs table
CREATE TABLE IF NOT EXISTS request_logs (
    id VARCHAR(36) PRIMARY KEY,
    request_id VARCHAR(36) NOT NULL,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(255) NOT NULL,
    user_id VARCHAR(36),
    user_type VARCHAR(20) NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms BIGINT NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create station_plays table
CREATE TABLE IF NOT EXISTS station_plays (
    id VARCHAR(36) PRIMARY KEY,
    station_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(36),
    user_type VARCHAR(20) NOT NULL,
    duration_ms BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create search_queries table
CREATE TABLE IF NOT EXISTS search_queries (
    id VARCHAR(36) PRIMARY KEY,
    query VARCHAR(255) NOT NULL,
    results_count INTEGER NOT NULL,
    user_id VARCHAR(36),
    user_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for analytics queries
CREATE INDEX idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX idx_request_logs_user_id ON request_logs(user_id);
CREATE INDEX idx_request_logs_path ON request_logs(path);

CREATE INDEX idx_station_plays_created_at ON station_plays(created_at);
CREATE INDEX idx_station_plays_station_id ON station_plays(station_id);
CREATE INDEX idx_station_plays_user_id ON station_plays(user_id);

CREATE INDEX idx_search_queries_created_at ON search_queries(created_at);
CREATE INDEX idx_search_queries_query ON search_queries(query);
CREATE INDEX idx_search_queries_user_id ON search_queries(user_id);
