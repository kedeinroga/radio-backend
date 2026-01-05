-- Migration: Create user_ad_profiles table
-- Description: Perfiles de usuario para publicidad y suscripciones premium

CREATE TABLE user_ad_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Preferencias inferidas (actualizadas periódicamente)
  favorite_genres TEXT[],
  favorite_countries TEXT[],
  listening_times JSONB,

  -- Comportamiento publicitario
  total_ad_impressions INTEGER NOT NULL DEFAULT 0,
  total_ad_clicks INTEGER NOT NULL DEFAULT 0,
  last_ad_shown_at TIMESTAMP,

  -- Frequency capping (límites de frecuencia)
  ads_shown_today INTEGER NOT NULL DEFAULT 0,
  ads_shown_today_reset_at DATE NOT NULL DEFAULT CURRENT_DATE,
  ads_shown_this_hour INTEGER NOT NULL DEFAULT 0,
  ads_shown_this_hour_reset_at TIMESTAMP NOT NULL DEFAULT date_trunc('hour', NOW()),

  -- Premium subscription
  is_premium BOOLEAN NOT NULL DEFAULT false,
  premium_since TIMESTAMP,
  premium_until TIMESTAMP,
  stripe_customer_id VARCHAR(255),
  stripe_subscription_id VARCHAR(255),

  -- Timestamps
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Índices
CREATE INDEX idx_user_ad_profiles_user ON user_ad_profiles(user_id);
CREATE INDEX idx_user_ad_profiles_premium ON user_ad_profiles(is_premium);
CREATE INDEX idx_user_ad_profiles_last_ad ON user_ad_profiles(last_ad_shown_at);
CREATE INDEX idx_user_ad_profiles_stripe_customer ON user_ad_profiles(stripe_customer_id);

-- Trigger para updated_at
CREATE TRIGGER update_user_ad_profiles_updated_at
  BEFORE UPDATE ON user_ad_profiles
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();

-- Función para resetear counters automáticamente
CREATE OR REPLACE FUNCTION reset_ad_counters()
RETURNS TRIGGER AS $$
BEGIN
  -- Reset daily counter si es un nuevo día
  IF NEW.ads_shown_today_reset_at < CURRENT_DATE THEN
    NEW.ads_shown_today = 0;
    NEW.ads_shown_today_reset_at = CURRENT_DATE;
  END IF;

  -- Reset hourly counter si es una nueva hora
  IF NEW.ads_shown_this_hour_reset_at < date_trunc('hour', NOW()) THEN
    NEW.ads_shown_this_hour = 0;
    NEW.ads_shown_this_hour_reset_at = date_trunc('hour', NOW());
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER reset_ad_counters_trigger
  BEFORE UPDATE ON user_ad_profiles
  FOR EACH ROW
  EXECUTE FUNCTION reset_ad_counters();

-- Comentarios
COMMENT ON TABLE user_ad_profiles IS 'Perfiles de usuario para ads y premium';
COMMENT ON COLUMN user_ad_profiles.listening_times IS 'JSONB con formato {"09": 45, "10": 32} (hora: cantidad)';
COMMENT ON COLUMN user_ad_profiles.ads_shown_today IS 'Contador de ads mostrados hoy (resetea automáticamente)';
COMMENT ON COLUMN user_ad_profiles.ads_shown_this_hour IS 'Contador de ads mostrados esta hora (resetea automáticamente)';
COMMENT ON COLUMN user_ad_profiles.is_premium IS 'Si el usuario tiene suscripción premium activa';
