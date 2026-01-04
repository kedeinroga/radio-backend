-- ===============================================
-- Migración 000012: Automated Materialized View Refresh
-- ===============================================
-- Refresh automático de vistas materializadas

-- ============================================
-- 1. FUNCIÓN: Refresh de vista materializada con logging
-- ============================================
CREATE OR REPLACE FUNCTION refresh_materialized_view_safe(
    view_name TEXT,
    use_concurrent BOOLEAN DEFAULT TRUE
)
RETURNS TABLE(
    view TEXT,
    refresh_started_at TIMESTAMPTZ,
    refresh_completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
DECLARE
    start_time TIMESTAMPTZ;
    end_time TIMESTAMPTZ;
    row_count BIGINT;
    refresh_command TEXT;
BEGIN
    start_time := clock_timestamp();

    BEGIN
        -- Construir comando de refresh
        IF use_concurrent THEN
            refresh_command := format('REFRESH MATERIALIZED VIEW CONCURRENTLY %I', view_name);
        ELSE
            refresh_command := format('REFRESH MATERIALIZED VIEW %I', view_name);
        END IF;

        -- Ejecutar refresh
        EXECUTE refresh_command;

        end_time := clock_timestamp();

        -- Obtener conteo de filas
        EXECUTE format('SELECT count(*) FROM %I', view_name) INTO row_count;

        -- Retornar resultado exitoso
        view := view_name;
        refresh_started_at := start_time;
        refresh_completed_at := end_time;
        duration_ms := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;
        rows_affected := row_count;
        status := 'SUCCESS';
        error_message := NULL;
        RETURN NEXT;

        RAISE NOTICE 'Refreshed materialized view % in % ms (% rows)',
            view_name, duration_ms, row_count;

    EXCEPTION WHEN OTHERS THEN
        end_time := clock_timestamp();

        -- Si falla CONCURRENT, intentar sin CONCURRENT
        IF use_concurrent AND SQLSTATE = '55000' THEN
            RAISE NOTICE 'CONCURRENT refresh failed for %, retrying without CONCURRENT', view_name;

            BEGIN
                EXECUTE format('REFRESH MATERIALIZED VIEW %I', view_name);
                end_time := clock_timestamp();
                EXECUTE format('SELECT count(*) FROM %I', view_name) INTO row_count;

                view := view_name;
                refresh_started_at := start_time;
                refresh_completed_at := end_time;
                duration_ms := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;
                rows_affected := row_count;
                status := 'SUCCESS_NON_CONCURRENT';
                error_message := NULL;
                RETURN NEXT;
                RETURN;
            EXCEPTION WHEN OTHERS THEN
                -- Si también falla, reportar error
                view := view_name;
                refresh_started_at := start_time;
                refresh_completed_at := end_time;
                duration_ms := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;
                rows_affected := 0;
                status := 'ERROR';
                error_message := SQLERRM;
                RETURN NEXT;
                RETURN;
            END;
        ELSE
            -- Reportar error
            view := view_name;
            refresh_started_at := start_time;
            refresh_completed_at := end_time;
            duration_ms := EXTRACT(EPOCH FROM (end_time - start_time)) * 1000;
            rows_affected := 0;
            status := 'ERROR';
            error_message := SQLERRM;
            RETURN NEXT;
        END IF;
    END;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 2. FUNCIÓN: Refresh de todas las vistas SEO
-- ============================================
CREATE OR REPLACE FUNCTION refresh_all_seo_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH refreshes AS (
        SELECT * FROM refresh_materialized_view_safe('mv_top_tags_seo', TRUE)
        UNION ALL
        SELECT * FROM refresh_materialized_view_safe('mv_top_countries_seo', TRUE)
    )
    SELECT
        view,
        duration_ms,
        rows_affected,
        status,
        error_message
    FROM refreshes;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 3. FUNCIÓN: Refresh de vistas de analytics
-- ============================================
CREATE OR REPLACE FUNCTION refresh_all_analytics_views()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT,
    error_message TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM refresh_materialized_view_safe('mv_station_stats_7d', TRUE);
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 4. FUNCIÓN: Refresh de TODAS las vistas materializadas
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
    RETURN QUERY
    WITH all_refreshes AS (
        SELECT * FROM refresh_materialized_view_safe('mv_top_tags_seo', TRUE)
        UNION ALL
        SELECT * FROM refresh_materialized_view_safe('mv_top_countries_seo', TRUE)
        UNION ALL
        SELECT * FROM refresh_materialized_view_safe('mv_station_stats_7d', TRUE)
    )
    SELECT
        view,
        duration_ms,
        rows_affected,
        status,
        error_message
    FROM all_refreshes;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 5. TABLA: Log de refresh de vistas materializadas
-- ============================================
CREATE TABLE IF NOT EXISTS materialized_view_refresh_log (
    id SERIAL PRIMARY KEY,
    view_name TEXT NOT NULL,
    refresh_started_at TIMESTAMPTZ NOT NULL,
    refresh_completed_at TIMESTAMPTZ NOT NULL,
    duration_ms BIGINT NOT NULL,
    rows_affected BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('SUCCESS', 'SUCCESS_NON_CONCURRENT', 'ERROR')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mv_refresh_log_view ON materialized_view_refresh_log(view_name, refresh_started_at DESC);
CREATE INDEX idx_mv_refresh_log_status ON materialized_view_refresh_log(status, refresh_started_at DESC);

-- ============================================
-- 6. FUNCIÓN: Refresh con logging
-- ============================================
CREATE OR REPLACE FUNCTION refresh_all_views_with_logging()
RETURNS TABLE(
    view_name TEXT,
    duration_ms BIGINT,
    rows_affected BIGINT,
    status TEXT
) AS $$
DECLARE
    refresh_result RECORD;
BEGIN
    FOR refresh_result IN
        SELECT * FROM refresh_all_materialized_views()
    LOOP
        -- Insertar en log
        INSERT INTO materialized_view_refresh_log (
            view_name,
            refresh_started_at,
            refresh_completed_at,
            duration_ms,
            rows_affected,
            status,
            error_message
        ) VALUES (
            refresh_result.view_name,
            refresh_result.refresh_started_at,
            refresh_result.refresh_completed_at,
            refresh_result.duration_ms,
            refresh_result.rows_affected,
            refresh_result.status,
            refresh_result.error_message
        );

        -- Retornar resultado
        view_name := refresh_result.view_name;
        duration_ms := refresh_result.duration_ms;
        rows_affected := refresh_result.rows_affected;
        status := refresh_result.status;
        RETURN NEXT;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 7. FUNCIÓN: Obtener estadísticas de refresh
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
        (SELECT status FROM materialized_view_refresh_log
         WHERE view_name = l.view_name
         ORDER BY refresh_completed_at DESC LIMIT 1) as last_status
    FROM materialized_view_refresh_log l
    WHERE l.refresh_started_at > NOW() - (days_back || ' days')::INTERVAL
    GROUP BY l.view_name
    ORDER BY l.view_name;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 8. SETUP: Intentar instalar pg_cron (opcional)
-- ============================================
DO $$
BEGIN
    -- Intentar crear extensión pg_cron
    CREATE EXTENSION IF NOT EXISTS pg_cron;

    -- Si existe, configurar jobs
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        -- Eliminar jobs existentes si existen
        PERFORM cron.unschedule(jobid)
        FROM cron.job
        WHERE jobname IN ('refresh-seo-views', 'refresh-analytics-views');

        -- Refresh vistas SEO cada 6 horas
        PERFORM cron.schedule(
            'refresh-seo-views',
            '0 */6 * * *',
            $CRON$SELECT * FROM refresh_all_seo_views()$CRON$
        );

        -- Refresh vistas analytics cada hora
        PERFORM cron.schedule(
            'refresh-analytics-views',
            '0 * * * *',
            $CRON$SELECT * FROM refresh_all_analytics_views()$CRON$
        );

        RAISE NOTICE 'pg_cron configured successfully with automatic refresh jobs';
    ELSE
        RAISE NOTICE 'pg_cron extension not available. Use external cron for scheduled refreshes.';
        RAISE NOTICE 'Add to crontab: 0 */6 * * * psql -U radio -d radio_backend -c "SELECT * FROM refresh_all_seo_views()"';
    END IF;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'Insufficient privileges to install pg_cron. Use external cron instead.';
    WHEN OTHERS THEN
        RAISE NOTICE 'Could not configure pg_cron: %. Use external cron instead.', SQLERRM;
END;
$$;

-- ============================================
-- Comentarios
-- ============================================
COMMENT ON FUNCTION refresh_materialized_view_safe(TEXT, BOOLEAN) IS
'Refresh seguro de vista materializada con fallback a modo no-concurrente y logging';

COMMENT ON FUNCTION refresh_all_seo_views() IS
'Refresh de todas las vistas materializadas relacionadas con SEO';

COMMENT ON FUNCTION refresh_all_analytics_views() IS
'Refresh de todas las vistas materializadas de analytics';

COMMENT ON FUNCTION refresh_all_materialized_views() IS
'Refresh de TODAS las vistas materializadas del sistema';

COMMENT ON FUNCTION refresh_all_views_with_logging() IS
'Refresh de todas las vistas con logging automático en materialized_view_refresh_log';

COMMENT ON TABLE materialized_view_refresh_log IS
'Log histórico de refresh de vistas materializadas para monitoreo y debugging';

COMMENT ON FUNCTION get_refresh_statistics(INTEGER) IS
'Obtiene estadísticas de refresh de vistas materializadas de los últimos N días';

-- ============================================
-- Verificación y primer refresh
-- ============================================
-- NOTA: El primer refresh se debe ejecutar manualmente vía endpoint REST:
-- POST /api/v1/admin/maintenance/refresh-views
-- O directamente en SQL:
-- SELECT * FROM refresh_all_views_with_logging();
