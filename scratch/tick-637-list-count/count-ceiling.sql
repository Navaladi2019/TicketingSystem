-- The floor of an exact count of 1,000,000 visible rows (service desk, tenant L),
-- as nava_app: the planner's choice, then an index-only scan forced.
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-00000000000a', true);
SET LOCAL jit = off;
EXPLAIN (ANALYZE, BUFFERS) SELECT count(*) FROM records r
 WHERE r.tenant_id = '00000000-0000-4000-8000-00000000000a' AND r.entity_definition_id = '95910030-8377-6777-7313-1b76e0f9756d';
SET LOCAL enable_seqscan = off; SET LOCAL enable_bitmapscan = off;
EXPLAIN (ANALYZE, BUFFERS) SELECT count(*) FROM records r
 WHERE r.tenant_id = '00000000-0000-4000-8000-00000000000a' AND r.entity_definition_id = '95910030-8377-6777-7313-1b76e0f9756d';
SET LOCAL max_parallel_workers_per_gather = 0;
EXPLAIN (ANALYZE, BUFFERS) SELECT count(*) FROM records r
 WHERE r.tenant_id = '00000000-0000-4000-8000-00000000000a' AND r.entity_definition_id = '95910030-8377-6777-7313-1b76e0f9756d';
ROLLBACK;
