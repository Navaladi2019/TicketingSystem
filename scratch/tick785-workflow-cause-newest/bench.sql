-- TICK-785: causeSQL's newest-cause EXISTS before and after
-- workflow_cause_ref_newest. Table, RLS and policy as in
-- db/migrations/20261113123300_workflow_definitions.sql (FK to tenants dropped:
-- no tenants table here; it plays no part in the read).
\timing off
SELECT version();
SET max_parallel_workers_per_gather = 0;
SET jit = off;

CREATE ROLE nava_worker NOLOGIN;
CREATE TABLE workflow_cause (
    tenant_id           uuid        NOT NULL,
    outbox_id           uuid        NOT NULL,
    ref_outbox_id       uuid        NOT NULL,
    causation_origin_id uuid        NOT NULL,
    causation_depth     smallint    NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workflow_cause_pkey PRIMARY KEY (tenant_id, outbox_id),
    CONSTRAINT workflow_cause_causation_depth_check CHECK (causation_depth >= 0)
);
ALTER TABLE workflow_cause ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_cause FORCE ROW LEVEL SECURITY;
CREATE POLICY workflow_cause_select ON workflow_cause FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT ON workflow_cause TO nava_worker;

-- Shape: 1,000 tenants x 200 runs x 10 causes = 2,000,000 rows, plus one heavy
-- tenant (00000000-...-01) with 2,000 runs x 50 causes = 100,000 rows.
-- created_at strictly increases per run; ids are random v4.
INSERT INTO workflow_cause
SELECT t.id, gen_random_uuid(), r.id, gen_random_uuid(), c.n % 5,
       timestamptz '2026-11-01' + (r.n * 10 + c.n) * interval '1 second'
FROM (SELECT gen_random_uuid() AS id FROM generate_series(1, 1000)) t
CROSS JOIN LATERAL (SELECT gen_random_uuid() AS id, n FROM generate_series(1, 200) n WHERE t.id IS NOT NULL) r
CROSS JOIN generate_series(1, 10) c(n);
INSERT INTO workflow_cause
SELECT '00000000-0000-4000-8000-000000000001', gen_random_uuid(), r.id, gen_random_uuid(), c.n % 5,
       timestamptz '2026-11-01' + (r.n * 50 + c.n) * interval '1 second'
FROM (SELECT gen_random_uuid() AS id, n FROM generate_series(1, 2000) n) r
CROSS JOIN generate_series(1, 50) c(n);
VACUUM ANALYZE workflow_cause;
SELECT count(*) AS rows, count(DISTINCT tenant_id) AS tenants FROM workflow_cause;

-- Probes: the heavy tenant's newest cause of one run (the hot path: a
-- current delivery, newer = false) and an older cause of the same run
-- (a replayed delivery, newer = true).
SELECT outbox_id AS newest FROM workflow_cause
WHERE tenant_id = '00000000-0000-4000-8000-000000000001'
ORDER BY ref_outbox_id, created_at DESC LIMIT 1 \gset
SELECT c.outbox_id AS stale FROM workflow_cause c
JOIN workflow_cause n ON n.tenant_id = c.tenant_id AND n.outbox_id = :'newest'
WHERE c.tenant_id = n.tenant_id AND c.ref_outbox_id = n.ref_outbox_id
ORDER BY c.created_at LIMIT 1 \gset

SET app.tenant_id = '00000000-0000-4000-8000-000000000001';
SET ROLE nava_worker;

PREPARE cause(uuid, uuid) AS
SELECT c.ref_outbox_id, c.causation_origin_id, c.causation_depth,
       EXISTS (SELECT 1 FROM workflow_cause n
               WHERE n.tenant_id = c.tenant_id AND n.ref_outbox_id = c.ref_outbox_id
                 AND (n.created_at, n.outbox_id) > (c.created_at, c.outbox_id)) AS newer
FROM workflow_cause c
WHERE c.tenant_id = $1 AND c.outbox_id = $2;
SET plan_cache_mode = force_custom_plan;

\echo '=== BEFORE: newest cause (newer = false) ==='
EXECUTE cause('00000000-0000-4000-8000-000000000001', :'newest');
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE cause('00000000-0000-4000-8000-000000000001', :'newest');
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE cause('00000000-0000-4000-8000-000000000001', :'newest');
\echo '=== BEFORE: stale cause (newer = true) ==='
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE cause('00000000-0000-4000-8000-000000000001', :'stale');

RESET ROLE;
\timing on
CREATE INDEX CONCURRENTLY workflow_cause_ref_newest
    ON workflow_cause (tenant_id, ref_outbox_id, created_at, outbox_id);
\timing off
ANALYZE workflow_cause;
SET ROLE nava_worker;
DEALLOCATE cause;
PREPARE cause(uuid, uuid) AS
SELECT c.ref_outbox_id, c.causation_origin_id, c.causation_depth,
       EXISTS (SELECT 1 FROM workflow_cause n
               WHERE n.tenant_id = c.tenant_id AND n.ref_outbox_id = c.ref_outbox_id
                 AND (n.created_at, n.outbox_id) > (c.created_at, c.outbox_id)) AS newer
FROM workflow_cause c
WHERE c.tenant_id = $1 AND c.outbox_id = $2;

\echo '=== AFTER: newest cause (newer = false) ==='
EXECUTE cause('00000000-0000-4000-8000-000000000001', :'newest');
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE cause('00000000-0000-4000-8000-000000000001', :'newest');
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE cause('00000000-0000-4000-8000-000000000001', :'newest');
\echo '=== AFTER: stale cause (newer = true) ==='
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE cause('00000000-0000-4000-8000-000000000001', :'stale');
RESET ROLE;

\echo '=== Sizes ==='
SELECT relname, pg_size_pretty(pg_relation_size(oid)) FROM pg_class
WHERE relname IN ('workflow_cause', 'workflow_cause_pkey', 'workflow_cause_ref_newest');

\echo '=== Insert cost: 100,000 rows into the light tenants, without vs with the index ==='
CREATE TEMP TABLE batch AS
SELECT tenant_id, gen_random_uuid() AS outbox_id, ref_outbox_id, causation_origin_id, causation_depth,
       created_at + interval '1 day' AS created_at
FROM workflow_cause TABLESAMPLE SYSTEM (5) LIMIT 100000;
-- Warm-up (the first run pays the cold cache), then alternate three times each.
BEGIN; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
BEGIN; DROP INDEX workflow_cause_ref_newest; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
CHECKPOINT;
\timing on
\echo 'without'
BEGIN; DROP INDEX workflow_cause_ref_newest; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
\echo 'with'
BEGIN; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
\echo 'without'
BEGIN; DROP INDEX workflow_cause_ref_newest; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
\echo 'with'
BEGIN; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
\echo 'without'
BEGIN; DROP INDEX workflow_cause_ref_newest; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
\echo 'with'
BEGIN; INSERT INTO workflow_cause SELECT * FROM batch; ROLLBACK;
