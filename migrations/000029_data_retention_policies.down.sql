-- ============================================================
--  000029_data_retention_policies.down.sql
-- ============================================================

BEGIN;

-- Remove pg_cron jobs
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.unschedule(jobid) FROM cron.job WHERE jobname IN (
            'cleanup-old-partitions',
            'cleanup-station-tracks',
            'cleanup-station-search-cache',
            'cleanup-token-blacklist'
        );
    END IF;
END $$;

-- Drop retention functions
DROP FUNCTION IF EXISTS public.cleanup_station_tracks(INTEGER);
DROP FUNCTION IF EXISTS public.cleanup_station_search_cache();
DROP FUNCTION IF EXISTS public.cleanup_token_blacklist();

COMMIT;
