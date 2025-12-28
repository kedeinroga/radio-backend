CREATE TABLE IF NOT EXISTS login_attempts (
    email VARCHAR(255) PRIMARY KEY,
    failed_count INT NOT NULL DEFAULT 0,
    last_attempt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    unlock_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_login_attempts_unlock_at ON login_attempts(unlock_at) WHERE is_locked = TRUE;
CREATE INDEX idx_login_attempts_is_locked ON login_attempts(is_locked);
CREATE INDEX idx_login_attempts_last_attempt ON login_attempts(last_attempt);
