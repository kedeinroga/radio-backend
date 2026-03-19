-- ============================================================
--  000024_performance_fixes.up.sql
--  1. Auth RLS initplan: reemplaza auth.uid() con (SELECT auth.uid())
--     para que la función se evalúe una sola vez por query, no por fila.
--  2. Elimina índices duplicados en particiones de ad_impressions.
--     NOTA: Se usa DROP INDEX (sin CONCURRENTLY) porque este archivo
--     corre dentro de una transacción gestionada por golang-migrate.
-- ============================================================

BEGIN;

-- ────────────────────────────────────────────────
--  1. RLS initplan — solo en Supabase (schema auth existe)
-- ────────────────────────────────────────────────
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth') THEN
    EXECUTE 'ALTER POLICY "users_own_record"    ON public.users          USING ((SELECT auth.uid()) = id)';
    EXECUTE 'ALTER POLICY "users_update_own"    ON public.users          USING ((SELECT auth.uid()) = id)';
    EXECUTE 'ALTER POLICY "user_favorites_own"  ON public.user_favorites USING ((SELECT auth.uid()) = user_id)';
    EXECUTE 'ALTER POLICY "sessions_own"        ON public.sessions       USING ((SELECT auth.uid()) = user_id)';
    EXECUTE 'ALTER POLICY "stream_sessions_own" ON public.stream_sessions USING ((SELECT auth.uid()) = user_id)';
  END IF;
END $$;

-- ────────────────────────────────────────────────
--  2. Índices duplicados en particiones de ad_impressions
--     Detectados por el advisor de Supabase (solo en producción).
--     En local el setup de particiones difiere — se omite.
-- ────────────────────────────────────────────────
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth') THEN
    DROP INDEX IF EXISTS public.ad_impressions_2026_01_ip_address_created_at_idx1;
    DROP INDEX IF EXISTS public.ad_impressions_2026_02_ip_address_created_at_idx1;
    DROP INDEX IF EXISTS public.ad_impressions_2026_03_ip_address_created_at_idx1;
    DROP INDEX IF EXISTS public.ad_impressions_2026_04_ip_address_created_at_idx1;
    DROP INDEX IF EXISTS public.ad_impressions_2026_05_ip_address_created_at_idx1;
    DROP INDEX IF EXISTS public.ad_impressions_2026_06_ip_address_created_at_idx1;
  END IF;
END $$;

COMMIT;
