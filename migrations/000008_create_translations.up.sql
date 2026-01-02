-- ===============================================
-- Migración 000008: Station Translations
-- ===============================================
-- Tabla de traducciones de estaciones para SEO multiidioma

CREATE TABLE station_translations (
    station_id VARCHAR(255) NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    language_code CHAR(2) NOT NULL, -- ISO 639-1
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    keywords TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (station_id, language_code)
);

-- Índices
CREATE INDEX idx_station_translations_station ON station_translations(station_id);
CREATE INDEX idx_station_translations_lang ON station_translations(language_code);
CREATE INDEX idx_station_translations_updated ON station_translations(updated_at DESC);

-- Trigger para updated_at
CREATE TRIGGER station_translations_updated_at
    BEFORE UPDATE ON station_translations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Comentarios
COMMENT ON TABLE station_translations IS 'Multilingual SEO metadata for stations';
COMMENT ON COLUMN station_translations.language_code IS 'ISO 639-1 language code (e.g., en, es, fr)';
COMMENT ON COLUMN station_translations.keywords IS 'SEO keywords array for this language';
