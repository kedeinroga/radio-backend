-- ============================================================
--  000024_performance_fixes.down.sql
--  Revierte las políticas a auth.uid() directo.
--  Los índices duplicados NO se recrean (eran inútiles).
-- ============================================================

BEGIN;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth') THEN
    EXECUTE 'ALTER POLICY "users_own_record"    ON public.users          USING (auth.uid() = id)';
    EXECUTE 'ALTER POLICY "users_update_own"    ON public.users          USING (auth.uid() = id)';
    EXECUTE 'ALTER POLICY "user_favorites_own"  ON public.user_favorites USING (auth.uid() = user_id)';
    EXECUTE 'ALTER POLICY "sessions_own"        ON public.sessions       USING (auth.uid() = user_id)';
    EXECUTE 'ALTER POLICY "stream_sessions_own" ON public.stream_sessions USING (auth.uid() = user_id)';
  END IF;
END $$;

COMMIT;
