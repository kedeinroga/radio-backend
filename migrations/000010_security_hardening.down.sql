-- ===============================================
-- Rollback 000010: Security Hardening
-- ===============================================

DROP TRIGGER IF EXISTS prevent_user_type_escalation ON users;
DROP TRIGGER IF EXISTS audit_user_role_changes ON users;
DROP TRIGGER IF EXISTS sanitize_security_event_logs ON security_events;
DROP TRIGGER IF EXISTS check_partition_bounds_plays ON station_plays;
DROP TRIGGER IF EXISTS check_partition_bounds_logs ON request_logs;
DROP TRIGGER IF EXISTS check_partition_bounds_searches ON search_queries;

DROP FUNCTION IF EXISTS prevent_privilege_escalation();
DROP FUNCTION IF EXISTS audit_role_changes();
DROP FUNCTION IF EXISTS sanitize_token_in_logs();
DROP FUNCTION IF EXISTS prevent_partition_overflow();
DROP FUNCTION IF EXISTS cleanup_rate_limits();

ALTER TABLE stations DROP CONSTRAINT IF EXISTS check_name_safe;
ALTER TABLE station_translations DROP CONSTRAINT IF EXISTS check_title_safe;

DROP VIEW IF EXISTS users_safe;
DROP TABLE IF EXISTS brute_force_attempts;
DROP TABLE IF EXISTS rate_limit_tracking;
