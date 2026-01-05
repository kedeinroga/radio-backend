-- Drop stream analytics
DROP MATERIALIZED VIEW IF EXISTS stream_analytics CASCADE;
DROP FUNCTION IF EXISTS refresh_stream_analytics() CASCADE;

-- Drop stream_sessions table
DROP TABLE IF EXISTS stream_sessions CASCADE;
