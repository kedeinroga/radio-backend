-- ============================================================
--  000025_fix_partition_triggers.up.sql
--  BUGFIX: Drop BEFORE INSERT triggers on partitioned tables.
--
--  Root cause: In PostgreSQL 13+, row-level BEFORE INSERT triggers
--  defined on a partitioned table fire on the TARGET PARTITION, so
--  TG_TABLE_NAME = 'request_logs_2026_03' (not 'request_logs').
--  The function then tries to CREATE TABLE ... PARTITION OF that
--  partition, which (a) requires AccessExclusiveLock while the
--  session already holds it, and (b) makes no sense because the
--  partition is not itself partitioned.
--
--  Fix: Remove the triggers. Partitions are pre-created by a
--  monthly pg_cron job (see below) and are already seeded several
--  months ahead in earlier migrations.
-- ============================================================

BEGIN;

-- ────────────────────────────────────────────────
--  1. Drop the broken triggers
-- ────────────────────────────────────────────────
DROP TRIGGER IF EXISTS auto_create_partition_station_plays  ON public.station_plays;
DROP TRIGGER IF EXISTS auto_create_partition_request_logs   ON public.request_logs;
DROP TRIGGER IF EXISTS auto_create_partition_search_queries ON public.search_queries;

-- ────────────────────────────────────────────────
--  2. Replace create_partition_if_not_exists with a
--     safe, proactive version (no longer a trigger).
--     Creates the NEXT N months for all three tables.
-- ────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.create_upcoming_partitions(
    months_ahead INTEGER DEFAULT 3
)
RETURNS TABLE(
    table_name      TEXT,
    partition_name  TEXT,
    action          TEXT
)
LANGUAGE plpgsql
SET search_path = public, extensions, pg_catalog
AS $$
DECLARE
    base_tables TEXT[]  := ARRAY['station_plays', 'request_logs', 'search_queries'];
    base_table  TEXT;
    i           INTEGER;
    part_date   DATE;
    part_name   TEXT;
    start_d     DATE;
    end_d       DATE;
BEGIN
    FOREACH base_table IN ARRAY base_tables LOOP
        FOR i IN 0..months_ahead LOOP
            part_date := DATE_TRUNC('month', CURRENT_DATE + make_interval(months => i));
            part_name := base_table || '_' || TO_CHAR(part_date, 'YYYY_MM');
            start_d   := part_date;
            end_d     := part_date + INTERVAL '1 month';

            IF NOT EXISTS (
                SELECT 1 FROM pg_tables
                WHERE schemaname = 'public' AND tablename = part_name
            ) THEN
                EXECUTE format(
                    'CREATE TABLE IF NOT EXISTS %I PARTITION OF %I
                     FOR VALUES FROM (%L) TO (%L)',
                    part_name, base_table, start_d, end_d
                );

                table_name     := base_table;
                partition_name := part_name;
                action         := 'CREATED';
                RETURN NEXT;

                RAISE NOTICE 'Created partition: % [% , %)', part_name, start_d, end_d;
            ELSE
                table_name     := base_table;
                partition_name := part_name;
                action         := 'EXISTS';
                RETURN NEXT;
            END IF;
        END LOOP;
    END LOOP;
END;
$$;

COMMENT ON FUNCTION public.create_upcoming_partitions(INTEGER) IS
'Pre-creates monthly partitions for station_plays, request_logs, and search_queries
for the current month plus months_ahead months. Safe to call at any time.
Intended to be called by pg_cron on the 1st of each month.';

-- ────────────────────────────────────────────────
--  3. Run it immediately to ensure no gaps exist
--     for the current month + next 3 months.
-- ────────────────────────────────────────────────
SELECT * FROM public.create_upcoming_partitions(3);

-- ────────────────────────────────────────────────
--  4. Schedule via pg_cron (Supabase has pg_cron)
--     Runs at 00:05 on the 1st of every month.
-- ────────────────────────────────────────────────
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        -- Remove old job if it exists
        PERFORM cron.unschedule(jobid)
        FROM cron.job
        WHERE jobname = 'create-upcoming-partitions';

        -- Schedule: 00:05 on the 1st of each month
        PERFORM cron.schedule(
            'create-upcoming-partitions',
            '5 0 1 * *',
            $CRON$SELECT * FROM public.create_upcoming_partitions(3)$CRON$
        );

        RAISE NOTICE 'pg_cron job "create-upcoming-partitions" scheduled (5 0 1 * *)';
    ELSE
        RAISE NOTICE 'pg_cron not available — run create_upcoming_partitions(3) manually each month';
    END IF;
END $$;

COMMIT;
