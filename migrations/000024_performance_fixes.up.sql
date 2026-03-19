-- ============================================================
--  000024_performance_fixes.up.sql
--  1. Auth RLS initplan: reemplaza auth.uid() con (SELECT auth.uid())
--     para que la función se evalúe una sola vez por query, no por fila.
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
--     OMITIDO: Los índices idx1 son partition indexes adjuntos al índice
--     padre idx_impressions_ip_created y no pueden eliminarse de forma
--     independiente. El advisor los identifica como duplicados, pero en
--     realidad son los índices de partición gestionados por PostgreSQL.
-- ────────────────────────────────────────────────

COMMIT;
