-- Create user_favorites table
CREATE TABLE IF NOT EXISTS user_favorites (
    user_id VARCHAR(36) NOT NULL,
    station_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, station_id),
    CONSTRAINT fk_user_favorites_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Create index on user_id for faster lookups of user's favorites
CREATE INDEX idx_user_favorites_user_id ON user_favorites(user_id);

-- Create index on station_id for analytics
CREATE INDEX idx_user_favorites_station_id ON user_favorites(station_id);

-- Create index on created_at for sorting
CREATE INDEX idx_user_favorites_created_at ON user_favorites(created_at DESC);
