-- Run as nava_purge, the purge's role, with RLS on. Each statement is the
-- one in internal/purge/purge.go, prepared as pgx prepares it, with the
-- ADR-001 A.3 windows (7 d dispatched, 30 d dead and closed). Every DELETE
-- is rolled back, so each run sees the same rows.
--   full     a batch that deletes MaxBatch (1000) rows, custom plan
--   generic  the same with the generic plan a cached statement switches to
--   last     the batch that ends every run: nothing left to delete
\pset pager off
PREPARE idem(int) AS
DELETE FROM idempotency_keys WHERE (tenant_id, principal_id, key) IN (
        SELECT tenant_id, principal_id, key FROM idempotency_keys
        WHERE expires_at < now() LIMIT $1);
PREPARE outbox(int, float8, float8) AS
DELETE FROM outbox WHERE (tenant_id, id) IN (
        SELECT tenant_id, id FROM outbox
        WHERE dispatched_at < now() - make_interval(secs => $2)
           OR dead_at < now() - make_interval(secs => $3) LIMIT $1);
PREPARE wref(int, float8) AS
DELETE FROM workflow_ref WHERE (tenant_id, outbox_id) IN (
        SELECT tenant_id, outbox_id FROM workflow_ref
        WHERE closed_at < now() - make_interval(secs => $2) LIMIT $1);

\echo === P1 idempotency_keys full ===
BEGIN; SET LOCAL plan_cache_mode = force_custom_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE idem(1000); ROLLBACK;
\echo === P1g idempotency_keys generic ===
BEGIN; SET LOCAL plan_cache_mode = force_generic_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE idem(1000); ROLLBACK;
\echo === P1l idempotency_keys last ===
BEGIN; DELETE FROM idempotency_keys WHERE expires_at < now(); SET LOCAL plan_cache_mode = force_custom_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE idem(1000); ROLLBACK;

\echo === P2 outbox full ===
BEGIN; SET LOCAL plan_cache_mode = force_custom_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE outbox(1000, 604800, 2592000); ROLLBACK;
\echo === P2g outbox generic ===
BEGIN; SET LOCAL plan_cache_mode = force_generic_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE outbox(1000, 604800, 2592000); ROLLBACK;
\echo === P2l outbox last ===
BEGIN; DELETE FROM outbox WHERE dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days';
SET LOCAL plan_cache_mode = force_custom_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE outbox(1000, 604800, 2592000); ROLLBACK;

\echo === P3 workflow_ref full ===
BEGIN; SET LOCAL plan_cache_mode = force_custom_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE wref(1000, 2592000); ROLLBACK;
\echo === P3g workflow_ref generic ===
BEGIN; SET LOCAL plan_cache_mode = force_generic_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE wref(1000, 2592000); ROLLBACK;
\echo === P3l workflow_ref last ===
BEGIN; DELETE FROM workflow_ref WHERE closed_at < now() - interval '30 days';
SET LOCAL plan_cache_mode = force_custom_plan;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE wref(1000, 2592000); ROLLBACK;
