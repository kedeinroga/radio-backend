-- ============================================================
--  000025_fix_partition_triggers.down.sql
-- ============================================================

BEGIN;

-- Remove the cron job
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.unschedule(jobid)
        FROM cron.job
        WHERE jobname = 'create-upcoming-partitions';
    END IF;
END $$;

-- Restore the original (broken) trigger function
CREATE OR REPLACE FUNCTION create_partition_if_not_exists()
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
        end_date := partition_date + INTERVAL '1 month';
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF %I
             FOR VALUES FROM (%L) TO (%L)',
            partition_name, TG_TABLE_NAME, start_date, end_date
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER auto_create_partition_station_plays
    BEFORE INSERT ON station_plays
    FOR EACH ROW EXECUTE FUNCTION create_partition_if_not_exists();

CREATE TRIGGER auto_create_partition_request_logs
    BEFORE INSERT ON request_logs
    FOR EACH ROW EXECUTE FUNCTION create_partition_if_not_exists();

CREATE TRIGGER auto_create_partition_search_queries
    BEFORE INSERT ON search_queries
    FOR EACH ROW EXECUTE FUNCTION create_partition_if_not_exists();

COMMIT;
