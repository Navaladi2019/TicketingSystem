-- The commit-check probe in its plpgsql-locals form (S1, P5-11), as a generic
-- plan, as nava_app under RLS. Run 1: every index present (the plan the
-- trigger gets). Run 2: the R2 key and the subject index removed in a
-- rolled-back transaction, so the planner shows its best
-- (tenant_id, occurred_at)-prefix path and that path's cost.
\set ON_ERROR_STOP on
SET plan_cache_mode = force_generic_plan;
SET ROLE nava_app;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000000001', false);
PREPARE probe(uuid, timestamptz, xid8, text, uuid) AS
  SELECT 1 FROM audit_log WHERE tenant_id = $1 AND occurred_at = $2 AND txid = $3
    AND subject_type = $4 AND subject_id = $5 LIMIT 1;
\echo '--- all indexes, generic plan (the plan the trigger caches)'
EXPLAIN EXECUTE probe('00000000-0000-4000-8000-000000000001', now(), '1'::xid8, 'record', gen_random_uuid());
SET plan_cache_mode = force_custom_plan;
\echo '--- all indexes, custom plan (plpgsql''s first five calls)'
EXPLAIN EXECUTE probe('00000000-0000-4000-8000-000000000001', now(), '1'::xid8, 'record', gen_random_uuid());
SET plan_cache_mode = force_generic_plan;
RESET ROLE;
BEGIN;
ALTER TABLE audit_log DROP CONSTRAINT audit_log_commit_key;
DROP INDEX audit_log_subject;
SET LOCAL ROLE nava_app;
\echo '--- R2 key and subject index removed, generic (best prefix path)'
EXPLAIN EXECUTE probe('00000000-0000-4000-8000-000000000001', now(), '1'::xid8, 'record', gen_random_uuid());
SET LOCAL plan_cache_mode = force_custom_plan;
\echo '--- R2 key and subject index removed, custom (best prefix path)'
EXPLAIN EXECUTE probe('00000000-0000-4000-8000-000000000001', now(), '1'::xid8, 'record', gen_random_uuid());
ROLLBACK;
RESET ROLE;
