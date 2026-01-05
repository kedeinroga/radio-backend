-- Migration: Create ad_impressions table (partitioned)
-- Description: Tabla particionada de impresiones de anuncios

-- Tabla principal (particionada por mes para mejor performance)
CREATE TABLE ad_impressions (
  id UUID DEFAULT gen_random_uuid(),
  advertisement_id UUID NOT NULL,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  session_id UUID NOT NULL,

  -- Contexto de la impresión
  placement VARCHAR(100) NOT NULL,
  page_url TEXT,
  referrer TEXT,

  -- Geo (extraído del IP del usuario)
  country_code VARCHAR(2),
  region VARCHAR(100),
  city VARCHAR(100),
  ip_address INET,

  -- Device information
  device_type VARCHAR(50),
  os VARCHAR(50),
  browser VARCHAR(50),
  user_agent TEXT,

  -- Métricas de visibilidad
  impression_duration_ms INTEGER,
  viewable BOOLEAN NOT NULL DEFAULT true,

  -- Acciones posteriores
  clicked BOOLEAN NOT NULL DEFAULT false,
  converted BOOLEAN NOT NULL DEFAULT false,

  -- Timestamp
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),

  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Crear particiones para 2026 (6 meses iniciales)
CREATE TABLE ad_impressions_2026_01 PARTITION OF ad_impressions
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE ad_impressions_2026_02 PARTITION OF ad_impressions
  FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE ad_impressions_2026_03 PARTITION OF ad_impressions
  FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE ad_impressions_2026_04 PARTITION OF ad_impressions
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE ad_impressions_2026_05 PARTITION OF ad_impressions
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE ad_impressions_2026_06 PARTITION OF ad_impressions
  FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Índices (se aplican automáticamente a todas las particiones)
CREATE INDEX idx_impressions_ad ON ad_impressions(advertisement_id, created_at);
CREATE INDEX idx_impressions_user ON ad_impressions(user_id, created_at);
CREATE INDEX idx_impressions_session ON ad_impressions(session_id);
CREATE INDEX idx_impressions_placement ON ad_impressions(placement, created_at);
CREATE INDEX idx_impressions_country ON ad_impressions(country_code, created_at);
CREATE INDEX idx_impressions_ip ON ad_impressions(ip_address, created_at);

-- Comentarios
COMMENT ON TABLE ad_impressions IS 'Impresiones de anuncios (particionada por mes)';
COMMENT ON COLUMN ad_impressions.placement IS 'Ubicación del ad (home_banner, player_audio, etc)';
COMMENT ON COLUMN ad_impressions.viewable IS 'Si el ad fue realmente visible (>50% viewport, >1s)';
COMMENT ON COLUMN ad_impressions.impression_duration_ms IS 'Tiempo que el ad fue visible en milisegundos';

-- Nota: Las particiones nuevas deben crearse mensualmente
-- Usar el job partition_manager para automatizar esto
