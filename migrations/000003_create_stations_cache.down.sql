-- Drop stations cache table
DROP INDEX IF EXISTS idx_stations_is_active;
DROP INDEX IF EXISTS idx_stations_last_synced;
DROP INDEX IF EXISTS idx_stations_tags;
DROP INDEX IF EXISTS idx_stations_votes;
DROP INDEX IF EXISTS idx_stations_country;
DROP INDEX IF EXISTS idx_stations_name_trgm;

DROP TABLE IF EXISTS stations;
