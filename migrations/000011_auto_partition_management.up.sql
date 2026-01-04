-- ===============================================
-- Migración 000011: Automatic Partition Management
-- ===============================================
-- Auto-creación de particiones para evitar fallos futuros

-- ============================================
-- 1. FUNCIÓN: Auto-crear particiones
-- ============================================
CREATE OR REPLACE FUNCTION create_partition_if_not_exists()
RETURNS TRIGGER AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    -- Obtener fecha de la partición (truncada al mes)
    partition_date := DATE_TRUNC('month', NEW.created_at);
    partition_name := TG_TABLE_NAME || '_' || TO_CHAR(partition_date, 'YYYY_MM');

    -- Verificar si la partición ya existe
    IF NOT EXISTS (
        SELECT 1
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename = partition_name
    ) THEN
        -- Calcular rangos de fechas
        start_date := partition_date;
        end_date := partition_date + INTERVAL '1 month';

        -- Crear partición
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF %I
             FOR VALUES FROM (%L) TO (%L)',
            partition_name, TG_TABLE_NAME, start_date, end_date
        );

        RAISE NOTICE 'Auto-created partition: % for range [%, %)',
            partition_name, start_date, end_date;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 2. TRIGGERS: Aplicar a todas las tablas particionadas
-- ============================================

-- Trigger para station_plays
CREATE TRIGGER auto_create_partition_station_plays
    BEFORE INSERT ON station_plays
    FOR EACH ROW
    EXECUTE FUNCTION create_partition_if_not_exists();

-- Trigger para request_logs
CREATE TRIGGER auto_create_partition_request_logs
    BEFORE INSERT ON request_logs
    FOR EACH ROW
    EXECUTE FUNCTION create_partition_if_not_exists();

-- Trigger para search_queries
CREATE TRIGGER auto_create_partition_search_queries
    BEFORE INSERT ON search_queries
    FOR EACH ROW
    EXECUTE FUNCTION create_partition_if_not_exists();

-- ============================================
-- 3. FUNCIÓN: Cleanup de particiones antiguas
-- ============================================
CREATE OR REPLACE FUNCTION cleanup_old_partitions(
    retention_months INTEGER DEFAULT 12
)
RETURNS TABLE(
    partition_name TEXT,
    partition_date DATE,
    action TEXT
) AS $$
DECLARE
    partition_record RECORD;
    extracted_date DATE;
BEGIN
    FOR partition_record IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename ~ '^(station_plays|request_logs|search_queries)_\d{4}_\d{2}$'
    LOOP
        BEGIN
            -- Extraer fecha del nombre de la partición
            extracted_date := TO_DATE(
                SUBSTRING(partition_record.tablename FROM '\d{4}_\d{2}$'),
                'YYYY_MM'
            );

            -- Verificar si debe ser eliminada
            IF extracted_date < CURRENT_DATE - make_interval(months => retention_months) THEN
                -- Eliminar partición antigua
                EXECUTE format('DROP TABLE IF EXISTS %I CASCADE', partition_record.tablename);

                partition_name := partition_record.tablename;
                partition_date := extracted_date;
                action := 'DROPPED';
                RETURN NEXT;

                RAISE NOTICE 'Dropped old partition: % (date: %)',
                    partition_record.tablename, extracted_date;
            END IF;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE WARNING 'Error processing partition %: %',
                    partition_record.tablename, SQLERRM;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 4. FUNCIÓN: Listar estado de particiones
-- ============================================
CREATE OR REPLACE FUNCTION list_partition_status()
RETURNS TABLE(
    partition_name TEXT,
    table_name TEXT,
    partition_date DATE,
    row_count BIGINT,
    size_mb NUMERIC,
    index_size_mb NUMERIC,
    total_size_mb NUMERIC
) AS $$
DECLARE
    partition_rec RECORD;
    base_table_name TEXT;
BEGIN
    FOR partition_rec IN
        SELECT
            c.relname::TEXT as pname,
            parent.relname::TEXT as tname
        FROM pg_class c
        JOIN pg_inherits i ON c.oid = i.inhrelid
        JOIN pg_class parent ON i.inhparent = parent.oid
        WHERE parent.relname IN ('station_plays', 'request_logs', 'search_queries')
        ORDER BY c.relname
    LOOP
        partition_name := partition_rec.pname;
        table_name := partition_rec.tname;
        partition_date := TO_DATE(SUBSTRING(partition_rec.pname FROM '\d{4}_\d{2}$'), 'YYYY_MM');

        EXECUTE format('SELECT count(*) FROM %I', partition_rec.pname) INTO row_count;

        EXECUTE format('SELECT pg_table_size(%L::regclass) / 1024.0 / 1024.0', partition_rec.pname) INTO size_mb;
        EXECUTE format('SELECT pg_indexes_size(%L::regclass) / 1024.0 / 1024.0', partition_rec.pname) INTO index_size_mb;
        EXECUTE format('SELECT pg_total_relation_size(%L::regclass) / 1024.0 / 1024.0', partition_rec.pname) INTO total_size_mb;

        RETURN NEXT;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 5. FUNCIÓN: Verificar particiones faltantes
-- ============================================
CREATE OR REPLACE FUNCTION check_missing_partitions(
    months_ahead INTEGER DEFAULT 3
)
RETURNS TABLE(
    table_name TEXT,
    missing_month DATE,
    partition_name TEXT,
    status TEXT
) AS $$
DECLARE
    base_table TEXT;
    check_date DATE;
    expected_partition TEXT;
    i INTEGER;
BEGIN
    -- Verificar para cada tabla base
    FOREACH base_table IN ARRAY ARRAY['station_plays', 'request_logs', 'search_queries']
    LOOP
        -- Verificar los próximos N meses
        FOR i IN 0..months_ahead LOOP
            check_date := DATE_TRUNC('month', CURRENT_DATE + make_interval(months => i));
            expected_partition := base_table || '_' || TO_CHAR(check_date, 'YYYY_MM');

            -- Verificar si existe
            IF NOT EXISTS (
                SELECT 1
                FROM pg_tables
                WHERE schemaname = 'public'
                AND tablename = expected_partition
            ) THEN
                table_name := base_table;
                missing_month := check_date;
                partition_name := expected_partition;
                status := 'MISSING';
                RETURN NEXT;
            END IF;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Comentarios
-- ============================================
COMMENT ON FUNCTION create_partition_if_not_exists() IS
'Auto-crea particiones mensuales cuando se intenta insertar datos en una partición inexistente';

COMMENT ON FUNCTION cleanup_old_partitions(INTEGER) IS
'Elimina particiones más antiguas que el período de retención especificado (default: 12 meses)';

COMMENT ON FUNCTION list_partition_status() IS
'Lista todas las particiones existentes con su tamaño y número de filas';

COMMENT ON FUNCTION check_missing_partitions(INTEGER) IS
'Verifica si faltan particiones para los próximos N meses (default: 3)';

-- ============================================
-- Verificación inicial
-- ============================================
-- NOTA: La verificación se puede hacer manualmente vía endpoint REST:
-- GET /api/v1/admin/maintenance/check-partitions
-- GET /api/v1/admin/maintenance/partition-status
-- O directamente en SQL:
-- SELECT * FROM check_missing_partitions(3);
-- SELECT * FROM list_partition_status();
