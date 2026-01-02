-- ===============================================
-- Migración 000002: Users Table
-- ===============================================
-- Tabla de usuarios optimizada con UUID nativo y ENUM

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(72) NOT NULL, -- bcrypt max 72 chars
    user_type user_type_enum NOT NULL DEFAULT 'guest',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices optimizados
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_user_type ON users(user_type);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Trigger para auto-update updated_at
CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Comentarios para documentación
COMMENT ON TABLE users IS 'Application users with authentication data';
COMMENT ON COLUMN users.id IS 'UUID primary key - generated automatically';
COMMENT ON COLUMN users.email IS 'Unique email address - used for login';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password (max 72 chars)';
COMMENT ON COLUMN users.user_type IS 'User role: guest (default), premium, or admin';
