-- Drop search cache table
DROP INDEX IF EXISTS idx_search_cache_expires;
DROP INDEX IF EXISTS idx_search_cache_hash;

DROP TABLE IF EXISTS station_search_cache;
