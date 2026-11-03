-- First run (migration A), a re-run with every entry present (migration B's
-- call), and the SHA-1 cost alone, each timed. Committed, so the re-run
-- sees the first run's entries.
\set ON_ERROR_STOP on
\timing on
SET ROLE nava_migrator;
BEGIN; SELECT audit_backfill_s8() AS first_run_added; COMMIT;
BEGIN; SELECT audit_backfill_s8() AS rerun_added; COMMIT;
RESET ROLE;
SELECT count(*) AS ids FROM (SELECT audit_backfill_id(tenant_id, id) FROM entity_definition_versions) s;
\timing off
SELECT count(*) AS entries, count(DISTINCT tenant_id) AS tenants_with_entries FROM audit_log;
SELECT count(*) AS migrator_policies FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles);
SELECT pg_size_pretty(pg_total_relation_size('audit_log_default')) AS audit_log_size;
