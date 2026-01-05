-- Migration: Create advertisements table
-- Description: Tabla principal de anuncios publicitarios

CREATE TABLE advertisements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id UUID NOT NULL REFERENCES ad_campaigns(id) ON DELETE CASCADE,

  -- Metadata
  title VARCHAR(255) NOT NULL,
  description TEXT,
  advertiser_name VARCHAR(255) NOT NULL,

  -- Formato y tipo de anuncio
  ad_format VARCHAR(50) NOT NULL CHECK (ad_format IN ('banner', 'interstitial', 'audio', 'native')),
  ad_type VARCHAR(50) NOT NULL CHECK (ad_type IN ('image', 'video', 'audio', 'html')),

  -- Assets (URLs de contenido)
  media_url TEXT NOT NULL,
  click_url TEXT NOT NULL,
  companion_banner_url TEXT,

  -- Dimensiones (para banners/display ads)
  width INTEGER,
  height INTEGER,

  -- Duración (para audio/video en segundos)
  duration_seconds INTEGER,

  -- Targeting (arrays para múltiples valores)
  target_countries TEXT[],
  target_genres TEXT[],
  target_languages TEXT[],
  target_devices TEXT[],

  -- Scheduling
  start_date TIMESTAMP NOT NULL,
  end_date TIMESTAMP NOT NULL,
  daily_budget_cents INTEGER CHECK (daily_budget_cents IS NULL OR daily_budget_cents > 0),
  total_budget_cents INTEGER NOT NULL CHECK (total_budget_cents > 0),

  -- Pricing (al menos uno debe estar configurado)
  cpm_rate_cents INTEGER CHECK (cpm_rate_cents IS NULL OR cpm_rate_cents > 0),
  cpc_rate_cents INTEGER CHECK (cpc_rate_cents IS NULL OR cpc_rate_cents > 0),

  -- Estado
  status VARCHAR(50) NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'active', 'paused', 'completed', 'expired')),
  priority INTEGER NOT NULL DEFAULT 0,

  -- Límites de impresiones
  max_impressions_per_user INTEGER,
  max_impressions_per_day INTEGER,

  -- Métricas (actualizadas en tiempo real)
  impressions_count INTEGER NOT NULL DEFAULT 0,
  clicks_count INTEGER NOT NULL DEFAULT 0,
  spend_cents INTEGER NOT NULL DEFAULT 0,

  -- Timestamps
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

  -- Constraints de validación
  CONSTRAINT valid_banner_dimensions CHECK (
    ad_format != 'banner' OR (width IS NOT NULL AND height IS NOT NULL)
  ),
  CONSTRAINT valid_audio_video_duration CHECK (
    ad_format NOT IN ('audio', 'video') OR duration_seconds IS NOT NULL
  ),
  CONSTRAINT valid_pricing_model CHECK (
    cpm_rate_cents IS NOT NULL OR cpc_rate_cents IS NOT NULL
  ),
  CONSTRAINT valid_ad_dates CHECK (end_date > start_date),
  CONSTRAINT prevent_budget_overflow CHECK (spend_cents <= total_budget_cents * 1.1)
);

-- Índices para queries rápidas
CREATE INDEX idx_ads_campaign ON advertisements(campaign_id);
CREATE INDEX idx_ads_format ON advertisements(ad_format);
CREATE INDEX idx_ads_status ON advertisements(status);
CREATE INDEX idx_ads_dates ON advertisements(start_date, end_date);
CREATE INDEX idx_ads_priority ON advertisements(priority DESC);
CREATE INDEX idx_ads_active ON advertisements(status, start_date, end_date)
  WHERE status = 'active';

-- Índices GIN para arrays de targeting (búsqueda eficiente en arrays)
CREATE INDEX idx_ads_countries ON advertisements USING GIN(target_countries);
CREATE INDEX idx_ads_genres ON advertisements USING GIN(target_genres);
CREATE INDEX idx_ads_languages ON advertisements USING GIN(target_languages);
CREATE INDEX idx_ads_devices ON advertisements USING GIN(target_devices);

-- Trigger para updated_at
CREATE TRIGGER update_advertisements_updated_at
  BEFORE UPDATE ON advertisements
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();

-- Comentarios
COMMENT ON TABLE advertisements IS 'Anuncios publicitarios con targeting y métricas';
COMMENT ON COLUMN advertisements.cpm_rate_cents IS 'Cost Per Mille (1000 impressions) en centavos';
COMMENT ON COLUMN advertisements.cpc_rate_cents IS 'Cost Per Click en centavos';
COMMENT ON COLUMN advertisements.priority IS 'Mayor valor = mayor prioridad de serving';
COMMENT ON COLUMN advertisements.target_countries IS 'Array de códigos ISO de países (ej: [''US'', ''MX'', ''ES''])';
COMMENT ON COLUMN advertisements.target_genres IS 'Array de géneros musicales (ej: [''rock'', ''pop''])';
