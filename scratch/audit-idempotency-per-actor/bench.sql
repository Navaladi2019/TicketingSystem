-- TICK-498: audit_log idempotency unique index, per tenant vs per actor.
-- Run: psql -v ON_ERROR_STOP=1 -f bench.sql (PostgreSQL 16.10, see README).
-- 2,000,000 audit rows, 200 tenants, 50 actors per tenant plus NULL
-- (system writer) on 1 row in 10; 1 row in 2 keyed (the primary row of a
-- write), the shape of scratch/records-pipeline-bench.
\timing off
SET max_parallel_maintenance_workers = 0;
DROP TABLE IF EXISTS audit_log;
CREATE TABLE audit_log (
    tenant_id uuid NOT NULL, id uuid NOT NULL, record_id uuid, action text NOT NULL,
    actor_id uuid, source text NOT NULL, after jsonb, idempotency_key text,
    at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (tenant_id, id));
INSERT INTO audit_log (tenant_id, id, record_id, action, actor_id, source, after, idempotency_key)
SELECT ('00000000-0000-7000-8000-' || lpad(to_hex(g % 200), 12, '0'))::uuid,
       gen_random_uuid(), gen_random_uuid(), 'create',
       CASE WHEN g % 10 = 0 THEN NULL
            ELSE ('00000000-0000-7000-9000-' || lpad(to_hex((g / 200) % 50), 12, '0'))::uuid END,
       'workflow', '{"number":"INC0000001","version":1,"schema_version":1,"data":{}}',
       CASE WHEN g % 2 = 0 THEN 'wf:' || md5(g::text) || ':step-1' END
FROM generate_series(1, 2000000) g;
VACUUM ANALYZE audit_log;

\echo '== candidate build times and sizes'
\timing on
CREATE UNIQUE INDEX old_tenant_key ON audit_log (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX new_tenant_actor_key ON audit_log (tenant_id, actor_id, idempotency_key) NULLS NOT DISTINCT WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX new_tenant_key_actor ON audit_log (tenant_id, idempotency_key, actor_id) NULLS NOT DISTINCT WHERE idempotency_key IS NOT NULL;
\timing off
SELECT relname, pg_size_pretty(pg_relation_size(oid)) AS size FROM pg_class
WHERE relname IN ('old_tenant_key', 'new_tenant_actor_key', 'new_tenant_key_actor', 'audit_log') ORDER BY relname;
ANALYZE audit_log;

-- A real keyed row by a member, and one by the system writer.
SELECT tenant_id AS t, actor_id AS a, idempotency_key AS k FROM audit_log
WHERE idempotency_key IS NOT NULL AND actor_id IS NOT NULL LIMIT 1 \gset m_
SELECT tenant_id AS t, idempotency_key AS k FROM audit_log
WHERE idempotency_key IS NOT NULL AND actor_id IS NULL LIMIT 1 \gset s_

-- store.AuditByKey's predicate, as shipped (IS NOT DISTINCT FROM).
PREPARE by_key(uuid, text, uuid) AS
SELECT a.record_id, a.action, a.after FROM audit_log a
WHERE a.tenant_id = $1 AND a.idempotency_key = $2 AND a.idempotency_key IS NOT NULL
  AND a.actor_id IS NOT DISTINCT FROM $3;
SET plan_cache_mode = force_generic_plan; -- nava_app's setting (db-roles)

\echo '== old index only'
BEGIN;
DROP INDEX new_tenant_actor_key, new_tenant_key_actor;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
ROLLBACK;

\echo '== (tenant_id, actor_id, idempotency_key) only'
BEGIN;
DROP INDEX old_tenant_key, new_tenant_key_actor;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY OFF) EXECUTE by_key(:'s_t', :'s_k', NULL);
ROLLBACK;

\echo '== (tenant_id, idempotency_key, actor_id) only (chosen)'
BEGIN;
DROP INDEX old_tenant_key, new_tenant_actor_key;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY OFF) EXECUTE by_key(:'s_t', :'s_k', NULL);
ROLLBACK;

\echo '== write cost: 20,000 keyed inserts, one unique index each'
DROP INDEX new_tenant_actor_key, new_tenant_key_actor;
\timing on
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key)
SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'w1:' || g FROM generate_series(1, 20000) g;
\timing off
DROP INDEX old_tenant_key;
CREATE UNIQUE INDEX new_tenant_key_actor ON audit_log (tenant_id, idempotency_key, actor_id) NULLS NOT DISTINCT WHERE idempotency_key IS NOT NULL;
\timing on
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key)
SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'w2:' || g FROM generate_series(1, 20000) g;
\timing off
