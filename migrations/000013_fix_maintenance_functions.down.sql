-- Rollback: Fix maintenance functions
-- Reverts to original versions

-- Revertir get_refresh_statistics
CREATE OR REPLACE FUNCTION get_refresh_statistics(
    days_back INTEGER DEFAULT 7
)
RETURNS TABLE(
    view_name TEXT,
    total_refreshes BIGINT,
    successful_refreshes BIGINT,
    failed_refreshes BIGINT,
    avg_duration_ms NUMERIC,
    max_duration_ms BIGINT,
    min_duration_ms BIGINT,
    last_refresh TIMESTAMPTZ,
    last_status TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        l.view_name,
        COUNT(*) as total_refreshes,
        COUNT(*) FILTER (WHERE l.status LIKE 'SUCCESS%') as successful_refreshes,
        COUNT(*) FILTER (WHERE l.status = 'ERROR') as failed_refreshes,
        ROUND(AVG(l.duration_ms), 2) as avg_duration_ms,
        MAX(l.duration_ms) as max_duration_ms,
        MIN(l.duration_ms) as min_duration_ms,
        MAX(l.refresh_completed_at) as last_refresh,
        (SELECT status FROM materialized_view_refresh_log
         WHERE view_name = l.view_name
         ORDER BY refresh_completed_at DESC LIMIT 1) as last_status
    FROM materialized_view_refresh_log l
    WHERE l.refresh_started_at > NOW() - (days_back || ' days')::INTERVAL
    GROUP BY l.view_name
    ORDER BY l.view_name;
END;
$$ LANGUAGE plpgsql;

-- Revertir refresh_all_views_with_logging (a la versión original de migration 000012)
-- Nota: Esta es una reversión simplificada. Ver migration 000012 para la versión completa original.
CREATE OR REPLACE FUNCTION refresh_all_views_with_logging()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT
) AS $$
BEGIN
    RETURN QUERY SELECT * FROM refresh_all_materialized_views();
END;
$$ LANGUAGE plpgsql;

-- Revertir refresh_all_seo_views
CREATE OR REPLACE FUNCTION refresh_all_seo_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
BEGIN
    RETURN QUERY SELECT * FROM refresh_materialized_view_safe('mv_top_tags_seo');
    RETURN QUERY SELECT * FROM refresh_materialized_view_safe('mv_top_countries_seo');
END;
$$ LANGUAGE plpgsql;

-- Revertir refresh_all_analytics_views
CREATE OR REPLACE FUNCTION refresh_all_analytics_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
BEGIN
    RETURN QUERY SELECT * FROM refresh_materialized_view_safe('mv_station_stats_7d');
END;
$$ LANGUAGE plpgsql;

-- Revertir refresh_all_materialized_views
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
BEGIN
    RETURN QUERY SELECT * FROM refresh_all_seo_views();
    RETURN QUERY SELECT * FROM refresh_all_analytics_views();
END;
$$ LANGUAGE plpgsql;
