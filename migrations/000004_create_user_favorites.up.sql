-- ===============================================
-- Migración 000004: User Favorites
-- ===============================================
-- Tabla de favoritos de usuarios con foreign keys

CREATE TABLE user_favorites (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    station_id VARCHAR(255) NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, station_id)
);

-- Índices optimizados
CREATE INDEX idx_user_favorites_user_id ON user_favorites(user_id, created_at DESC);
CREATE INDEX idx_user_favorites_station_id ON user_favorites(station_id);

-- Comentarios
COMMENT ON TABLE user_favorites IS 'Junction table for user favorite stations';
COMMENT ON COLUMN user_favorites.user_id IS 'Reference to users table';
COMMENT ON COLUMN user_favorites.station_id IS 'Reference to stations table';
