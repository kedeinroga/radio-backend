-- ===============================================
-- Migración 000006: Sessions and Security
-- ===============================================
-- Tablas de sesiones, eventos de seguridad y login attempts

-- ============================================
-- 1. SESSIONS
-- ============================================
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(64) NOT NULL UNIQUE,
    token_id VARCHAR(64) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    
    -- Device info desnormalizado
    browser VARCHAR(100),
    os VARCHAR(100),
    device_type VARCHAR(20),
    country CHAR(2),
    city VARCHAR(100),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Índices optimizados con parciales
CREATE INDEX idx_sessions_user_active ON sessions(user_id, is_active, last_activity DESC) 
    WHERE is_active = true;
CREATE INDEX idx_sessions_session_id ON sessions(session_id);
CREATE INDEX idx_sessions_token_id ON sessions(token_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at) 
    WHERE is_active = true;

-- ============================================
-- 2. SECURITY EVENTS
-- ============================================
CREATE TABLE security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_type VARCHAR(50) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    email VARCHAR(255),
    token_id VARCHAR(64),
    session_id VARCHAR(64),
    ip_address INET,
    user_agent TEXT,
    reason VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices
CREATE INDEX idx_security_events_timestamp ON security_events(timestamp DESC);
CREATE INDEX idx_security_events_user ON security_events(user_id, timestamp DESC) 
    WHERE user_id IS NOT NULL;
CREATE INDEX idx_security_events_type ON security_events(event_type, timestamp DESC);
CREATE INDEX idx_security_events_ip ON security_events(ip_address, timestamp DESC) 
    WHERE ip_address IS NOT NULL;

-- ============================================
-- 3. LOGIN ATTEMPTS
-- ============================================
CREATE TABLE login_attempts (
    email VARCHAR(255) PRIMARY KEY,
    failed_count SMALLINT NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    last_attempt TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    unlock_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices
CREATE INDEX idx_login_attempts_locked ON login_attempts(is_locked, unlock_at) 
    WHERE is_locked = TRUE;
CREATE INDEX idx_login_attempts_last ON login_attempts(last_attempt DESC);

-- ============================================
-- FUNCIONES DE MANTENIMIENTO
-- ============================================

-- Auto-cleanup de sesiones expiradas
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS void AS $$
BEGIN
    UPDATE sessions 
    SET is_active = false 
    WHERE is_active = true 
      AND expires_at < NOW();
      
    DELETE FROM sessions 
    WHERE is_active = false 
      AND expires_at < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- Auto-unlock de cuentas
CREATE OR REPLACE FUNCTION unlock_expired_accounts()
RETURNS void AS $$
BEGIN
    UPDATE login_attempts 
    SET is_locked = FALSE, 
        failed_count = 0,
        unlock_at = NULL
    WHERE is_locked = TRUE 
      AND unlock_at IS NOT NULL 
      AND unlock_at <= NOW();
END;
$$ LANGUAGE plpgsql;

-- Comentarios
COMMENT ON TABLE sessions IS 'Active user sessions with device info';
COMMENT ON TABLE security_events IS 'Security audit log for all authentication events';
COMMENT ON TABLE login_attempts IS 'Failed login tracking for brute force protection';
COMMENT ON FUNCTION cleanup_expired_sessions() IS 'Cleans up expired and old inactive sessions';
COMMENT ON FUNCTION unlock_expired_accounts() IS 'Auto-unlocks accounts after timeout period';
