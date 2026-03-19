-- ============================================================
--  000023_security_advisor_fixes.down.sql
-- ============================================================

BEGIN;

-- 2. Revertir RLS en schema_migrations
ALTER TABLE public.schema_migrations DISABLE ROW LEVEL SECURITY;

-- 1. Revertir users_safe a vista sin opción explícita
DO $$
BEGIN
  IF current_setting('server_version_num')::INTEGER >= 150000 THEN
    DROP VIEW IF EXISTS public.users_safe;
    CREATE VIEW public.users_safe AS
      SELECT id, email, user_type, created_at, updated_at
      FROM public.users;
  END IF;
END $$;

COMMIT;
