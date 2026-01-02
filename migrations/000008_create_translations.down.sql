-- ===============================================
-- Rollback 000008: Station Translations
-- ===============================================

DROP TRIGGER IF EXISTS station_translations_updated_at ON station_translations;
DROP TABLE IF EXISTS station_translations CASCADE;
