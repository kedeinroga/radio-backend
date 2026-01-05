-- Create stream_sessions table for audio proxy tracking
CREATE TABLE IF NOT EXISTS stream_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    station_id TEXT NOT NULL, -- No foreign key porque stations.id es VARCHAR
    ad_id UUID REFERENCES advertisements(id) ON DELETE SET NULL,

    -- Token de acceso
    stream_token TEXT NOT NULL,
    token_expires_at TIMESTAMPTZ NOT NULL,

    -- Timestamps
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Métricas
    bytes_streamed BIGINT DEFAULT 0,
    listening_duration INTERVAL DEFAULT '0 seconds',

    -- Estado
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'abandoned', 'error')),

    -- Metadata
    user_agent TEXT,
    ip_address INET,
    country_code CHAR(2),

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices para performance
CREATE INDEX idx_stream_sessions_user_id ON stream_sessions(user_id);
CREATE INDEX idx_stream_sessions_station_id ON stream_sessions(station_id);
CREATE INDEX idx_stream_sessions_ad_id ON stream_sessions(ad_id) WHERE ad_id IS NOT NULL;
CREATE INDEX idx_stream_sessions_status ON stream_sessions(status);
CREATE INDEX idx_stream_sessions_started_at ON stream_sessions(started_at DESC);
CREATE INDEX idx_stream_sessions_active ON stream_sessions(user_id, status) WHERE status = 'active';
CREATE INDEX idx_stream_sessions_token ON stream_sessions(stream_token);

-- Vista materializada para analytics de streaming
CREATE MATERIALIZED VIEW stream_analytics AS
SELECT
    DATE_TRUNC('hour', started_at) as hour,
    station_id,
    COUNT(*) as total_sessions,
    COUNT(DISTINCT user_id) as unique_listeners,
    AVG(EXTRACT(EPOCH FROM listening_duration)) as avg_duration_seconds,
    SUM(bytes_streamed) as total_bytes_streamed,
    COUNT(*) FILTER (WHERE ad_id IS NOT NULL) as sessions_with_ads,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_sessions,
    COUNT(*) FILTER (WHERE status = 'abandoned') as abandoned_sessions
FROM stream_sessions
WHERE started_at > NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('hour', started_at), station_id;

-- Índice único para refresh concurrente
CREATE UNIQUE INDEX idx_stream_analytics_hour_station ON stream_analytics(hour, station_id);

-- Función para refresh automático de analytics
CREATE OR REPLACE FUNCTION refresh_stream_analytics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY stream_analytics;
END;
$$ LANGUAGE plpgsql;

-- Comentarios para documentación
COMMENT ON TABLE stream_sessions IS 'Tracks audio streaming sessions for analytics and ad validation';
COMMENT ON COLUMN stream_sessions.stream_token IS 'JWT token for accessing the stream proxy';
COMMENT ON COLUMN stream_sessions.bytes_streamed IS 'Total bytes transferred during the session';
COMMENT ON COLUMN stream_sessions.listening_duration IS 'Actual listening time (excludes buffering/pauses)';
COMMENT ON COLUMN stream_sessions.status IS 'Session status: active, completed, abandoned, or error';
