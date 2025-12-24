-- Rollback de traducciones de estaciones
DROP TRIGGER IF EXISTS trigger_update_station_translations_timestamp ON station_translations;
DROP FUNCTION IF EXISTS update_station_translations_updated_at();
DROP TABLE IF EXISTS station_translations CASCADE;
