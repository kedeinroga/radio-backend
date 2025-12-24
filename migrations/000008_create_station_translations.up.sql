-- Tabla de traducciones de metadata SEO para estaciones
CREATE TABLE IF NOT EXISTS station_translations (
    station_id VARCHAR(255) NOT NULL,
    language_code VARCHAR(5) NOT NULL,
    
    -- SEO Metadata traducido
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    keywords TEXT[], -- Array de keywords por idioma
    
    -- Metadata adicional
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Clave primaria compuesta
    PRIMARY KEY (station_id, language_code),
    
    -- Foreign key a stations (opcional, por si quieres integridad referencial)
    CONSTRAINT fk_station_translations_station 
        FOREIGN KEY (station_id) 
        REFERENCES stations(id) 
        ON DELETE CASCADE
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_station_translations_station_id 
    ON station_translations(station_id);

CREATE INDEX IF NOT EXISTS idx_station_translations_language 
    ON station_translations(language_code);

CREATE INDEX IF NOT EXISTS idx_station_translations_updated 
    ON station_translations(updated_at DESC);

-- Trigger para actualizar updated_at automáticamente
CREATE OR REPLACE FUNCTION update_station_translations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_station_translations_timestamp
    BEFORE UPDATE ON station_translations
    FOR EACH ROW
    EXECUTE FUNCTION update_station_translations_updated_at();

-- Insertar traducciones por defecto para idiomas principales
-- Esto es opcional, puedes popularlo después con un script

COMMENT ON TABLE station_translations IS 'Traducciones de metadata SEO para estaciones en múltiples idiomas';
COMMENT ON COLUMN station_translations.language_code IS 'Código de idioma ISO 639-1 (ej: es, en, fr, de)';
COMMENT ON COLUMN station_translations.title IS 'Título SEO traducido (máx 200 caracteres)';
COMMENT ON COLUMN station_translations.description IS 'Descripción SEO traducida (recomendado ~160 caracteres)';
COMMENT ON COLUMN station_translations.keywords IS 'Array de keywords SEO en el idioma específico';
