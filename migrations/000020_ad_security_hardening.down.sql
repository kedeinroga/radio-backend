-- Revert ad security hardening

-- 1. Eliminar trigger de audit
DROP TRIGGER IF EXISTS audit_advertisement_changes ON advertisements;
DROP FUNCTION IF EXISTS log_ad_change();

-- 2. Eliminar tabla de audit
DROP TABLE IF EXISTS ad_audit_log CASCADE;

-- 3. Eliminar RLS de campañas
DROP POLICY IF EXISTS campaign_owner_policy ON ad_campaigns;
ALTER TABLE ad_campaigns DISABLE ROW LEVEL SECURITY;

-- 4. Eliminar trigger de detección de clicks sospechosos
DROP TRIGGER IF EXISTS check_suspicious_clicks ON ad_clicks;
DROP FUNCTION IF EXISTS detect_suspicious_clicks();

-- 5. Eliminar índice de fraude por IP
DROP INDEX IF EXISTS idx_impressions_ip_created;
