-- ============================================================
--  000026_drop_orphan_partition_function.down.sql
--
--  Restores create_partition_if_not_exists() as a standalone function
--  (intentionally WITHOUT re-attaching triggers — those are managed
--  by migrations 000011/000025).
-- ============================================================

CREATE OR REPLACE FUNCTION public.create_partition_if_not_exists()
RETURNS TRIGGER AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    partition_date := DATE_TRUNC('month', NEW.created_at);
    partition_name := TG_TABLE_NAME || '_' || TO_CHAR(partition_date, 'YYYY_MM');

    IF NOT EXISTS (
        SELECT 1 FROM pg_tables
        WHERE schemaname = 'public' AND tablename = partition_name
    ) THEN
        start_date := partition_date;
        end_date   := partition_date + INTERVAL '1 month';
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF %I
             FOR VALUES FROM (%L) TO (%L)',
            partition_name, TG_TABLE_NAME, start_date, end_date
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql
SET search_path = public, extensions, pg_catalog;
