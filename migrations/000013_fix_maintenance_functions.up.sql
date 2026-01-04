-- Migration: Fix maintenance functions
-- Fixes ambiguous column references and adds safety checks for materialized views

-- ============================================
-- 1. FIX: get_refresh_statistics - Resolver ambigüedad en duration_ms
-- ============================================
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
        (SELECT sub.status FROM materialized_view_refresh_log sub
         WHERE sub.view_name = l.view_name
         ORDER BY sub.refresh_completed_at DESC LIMIT 1) as last_status
    FROM materialized_view_refresh_log l
    WHERE l.refresh_started_at > NOW() - (days_back || ' days')::INTERVAL
    GROUP BY l.view_name
    ORDER BY l.view_name;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_refresh_statistics(INTEGER) IS
'Obtiene estadísticas de refrescos de vistas materializadas en los últimos N días. FIX: Resuelve ambigüedad en subconsulta.';

-- ============================================
-- 2. FIX: refresh_all_views_with_logging - Agregar verificación de existencia
-- ============================================
CREATE OR REPLACE FUNCTION refresh_all_views_with_logging()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT
) AS $$
DECLARE
    start_time TIMESTAMPTZ;
    end_time TIMESTAMPTZ;
    duration BIGINT;
    affected_rows BIGINT;
    current_status TEXT;
    error_msg TEXT;
    view_exists BOOLEAN;
BEGIN
    -- Verificar si existen las vistas materializadas
    SELECT EXISTS (
        SELECT 1 FROM pg_matviews WHERE schemaname = 'public' AND matviewname IN ('mv_top_tags_seo', 'mv_top_countries_seo', 'mv_station_stats_7d')
    ) INTO view_exists;

    -- Si no existen vistas, retornar mensaje informativo
    IF NOT view_exists THEN
        RETURN QUERY SELECT
            'N/A'::TEXT as view_name,
            0::BIGINT as duration_ms,
            0::BIGINT as rows_affected,
            'INFO: No materialized views found. Views will be created automatically when base tables have data.'::TEXT as status;
        RETURN;
    END IF;

    -- Refrescar vistas SEO
    FOR view_name IN
        SELECT matviewname::TEXT
        FROM pg_matviews
        WHERE schemaname = 'public'
        AND matviewname IN ('mv_top_tags_seo', 'mv_top_countries_seo')
    LOOP
        BEGIN
            start_time := clock_timestamp();

            EXECUTE format('REFRESH MATERIALIZED VIEW CONCURRENTLY %I', view_name);

            end_time := clock_timestamp();
            duration := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;

            EXECUTE format('SELECT count(*) FROM %I', view_name) INTO affected_rows;
            current_status := 'SUCCESS';
            error_msg := NULL;

        EXCEPTION WHEN OTHERS THEN
            end_time := clock_timestamp();
            duration := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;
            affected_rows := 0;
            current_status := 'ERROR';
            error_msg := SQLERRM;
        END;

        -- Insertar log
        INSERT INTO materialized_view_refresh_log (
            view_name, refresh_started_at, refresh_completed_at,
            duration_ms, rows_affected, status, error_message
        ) VALUES (
            view_name, start_time, end_time,
            duration, affected_rows, current_status, error_msg
        );

        RETURN QUERY SELECT view_name, duration, affected_rows, current_status;
    END LOOP;

    -- Refrescar vistas Analytics
    FOR view_name IN
        SELECT matviewname::TEXT
        FROM pg_matviews
        WHERE schemaname = 'public'
        AND matviewname = 'mv_station_stats_7d'
    LOOP
        BEGIN
            start_time := clock_timestamp();

            EXECUTE format('REFRESH MATERIALIZED VIEW CONCURRENTLY %I', view_name);

            end_time := clock_timestamp();
            duration := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;

            EXECUTE format('SELECT count(*) FROM %I', view_name) INTO affected_rows;
            current_status := 'SUCCESS';
            error_msg := NULL;

        EXCEPTION WHEN OTHERS THEN
            end_time := clock_timestamp();
            duration := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;
            affected_rows := 0;
            current_status := 'ERROR';
            error_msg := SQLERRM;
        END;

        -- Insertar log
        INSERT INTO materialized_view_refresh_log (
            view_name, refresh_started_at, refresh_completed_at,
            duration_ms, rows_affected, status, error_message
        ) VALUES (
            view_name, start_time, end_time,
            duration, affected_rows, current_status, error_msg
        );

        RETURN QUERY SELECT view_name, duration, affected_rows, current_status;
    END LOOP;

    RETURN;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_all_views_with_logging() IS
'Refresca todas las vistas materializadas con logging. FIX: Verifica existencia antes de refrescar.';

-- ============================================
-- 3. FIX: refresh_all_seo_views - Verificar existencia
-- ============================================
CREATE OR REPLACE FUNCTION refresh_all_seo_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
DECLARE
    view_exists BOOLEAN;
BEGIN
    -- Verificar si existen las vistas
    SELECT EXISTS (
        SELECT 1 FROM pg_matviews
        WHERE schemaname = 'public'
        AND matviewname IN ('mv_top_tags_seo', 'mv_top_countries_seo')
    ) INTO view_exists;

    IF NOT view_exists THEN
        RETURN QUERY SELECT
            'N/A'::TEXT,
            0::BIGINT,
            0::BIGINT,
            'INFO'::TEXT,
            'No SEO materialized views found. They will be created when data is available.'::TEXT;
        RETURN;
    END IF;

    -- Refrescar mv_top_tags_seo
    IF EXISTS (SELECT 1 FROM pg_matviews WHERE schemaname = 'public' AND matviewname = 'mv_top_tags_seo') THEN
        RETURN QUERY SELECT * FROM refresh_materialized_view_safe('mv_top_tags_seo');
    END IF;

    -- Refrescar mv_top_countries_seo
    IF EXISTS (SELECT 1 FROM pg_matviews WHERE schemaname = 'public' AND matviewname = 'mv_top_countries_seo') THEN
        RETURN QUERY SELECT * FROM refresh_materialized_view_safe('mv_top_countries_seo');
    END IF;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_all_seo_views() IS
'Refresca las vistas materializadas de SEO de manera segura. FIX: Verifica existencia.';

-- ============================================
-- 4. FIX: refresh_all_analytics_views - Verificar existencia
-- ============================================
CREATE OR REPLACE FUNCTION refresh_all_analytics_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
DECLARE
    view_exists BOOLEAN;
BEGIN
    -- Verificar si existe la vista
    SELECT EXISTS (
        SELECT 1 FROM pg_matviews
        WHERE schemaname = 'public'
        AND matviewname = 'mv_station_stats_7d'
    ) INTO view_exists;

    IF NOT view_exists THEN
        RETURN QUERY SELECT
            'N/A'::TEXT,
            0::BIGINT,
            0::BIGINT,
            'INFO'::TEXT,
            'No analytics materialized views found. They will be created when data is available.'::TEXT;
        RETURN;
    END IF;

    -- Refrescar mv_station_stats_7d
    RETURN QUERY SELECT * FROM refresh_materialized_view_safe('mv_station_stats_7d');
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_all_analytics_views() IS
'Refresca las vistas materializadas de analytics de manera segura. FIX: Verifica existencia.';

-- ============================================
-- 5. FIX: refresh_all_materialized_views - Verificar existencia
-- ============================================
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
BEGIN
    -- Refrescar todas las vistas SEO
    RETURN QUERY SELECT * FROM refresh_all_seo_views();

    -- Refrescar todas las vistas Analytics
    RETURN QUERY SELECT * FROM refresh_all_analytics_views();
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_all_materialized_views() IS
'Refresca todas las vistas materializadas del sistema. FIX: Usa funciones mejoradas con verificación.';
