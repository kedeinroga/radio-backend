-- Migration: Create ad_campaigns table
-- Description: Tabla principal de campañas publicitarias

CREATE TABLE ad_campaigns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  advertiser_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Campaign info
  name VARCHAR(255) NOT NULL,
  description TEXT,

  -- Budget (en centavos para evitar problemas de floating point)
  total_budget_cents INTEGER NOT NULL CHECK (total_budget_cents > 0),
  daily_budget_cents INTEGER CHECK (daily_budget_cents IS NULL OR daily_budget_cents > 0),
  spent_cents INTEGER NOT NULL DEFAULT 0,

  -- Scheduling
  start_date TIMESTAMP NOT NULL,
  end_date TIMESTAMP NOT NULL,

  -- Status
  status VARCHAR(50) NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'active', 'paused', 'completed', 'expired')),

  -- Timestamps
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

  -- Constraints
  CONSTRAINT valid_campaign_dates CHECK (end_date > start_date)
);

-- Índices para mejorar performance
CREATE INDEX idx_ad_campaigns_advertiser ON ad_campaigns(advertiser_id);
CREATE INDEX idx_ad_campaigns_status ON ad_campaigns(status);
CREATE INDEX idx_ad_campaigns_dates ON ad_campaigns(start_date, end_date);

-- Trigger para updated_at automático
CREATE TRIGGER update_ad_campaigns_updated_at
  BEFORE UPDATE ON ad_campaigns
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();

-- Comentarios
COMMENT ON TABLE ad_campaigns IS 'Campañas publicitarias con presupuesto y scheduling';
COMMENT ON COLUMN ad_campaigns.total_budget_cents IS 'Presupuesto total en centavos (100 = $1.00)';
COMMENT ON COLUMN ad_campaigns.spent_cents IS 'Cantidad gastada en centavos';
