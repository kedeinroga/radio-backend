-- ============================================================
--  000022_rls_and_search_path_hardening.down.sql
--  Rollback completo — ejecutado por: migrate down 1
-- ============================================================

BEGIN;

-- 6. Usuarios y seguridad
DROP POLICY IF EXISTS "users_own_record"    ON public.users;
DROP POLICY IF EXISTS "users_update_own"    ON public.users;
DROP POLICY IF EXISTS "user_favorites_own"  ON public.user_favorites;
DROP POLICY IF EXISTS "sessions_own"        ON public.sessions;
DROP POLICY IF EXISTS "stream_sessions_own" ON public.stream_sessions;

ALTER TABLE public.users                DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_favorites       DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.sessions             DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.stream_sessions      DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.security_events      DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.login_attempts       DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.brute_force_attempts DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.rate_limit_tracking  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.token_blacklist      DISABLE ROW LEVEL SECURITY;

-- 5. Particiones
ALTER TABLE public.station_plays_2026_01   DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_02   DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_03   DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_04   DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_05   DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_plays_2026_06   DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_01    DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_02    DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_03    DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_04    DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_05    DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.request_logs_2026_06    DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_01  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_02  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_03  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_04  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_05  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.search_queries_2026_06  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_01  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_02  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_03  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_04  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_05  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_impressions_2026_06  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_clicks               DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.ad_audit_log            DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_ad_profiles        DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.materialized_view_refresh_log DISABLE ROW LEVEL SECURITY;

-- 4. Tablas públicas
DROP POLICY IF EXISTS "stations_public_read"             ON public.stations;
DROP POLICY IF EXISTS "station_translations_public_read" ON public.station_translations;
DROP POLICY IF EXISTS "station_search_cache_public_read" ON public.station_search_cache;
DROP POLICY IF EXISTS "station_play_stats_public_read"   ON public.station_play_stats;
DROP POLICY IF EXISTS "seo_tag_stats_public_read"        ON public.seo_tag_stats;
DROP POLICY IF EXISTS "seo_country_stats_public_read"    ON public.seo_country_stats;
DROP POLICY IF EXISTS "ads_public_read"                  ON public.advertisements;

ALTER TABLE public.stations              DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_translations  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_search_cache  DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.station_play_stats    DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.seo_tag_stats         DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.seo_country_stats     DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.advertisements        DISABLE ROW LEVEL SECURITY;

-- 3. Revertir search_path de funciones
ALTER FUNCTION public.update_updated_at()                RESET search_path;
ALTER FUNCTION public.stations_search_vector_update()    RESET search_path;
ALTER FUNCTION public.cleanup_expired_sessions()         RESET search_path;
ALTER FUNCTION public.unlock_expired_accounts()          RESET search_path;
ALTER FUNCTION public.cleanup_expired_search_cache()     RESET search_path;
ALTER FUNCTION public.prevent_privilege_escalation()     RESET search_path;
ALTER FUNCTION public.audit_role_changes()               RESET search_path;
ALTER FUNCTION public.sanitize_token_in_logs()           RESET search_path;
ALTER FUNCTION public.prevent_partition_overflow()       RESET search_path;
ALTER FUNCTION public.cleanup_rate_limits()              RESET search_path;
ALTER FUNCTION public.create_partition_if_not_exists()   RESET search_path;
ALTER FUNCTION public.cleanup_old_partitions(INTEGER)               RESET search_path;
ALTER FUNCTION public.list_partition_status()                       RESET search_path;
ALTER FUNCTION public.check_missing_partitions(INTEGER)             RESET search_path;
ALTER FUNCTION public.refresh_materialized_view_safe(TEXT, BOOLEAN) RESET search_path;
ALTER FUNCTION public.refresh_all_analytics_views()      RESET search_path;
ALTER FUNCTION public.get_refresh_statistics(INTEGER)    RESET search_path;
ALTER FUNCTION public.refresh_all_views_with_logging()   RESET search_path;
ALTER FUNCTION public.refresh_all_seo_views()            RESET search_path;
ALTER FUNCTION public.refresh_all_materialized_views()   RESET search_path;
ALTER FUNCTION public.cleanup_expired_tokens()           RESET search_path;
ALTER FUNCTION public.is_token_blacklisted(VARCHAR)      RESET search_path;
ALTER FUNCTION public.reset_ad_counters()                RESET search_path;
ALTER FUNCTION public.detect_suspicious_clicks()         RESET search_path;
ALTER FUNCTION public.log_ad_change()                    RESET search_path;
ALTER FUNCTION public.refresh_stream_analytics()         RESET search_path;

-- 2. pg_trgm de vuelta a public (solo si fue movida a extensions por esta migración)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'extensions')
     AND (
       SELECT nspname
       FROM pg_extension
       JOIN pg_namespace ON extnamespace = pg_namespace.oid
       WHERE extname = 'pg_trgm'
     ) = 'extensions'
  THEN
    ALTER EXTENSION pg_trgm SET SCHEMA public;
  END IF;
END $$;

-- 1. Vista users_safe — restaurar sin security_invoker
DROP VIEW IF EXISTS public.users_safe;
CREATE VIEW public.users_safe AS
  SELECT id, email, user_type, created_at, updated_at
  FROM public.users;

COMMIT;
