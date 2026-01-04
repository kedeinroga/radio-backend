-- ===============================================
-- Rollback Migración 000012: Automated Materialized View Refresh
-- ===============================================

-- Desinstalar pg_cron jobs (si existen)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.unschedule(jobid)
        FROM cron.job
        WHERE jobname IN ('refresh-seo-views', 'refresh-analytics-views');
    END IF;
END;
$$;

-- Drop functions (en orden inverso)
DROP FUNCTION IF EXISTS get_refresh_statistics(INTEGER);
DROP FUNCTION IF EXISTS refresh_all_views_with_logging();
DROP TABLE IF EXISTS materialized_view_refresh_log;
DROP FUNCTION IF EXISTS refresh_all_materialized_views();
DROP FUNCTION IF EXISTS refresh_all_analytics_views();
DROP FUNCTION IF EXISTS refresh_all_seo_views();
DROP FUNCTION IF EXISTS refresh_materialized_view_safe(TEXT, BOOLEAN);
