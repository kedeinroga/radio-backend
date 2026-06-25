-- ============================================================
--  000028_enable_rls_new_tables.down.sql
-- ============================================================

BEGIN;

-- Restore create_upcoming_partitions to the 000025 version (without RLS)
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

                RAISE NOTICE 'Created partition: % [%, %)', part_name, start_d, end_d;
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

ALTER TABLE public.station_tracks                  DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.station_plays_2026_07  DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.request_logs_2026_07   DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.search_queries_2026_07 DISABLE ROW LEVEL SECURITY;

COMMIT;
