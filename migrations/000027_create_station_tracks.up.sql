-- Create station_tracks table for "now playing" / "recently played" history.
-- Populated by the now-playing poll job (and optionally the audio proxy) from ICY metadata.
CREATE TABLE IF NOT EXISTS station_tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id TEXT NOT NULL, -- No foreign key porque las estaciones viven en Redis/radio-browser

    -- Track metadata extraída del StreamTitle ICY
    raw_title TEXT NOT NULL,
    artist TEXT,
    title TEXT,

    -- Origen de la captura
    source TEXT NOT NULL DEFAULT 'poll' CHECK (source IN ('poll', 'proxy')),

    -- Timestamps
    played_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índice principal: sirve "now playing" (LIMIT 1) y "recently played" por estación
CREATE INDEX idx_station_tracks_station_played ON station_tracks(station_id, played_at DESC);

-- Índice para la retención (limpieza por antigüedad)
CREATE INDEX idx_station_tracks_played_at ON station_tracks(played_at);

-- Comentarios para documentación
COMMENT ON TABLE station_tracks IS 'Historial de canciones (now playing / recently played) capturado del metadata ICY de los streams';
COMMENT ON COLUMN station_tracks.raw_title IS 'StreamTitle ICY crudo, ej. "Artist - Song"';
COMMENT ON COLUMN station_tracks.source IS 'Mecanismo de captura: poll (job de sondeo) o proxy (oyente autenticado)';
COMMENT ON COLUMN station_tracks.played_at IS 'Momento en que se detectó la pista sonando';
