-- Migration: Drop user_ad_profiles table
-- Description: Rollback de la tabla de perfiles de usuario

DROP TRIGGER IF EXISTS reset_ad_counters_trigger ON user_ad_profiles;
DROP FUNCTION IF EXISTS reset_ad_counters();
DROP TRIGGER IF EXISTS update_user_ad_profiles_updated_at ON user_ad_profiles;
DROP TABLE IF EXISTS user_ad_profiles CASCADE;
