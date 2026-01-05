-- Migration: Drop ad_campaigns table
-- Description: Rollback de la tabla de campañas publicitarias

DROP TRIGGER IF EXISTS update_ad_campaigns_updated_at ON ad_campaigns;
DROP TABLE IF EXISTS ad_campaigns CASCADE;
