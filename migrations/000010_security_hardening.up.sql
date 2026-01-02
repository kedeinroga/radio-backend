-- ===============================================
-- Migración 000010: Security Hardening
-- ===============================================
-- Triggers y constraints de seguridad adicionales

-- ============================================
-- 1. PREVENIR PRIVILEGE ESCALATION
-- ============================================
CREATE OR REPLACE FUNCTION prevent_privilege_escalation()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.user_type = 'admin' AND OLD.user_type != 'admin' THEN
        -- Log intento de escalación
        INSERT INTO security_events (event_type, user_id, reason, metadata)
        VALUES ('privilege_escalation_attempt', NEW.id, 'Unauthorized admin escalation', 
                jsonb_build_object('old_type', OLD.user_type, 'new_type', NEW.user_type));
        
        RAISE EXCEPTION 'Cannot escalate to admin without authorization';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_user_type_escalation
    BEFORE UPDATE ON users
    FOR EACH ROW
    WHEN (OLD.user_type IS DISTINCT FROM NEW.user_type)
    EXECUTE FUNCTION prevent_privilege_escalation();

-- ============================================
-- 2. AUDIT LOG DE CAMBIOS DE ROLES
-- ============================================
CREATE OR REPLACE FUNCTION audit_role_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.user_type != NEW.user_type THEN
        INSERT INTO security_events (event_type, user_id, reason, metadata)
        VALUES (
            'role_changed',
            NEW.id,
            'User role modified',
            jsonb_build_object(
                'old_role', OLD.user_type,
                'new_role', NEW.user_type,
                'changed_at', NOW()
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_user_role_changes
    AFTER UPDATE ON users
    FOR EACH ROW
    WHEN (OLD.user_type IS DISTINCT FROM NEW.user_type)
    EXECUTE FUNCTION audit_role_changes();

-- ============================================
-- 3. SANITIZACIÓN DE TOKENS EN LOGS
-- ============================================
CREATE OR REPLACE FUNCTION sanitize_token_in_logs()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.token_id IS NOT NULL AND LENGTH(NEW.token_id) > 8 THEN
        NEW.token_id := LEFT(NEW.token_id, 8) || '***';
    END IF;
    IF NEW.session_id IS NOT NULL AND LENGTH(NEW.session_id) > 8 THEN
        NEW.session_id := LEFT(NEW.session_id, 8) || '***';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sanitize_security_event_logs
    BEFORE INSERT ON security_events
    FOR EACH ROW
    EXECUTE FUNCTION sanitize_token_in_logs();

-- ============================================
-- 4. PROTECCIÓN CONTRA PARTITION OVERFLOW (DoS)
-- ============================================
CREATE OR REPLACE FUNCTION prevent_partition_overflow()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.created_at < NOW() - INTERVAL '3 months' THEN
        RAISE EXCEPTION 'Cannot insert data older than 3 months';
    END IF;
    
    IF NEW.created_at > NOW() + INTERVAL '6 months' THEN
        RAISE EXCEPTION 'Cannot insert data more than 6 months in future';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_partition_bounds_plays
    BEFORE INSERT ON station_plays
    FOR EACH ROW
    EXECUTE FUNCTION prevent_partition_overflow();

CREATE TRIGGER check_partition_bounds_logs
    BEFORE INSERT ON request_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_partition_overflow();

CREATE TRIGGER check_partition_bounds_searches
    BEFORE INSERT ON search_queries
    FOR EACH ROW
    EXECUTE FUNCTION prevent_partition_overflow();

-- ============================================
-- 5. VALIDACIÓN DE HTML EN INPUTS (Anti-XSS)
-- ============================================
ALTER TABLE stations ADD CONSTRAINT check_name_safe
    CHECK (name !~ '<script|<iframe|javascript:|onerror=|onclick=');

ALTER TABLE station_translations ADD CONSTRAINT check_title_safe
    CHECK (title !~ '<script|<iframe|javascript:|onerror=|onclick=');

-- ============================================
-- 6. BRUTE FORCE PROTECTION POR IP
-- ============================================
CREATE TABLE brute_force_attempts (
    ip_address INET PRIMARY KEY,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    unique_emails_tried INTEGER NOT NULL DEFAULT 0,
    first_attempt TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_attempt TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    blocked_until TIMESTAMPTZ
);

CREATE INDEX idx_brute_force_blocked ON brute_force_attempts(is_blocked, blocked_until) 
    WHERE is_blocked = TRUE;

-- ============================================
-- 7. VIEW SEGURA PARA ADMINS (SIN PASSWORD_HASH)
-- ============================================
CREATE VIEW users_safe AS
SELECT 
    id, 
    email, 
    user_type, 
    created_at, 
    updated_at
FROM users;

-- ============================================
-- 8. RATE LIMITING TRACKING
-- ============================================
CREATE TABLE rate_limit_tracking (
    resource_key VARCHAR(255) PRIMARY KEY,
    request_count INTEGER NOT NULL DEFAULT 0,
    window_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_request TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rate_limit_window ON rate_limit_tracking(window_start);

-- Función de limpieza
CREATE OR REPLACE FUNCTION cleanup_rate_limits()
RETURNS void AS $$
BEGIN
    DELETE FROM rate_limit_tracking 
    WHERE window_start < NOW() - INTERVAL '10 minutes';
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- COMENTARIOS DE DOCUMENTACIÓN
-- ============================================
COMMENT ON TRIGGER prevent_user_type_escalation ON users IS 
    'Prevents unauthorized privilege escalation to admin role';

COMMENT ON TRIGGER audit_user_role_changes ON users IS 
    'Audits all user role changes to security_events table';

COMMENT ON FUNCTION prevent_partition_overflow() IS 
    'Prevents DoS via inserts in non-existent partitions';

COMMENT ON CONSTRAINT check_name_safe ON stations IS 
    'Prevents XSS attacks via station names';

COMMENT ON TABLE brute_force_attempts IS 
    'Tracks brute force login attempts by IP address';

COMMENT ON VIEW users_safe IS 
    'Safe view of users without password_hash for admin queries';
