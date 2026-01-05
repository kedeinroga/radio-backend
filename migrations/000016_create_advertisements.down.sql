-- Migration: Drop advertisements table
-- Description: Rollback de la tabla de anuncios

DROP TRIGGER IF EXISTS update_advertisements_updated_at ON advertisements;
DROP TABLE IF EXISTS advertisements CASCADE;
