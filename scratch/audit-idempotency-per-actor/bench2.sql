-- TICK-498 follow-up on bench.sql's table: execution times (median of 5 runs) and
-- write cost, alternating the two indexes 3 times each.
SELECT tenant_id AS t, actor_id AS a, idempotency_key AS k FROM audit_log
WHERE idempotency_key LIKE 'wf:%' AND actor_id IS NOT NULL LIMIT 1 \gset m_
PREPARE by_key(uuid, text, uuid) AS
SELECT a.record_id, a.action, a.after FROM audit_log a
WHERE a.tenant_id = $1 AND a.idempotency_key = $2 AND a.idempotency_key IS NOT NULL
  AND a.actor_id IS NOT DISTINCT FROM $3;
SET plan_cache_mode = force_generic_plan;
DROP INDEX IF EXISTS new_tenant_key_actor;
CREATE UNIQUE INDEX old_tenant_key ON audit_log (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX new_tenant_actor_key ON audit_log (tenant_id, actor_id, idempotency_key) NULLS NOT DISTINCT WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX new_tenant_key_actor ON audit_log (tenant_id, idempotency_key, actor_id) NULLS NOT DISTINCT WHERE idempotency_key IS NOT NULL;
ANALYZE audit_log;
\o /dev/null
EXECUTE by_key(:'m_t', :'m_k', :'m_a');
\o
\echo '== lookup, warm, each index alone (5 runs)'
BEGIN; DROP INDEX new_tenant_actor_key, new_tenant_key_actor;
\echo old_tenant_key
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a') \g /dev/null
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
ROLLBACK;
BEGIN; DROP INDEX old_tenant_key, new_tenant_key_actor;
\echo new_tenant_actor_key
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a') \g /dev/null
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
ROLLBACK;
BEGIN; DROP INDEX old_tenant_key, new_tenant_actor_key;
\echo new_tenant_key_actor
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a') \g /dev/null
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE by_key(:'m_t', :'m_k', :'m_a');
ROLLBACK;
\echo '== write: 20,000 keyed inserts in one statement, one unique index present'
DROP INDEX new_tenant_actor_key;
\timing on
BEGIN; DROP INDEX new_tenant_key_actor; \echo old
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key) SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'x1:' || g FROM generate_series(1, 20000) g; ROLLBACK;
BEGIN; DROP INDEX old_tenant_key; \echo new
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key) SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'x2:' || g FROM generate_series(1, 20000) g; ROLLBACK;
BEGIN; DROP INDEX new_tenant_key_actor; \echo old
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key) SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'x3:' || g FROM generate_series(1, 20000) g; ROLLBACK;
BEGIN; DROP INDEX old_tenant_key; \echo new
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key) SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'x4:' || g FROM generate_series(1, 20000) g; ROLLBACK;
BEGIN; DROP INDEX new_tenant_key_actor; \echo old
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key) SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'x5:' || g FROM generate_series(1, 20000) g; ROLLBACK;
BEGIN; DROP INDEX old_tenant_key; \echo new
INSERT INTO audit_log (tenant_id, id, action, actor_id, source, idempotency_key) SELECT :'m_t', gen_random_uuid(), 'create', :'m_a', 'workflow', 'x6:' || g FROM generate_series(1, 20000) g; ROLLBACK;
