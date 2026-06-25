-- ============================================================
--  000029_data_retention_policies.up.sql
--
--  Automatic data retention via pg_cron:
--
--  - request_logs / search_queries / station_plays partitions:
--    drop partitions older than 2 months (uses existing
--    cleanup_old_partitions function from migration 000011).
--  - station_tracks: delete rows older than 7 days.
--  - station_search_cache: truncate weekly (pure cache).
--  - token_blacklist: delete expired tokens daily.
--
--  All jobs are also run immediately to reclaim space now.
-- ============================================================

BEGIN;

-- ────────────────────────────────────────────────────────────
-- 1. station_tracks — keep 7 days of now-playing history
-- ────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.cleanup_station_tracks(
    retention_days INTEGER DEFAULT 7
)
RETURNS INTEGER
LANGUAGE plpgsql
SET search_path = public, pg_catalog
AS $$
DECLARE
    deleted INTEGER;
BEGIN
    DELETE FROM public.station_tracks
    WHERE played_at < NOW() - make_interval(days => retention_days);

    GET DIAGNOSTICS deleted = ROW_COUNT;
    RAISE NOTICE 'cleanup_station_tracks: deleted % rows older than % days', deleted, retention_days;
    RETURN deleted;
END;
$$;

-- ────────────────────────────────────────────────────────────
-- 2. station_search_cache — full truncate (rebuilds on demand)
-- ────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.cleanup_station_search_cache()
RETURNS VOID
LANGUAGE plpgsql
SET search_path = public, pg_catalog
AS $$
BEGIN
    TRUNCATE public.station_search_cache;
    RAISE NOTICE 'cleanup_station_search_cache: table truncated';
END;
$$;

-- ────────────────────────────────────────────────────────────
-- 3. token_blacklist — delete expired tokens
-- ────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.cleanup_token_blacklist()
RETURNS INTEGER
LANGUAGE plpgsql
SET search_path = public, pg_catalog
AS $$
DECLARE
    deleted INTEGER;
BEGIN
    DELETE FROM public.token_blacklist
    WHERE expires_at < NOW();

    GET DIAGNOSTICS deleted = ROW_COUNT;
    RAISE NOTICE 'cleanup_token_blacklist: deleted % expired tokens', deleted;
    RETURN deleted;
END;
$$;

-- ────────────────────────────────────────────────────────────
-- 4. Schedule all jobs via pg_cron
-- ────────────────────────────────────────────────────────────
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        RAISE NOTICE 'pg_cron not available — run retention functions manually';
        RETURN;
    END IF;

    -- Drop existing jobs to avoid duplicates on re-run
    PERFORM cron.unschedule(jobid) FROM cron.job WHERE jobname IN (
        'cleanup-old-partitions',
        'cleanup-station-tracks',
        'cleanup-station-search-cache',
        'cleanup-token-blacklist'
    );

    -- Partitions: 1st of each month at 01:00, retain 2 months
    PERFORM cron.schedule(
        'cleanup-old-partitions',
        '0 1 1 * *',
        $CRON$SELECT * FROM public.cleanup_old_partitions(2)$CRON$
    );

    -- station_tracks: daily at 02:00, retain 7 days
    PERFORM cron.schedule(
        'cleanup-station-tracks',
        '0 2 * * *',
        $CRON$SELECT public.cleanup_station_tracks(7)$CRON$
    );

    -- station_search_cache: every Sunday at 03:00
    PERFORM cron.schedule(
        'cleanup-station-search-cache',
        '0 3 * * 0',
        $CRON$SELECT public.cleanup_station_search_cache()$CRON$
    );

    -- token_blacklist: daily at 03:30
    PERFORM cron.schedule(
        'cleanup-token-blacklist',
        '30 3 * * *',
        $CRON$SELECT public.cleanup_token_blacklist()$CRON$
    );

    RAISE NOTICE 'Retention pg_cron jobs scheduled';
END $$;

-- ────────────────────────────────────────────────────────────
-- 5. Run immediately to reclaim space now
-- ────────────────────────────────────────────────────────────
SELECT * FROM public.cleanup_old_partitions(2);
SELECT public.cleanup_station_tracks(7);
SELECT public.cleanup_station_search_cache();
SELECT public.cleanup_token_blacklist();

COMMIT;
