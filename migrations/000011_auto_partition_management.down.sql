-- ===============================================
-- Rollback: Automatic Partition Management
-- ===============================================

-- Drop verification functions
DROP FUNCTION IF EXISTS check_missing_partitions(INTEGER);
DROP FUNCTION IF EXISTS list_partition_status();
DROP FUNCTION IF EXISTS cleanup_old_partitions(INTEGER);

-- Drop triggers
DROP TRIGGER IF EXISTS auto_create_partition_search_queries ON search_queries;
DROP TRIGGER IF EXISTS auto_create_partition_request_logs ON request_logs;
DROP TRIGGER IF EXISTS auto_create_partition_station_plays ON station_plays;

-- Drop function
DROP FUNCTION IF EXISTS create_partition_if_not_exists();
