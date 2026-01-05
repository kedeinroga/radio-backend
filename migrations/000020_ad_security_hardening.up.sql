-- Migration: Ad security hardening
-- Description: Mejoras de seguridad para el sistema de publicidad

-- 1. Agregar índice para detección de fraude por IP
CREATE INDEX idx_impressions_ip_created
ON ad_impressions(ip_address, created_at);

-- 2. Función para detectar clicks sospechosos
CREATE OR REPLACE FUNCTION detect_suspicious_clicks()
RETURNS TRIGGER AS $$
DECLARE
    recent_clicks INTEGER;
BEGIN
    -- Contar clicks de la misma IP en los últimos 5 minutos
    SELECT COUNT(*) INTO recent_clicks
    FROM ad_clicks c
    JOIN ad_impressions i ON c.impression_id = i.id
    WHERE i.ip_address = (
        SELECT ip_address
        FROM ad_impressions
        WHERE id = NEW.impression_id
    )
    AND c.created_at > NOW() - INTERVAL '5 minutes';

    -- Si hay más de 5 clicks, marcar como sospechoso (logging, no bloqueamos)
    IF recent_clicks > 5 THEN
        RAISE WARNING 'Suspicious click pattern detected from IP, impression_id: %', NEW.impression_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_suspicious_clicks
BEFORE INSERT ON ad_clicks
FOR EACH ROW
EXECUTE FUNCTION detect_suspicious_clicks();

-- 3. Row-Level Security (RLS) para campañas
ALTER TABLE ad_campaigns ENABLE ROW LEVEL SECURITY;

-- Policy: Solo el advertiser dueño o admins pueden ver/editar sus campañas
CREATE POLICY campaign_owner_policy ON ad_campaigns
FOR ALL
USING (
    advertiser_id = current_setting('app.current_user_id', true)::UUID
    OR
    (SELECT user_type FROM users WHERE id = current_setting('app.current_user_id', true)::UUID) = 'admin'
);

-- 4. Audit log para cambios en anuncios
CREATE TABLE ad_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    advertisement_id UUID NOT NULL,
    user_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'created', 'updated', 'deleted', 'status_changed'
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ad_audit_log_ad ON ad_audit_log(advertisement_id, created_at);
CREATE INDEX idx_ad_audit_log_user ON ad_audit_log(user_id, created_at);
CREATE INDEX idx_ad_audit_log_action ON ad_audit_log(action, created_at);

-- 5. Función helper para logging de audit
CREATE OR REPLACE FUNCTION log_ad_change()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'UPDATE') THEN
        INSERT INTO ad_audit_log (
            advertisement_id,
            user_id,
            action,
            old_values,
            new_values
        ) VALUES (
            NEW.id,
            current_setting('app.current_user_id', true)::UUID,
            'updated',
            to_jsonb(OLD),
            to_jsonb(NEW)
        );
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO ad_audit_log (
            advertisement_id,
            user_id,
            action,
            old_values
        ) VALUES (
            OLD.id,
            current_setting('app.current_user_id', true)::UUID,
            'deleted',
            to_jsonb(OLD)
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Aplicar trigger de audit a advertisements
CREATE TRIGGER audit_advertisement_changes
AFTER UPDATE OR DELETE ON advertisements
FOR EACH ROW
EXECUTE FUNCTION log_ad_change();

-- Comentarios
COMMENT ON TABLE ad_audit_log IS 'Log de auditoría de cambios en anuncios';
COMMENT ON FUNCTION detect_suspicious_clicks() IS 'Detecta patrones sospechosos de clicks (>5 en 5 min de misma IP)';
