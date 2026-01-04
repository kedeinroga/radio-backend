-- ===============================================
-- Migración 000001: Types and Extensions
-- ===============================================
-- Crea tipos base y extensiones necesarias para el proyecto

-- Extensiones necesarias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- Full-text search trigram

-- Tipos ENUM para validación a nivel de BD
DROP TYPE IF EXISTS user_type_enum CASCADE;
CREATE TYPE user_type_enum AS ENUM ('guest', 'premium', 'admin');

-- Función reutilizable para auto-update del campo updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_updated_at() IS 'Auto-updates updated_at column on row update';
COMMENT ON TYPE user_type_enum IS 'User types: guest (default), premium (paid), admin (full access)';
