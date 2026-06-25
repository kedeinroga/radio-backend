-- ============================================================
--  000028_enable_rls_new_tables.up.sql
--
--  Two issues fixed:
--
--  1. station_tracks (created in 000027) was never given RLS.
--     It is backend-only (service role), so no public policies
--     are needed — enabling RLS is sufficient to block PostgREST.
--
--  2. The July 2026 partitions were pre-created by create_upcoming_partitions()
--     (scheduled in 000025) before RLS was a requirement, so they
--     have no RLS. Same fix: enable, no policy.
--
--  3. create_upcoming_partitions() is updated to enable RLS
--     immediately after creating each new partition, so this
--     situation cannot recur for future months.
-- ============================================================

BEGIN;

-- 1. station_tracks
ALTER TABLE public.station_tracks ENABLE ROW LEVEL SECURITY;

-- 2. July 2026 partitions (exist in production, may not exist in local envs)
ALTER TABLE IF EXISTS public.station_plays_2026_07   ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.request_logs_2026_07    ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.search_queries_2026_07  ENABLE ROW LEVEL SECURITY;

-- 3. Update create_upcoming_partitions so future partitions get RLS at birth
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

                EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', part_name);

                table_name     := base_table;
                partition_name := part_name;
                action         := 'CREATED';
                RETURN NEXT;

                RAISE NOTICE 'Created partition with RLS: % [%, %)', part_name, start_d, end_d;
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

COMMIT;
