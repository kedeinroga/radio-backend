-- Migration: Drop ad_impressions table
-- Description: Rollback de la tabla de impresiones (incluye todas las particiones)

DROP TABLE IF EXISTS ad_impressions CASCADE;
