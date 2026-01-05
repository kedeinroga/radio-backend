-- Migration: Create ad_clicks table
-- Description: Tabla de clicks en anuncios

CREATE TABLE ad_clicks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  impression_id UUID NOT NULL,
  advertisement_id UUID NOT NULL REFERENCES advertisements(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  session_id UUID NOT NULL,

  -- Click details
  click_url TEXT NOT NULL,
  click_position_x INTEGER,
  click_position_y INTEGER,

  -- Conversion tracking
  converted BOOLEAN NOT NULL DEFAULT false,
  conversion_value_cents INTEGER,

  -- Timestamp
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Índices para queries de analytics
CREATE INDEX idx_clicks_impression ON ad_clicks(impression_id);
CREATE INDEX idx_clicks_ad ON ad_clicks(advertisement_id, created_at);
CREATE INDEX idx_clicks_user ON ad_clicks(user_id, created_at);
CREATE INDEX idx_clicks_session ON ad_clicks(session_id);

-- Comentarios
COMMENT ON TABLE ad_clicks IS 'Clicks en anuncios con tracking de conversiones';
COMMENT ON COLUMN ad_clicks.impression_id IS 'FK a ad_impressions (sin constraint por partitioning)';
COMMENT ON COLUMN ad_clicks.converted IS 'Si el usuario completó una acción deseada post-click';
COMMENT ON COLUMN ad_clicks.conversion_value_cents IS 'Valor monetario de la conversión en centavos';
