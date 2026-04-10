-- ============================================================
--  000026_drop_orphan_partition_function.up.sql
--
--  Removes the orphaned trigger function create_partition_if_not_exists().
--
--  Background: migration 000011 installed this as a BEFORE INSERT trigger
--  on request_logs, station_plays, and search_queries. In PostgreSQL 13+
--  those triggers fired on the target partition (TG_TABLE_NAME =
--  'request_logs_2026_03'), causing it to attempt
--  CREATE TABLE ... PARTITION OF "request_logs_2026_03", which always
--  failed with a lock conflict.
--
--  Migration 000025 already dropped the three triggers. This migration
--  removes the now-unused function so it cannot be accidentally reattached.
-- ============================================================

DROP FUNCTION IF EXISTS public.create_partition_if_not_exists();
