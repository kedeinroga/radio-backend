-- Migration: Drop ad_clicks table
-- Description: Rollback de la tabla de clicks

DROP TABLE IF EXISTS ad_clicks CASCADE;
