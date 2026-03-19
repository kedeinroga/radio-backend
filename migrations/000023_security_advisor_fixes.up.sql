-- ============================================================
--  000023_security_advisor_fixes.up.sql
--  Corrige los dos items pendientes del advisor de Supabase:
--  1. users_safe con security_invoker explícito (requiere PG15+)
--  2. RLS en schema_migrations (tabla interna de golang-migrate)
-- ============================================================

BEGIN;

-- ────────────────────────────────────────────────
--  1. Vista users_safe — security_invoker explícito
--     WITH (security_invoker = true) existe desde PG15.
--     En local (PG14) se omite; el advisor corre en Supabase (PG15+).
-- ────────────────────────────────────────────────
DO $$
BEGIN
  IF current_setting('server_version_num')::INTEGER >= 150000 THEN
    DROP VIEW IF EXISTS public.users_safe;
    EXECUTE '
      CREATE VIEW public.users_safe
        WITH (security_invoker = true)
      AS
        SELECT id, email, user_type, created_at, updated_at
        FROM public.users
    ';
    COMMENT ON VIEW public.users_safe IS
      'Vista segura sin password_hash. security_invoker=true: el caller necesita SELECT en public.users.';
  END IF;
END $$;

-- ────────────────────────────────────────────────
--  2. RLS en schema_migrations (tabla de golang-migrate)
--     Sin política = solo service_role/superuser puede acceder.
--     El migrate CLI usa service_role → bypasea RLS, no se rompe.
-- ────────────────────────────────────────────────
ALTER TABLE public.schema_migrations ENABLE ROW LEVEL SECURITY;

COMMIT;
