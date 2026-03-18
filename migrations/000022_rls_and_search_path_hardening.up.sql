-- ============================================================
--  000022_rls_and_search_path_hardening.up.sql
--  Seguridad: RLS en todas las tablas, search_path fijo en
--  funciones, y vista users_safe recreada sin password_hash.
--
--  NOTA: El backend usa service_role key → bypasea RLS.
--  Estas políticas protegen accesos directos vía PostgREST
--  o clientes Supabase que usan anon/authenticated keys.
-- ============================================================

BEGIN;

-- ────────────────────────────────────────────────
--  1. Vista users_safe → security_invoker (corrige SECURITY DEFINER implícito)
-- ────────────────────────────────────────────────
DROP VIEW IF EXISTS public.users_safe;
CREATE VIEW public.users_safe AS
  SELECT id, email, user_type, created_at, updated_at
  FROM public.users;
-- Las vistas son SECURITY INVOKER por defecto en todas las versiones de PostgreSQL.
-- El caller necesita SELECT en public.users para usar esta vista.

COMMENT ON VIEW public.users_safe IS
  'Vista segura de usuarios sin password_hash. SECURITY INVOKER por defecto: el llamador necesita SELECT en public.users.';

-- ────────────────────────────────────────────────
--  2. pg_trgm → schema extensions (solo si está en public)
-- ────────────────────────────────────────────────
DO $$
BEGIN
  -- Solo mover pg_trgm si el schema extensions existe (Supabase) y pg_trgm está en public
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'extensions')
     AND (
       SELECT nspname
       FROM pg_extension
       JOIN pg_namespace ON extnamespace = pg_namespace.oid
       WHERE extname = 'pg_trgm'
     ) = 'public'
  THEN
    ALTER EXTENSION pg_trgm SET SCHEMA extensions;
  END IF;
END $$;

-- ────────────────────────────────────────────────
--  3. Fijar search_path en todas las funciones
--     (evita hijacking via schema injection)
-- ────────────────────────────────────────────────
ALTER FUNCTION public.update_updated_at()                  SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.stations_search_vector_update()      SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.cleanup_expired_sessions()           SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.unlock_expired_accounts()            SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.cleanup_expired_search_cache()       SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.prevent_privilege_escalation()       SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.audit_role_changes()                 SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.sanitize_token_in_logs()             SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.prevent_partition_overflow()         SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.cleanup_rate_limits()                SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.create_partition_if_not_exists()     SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.cleanup_old_partitions(INTEGER)                SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.list_partition_status()                        SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.check_missing_partitions(INTEGER)              SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.refresh_materialized_view_safe(TEXT, BOOLEAN)  SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.refresh_all_analytics_views()                  SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.get_refresh_statistics(INTEGER)                SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.refresh_all_views_with_logging()     SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.refresh_all_seo_views()              SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.refresh_all_materialized_views()     SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.cleanup_expired_tokens()             SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.is_token_blacklisted(VARCHAR)        SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.reset_ad_counters()                  SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.detect_suspicious_clicks()           SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.log_ad_change()                      SET search_path = public, extensions, pg_catalog;
ALTER FUNCTION public.refresh_stream_analytics()           SET search_path = public, extensions, pg_catalog;

-- ────────────────────────────────────────────────
--  4. RLS — tablas de lectura pública
-- ────────────────────────────────────────────────
CREATE POLICY "stations_public_read"            ON public.stations            FOR SELECT USING (true);
CREATE POLICY "station_translations_public_read" ON public.station_translations FOR SELECT USING (true);
CREATE POLICY "station_search_cache_public_read" ON public.station_search_cache FOR SELECT USING (true);
CREATE POLICY "station_play_stats_public_read"  ON public.station_play_stats   FOR SELECT USING (true);
CREATE POLICY "seo_tag_stats_public_read"       ON public.seo_tag_stats        FOR SELECT USING (true);
CREATE POLICY "seo_country_stats_public_read"   ON public.seo_country_stats    FOR SELECT USING (true);
CREATE POLICY "ads_public_read"                 ON public.advertisements        FOR SELECT USING (true);

ALTER TABLE public.stations              ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_translations  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_search_cache  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_play_stats    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.seo_tag_stats         ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.seo_country_stats     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.advertisements        ENABLE ROW LEVEL SECURITY;

-- ────────────────────────────────────────────────
--  5. RLS — particiones (solo service_role puede acceder, sin políticas permisivas)
-- ────────────────────────────────────────────────
ALTER TABLE public.station_plays_2026_01      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_02      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_03      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_04      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_05      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_06      ENABLE ROW LEVEL SECURITY;

ALTER TABLE public.request_logs_2026_01       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_02       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_03       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_04       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_05       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_06       ENABLE ROW LEVEL SECURITY;

ALTER TABLE public.search_queries_2026_01     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_02     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_03     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_04     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_05     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_06     ENABLE ROW LEVEL SECURITY;

ALTER TABLE public.ad_impressions_2026_01     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_02     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_03     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_04     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_05     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_06     ENABLE ROW LEVEL SECURITY;

ALTER TABLE public.ad_clicks                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_audit_log               ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_ad_profiles           ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.materialized_view_refresh_log ENABLE ROW LEVEL SECURITY;

-- ────────────────────────────────────────────────
--  6. RLS — tablas de usuarios y seguridad
--     Políticas: cada usuario solo ve/modifica sus propios datos
-- ────────────────────────────────────────────────
-- auth.uid() es exclusivo de Supabase; en local se omiten estas políticas
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth') THEN
    EXECUTE 'CREATE POLICY "users_own_record"    ON public.users         FOR SELECT USING (auth.uid() = id)';
    EXECUTE 'CREATE POLICY "users_update_own"    ON public.users         FOR UPDATE USING (auth.uid() = id)';
    EXECUTE 'CREATE POLICY "user_favorites_own"  ON public.user_favorites FOR ALL    USING (auth.uid() = user_id)';
    EXECUTE 'CREATE POLICY "sessions_own"        ON public.sessions       FOR ALL    USING (auth.uid() = user_id)';
    EXECUTE 'CREATE POLICY "stream_sessions_own" ON public.stream_sessions FOR ALL   USING (auth.uid() = user_id)';
  END IF;
END $$;

ALTER TABLE public.users                ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_favorites       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.sessions             ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.stream_sessions      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.security_events      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.login_attempts       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.brute_force_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.rate_limit_tracking  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.token_blacklist      ENABLE ROW LEVEL SECURITY;

COMMIT;
