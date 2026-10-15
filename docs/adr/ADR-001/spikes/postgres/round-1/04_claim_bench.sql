-- Run as nava_relay. Each claim runs in a rolled-back txn so state stays identical.
\set ON_ERROR_STOP 0
\echo '=== C1 FIFO claim, 100 rows: how many tenants get served? ==='
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id, tenant_id FROM app.outbox2
WHERE dispatched_at IS NULL AND dead_at IS NULL AND available_at <= now()
ORDER BY available_at LIMIT 100 FOR UPDATE SKIP LOCKED;
SELECT count(DISTINCT tenant_id) AS tenants_served_fifo FROM (
SELECT tenant_id FROM app.outbox2
WHERE dispatched_at IS NULL AND dead_at IS NULL AND available_at <= now()
ORDER BY available_at LIMIT 100 FOR UPDATE SKIP LOCKED) s;
ROLLBACK;

\echo '=== C2 fair claim: loose index scan for pending tenants, then <=2 rows per tenant ==='
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
WITH RECURSIVE tenants AS (
  (SELECT tenant_id FROM app.outbox2 WHERE dispatched_at IS NULL AND dead_at IS NULL ORDER BY tenant_id LIMIT 1)
  UNION ALL
  SELECT (SELECT o.tenant_id FROM app.outbox2 o WHERE o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.tenant_id > tenants.tenant_id ORDER BY o.tenant_id LIMIT 1)
  FROM tenants WHERE tenants.tenant_id IS NOT NULL
)
SELECT c.id, c.tenant_id FROM tenants
CROSS JOIN LATERAL (
  SELECT id, tenant_id FROM app.outbox2 o
  WHERE o.tenant_id = tenants.tenant_id AND o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.available_at <= now()
  ORDER BY o.available_at LIMIT 2 FOR UPDATE SKIP LOCKED) c
LIMIT 400;
ROLLBACK;
BEGIN;
WITH RECURSIVE tenants AS (
  (SELECT tenant_id FROM app.outbox2 WHERE dispatched_at IS NULL AND dead_at IS NULL ORDER BY tenant_id LIMIT 1)
  UNION ALL
  SELECT (SELECT o.tenant_id FROM app.outbox2 o WHERE o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.tenant_id > tenants.tenant_id ORDER BY o.tenant_id LIMIT 1)
  FROM tenants WHERE tenants.tenant_id IS NOT NULL
)
SELECT count(*) AS rows_claimed, count(DISTINCT c.tenant_id) AS tenants_served_fair FROM tenants
CROSS JOIN LATERAL (
  SELECT id, tenant_id FROM app.outbox2 o
  WHERE o.tenant_id = tenants.tenant_id AND o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.available_at <= now()
  ORDER BY o.available_at LIMIT 2 FOR UPDATE SKIP LOCKED) c;
ROLLBACK;

\echo '=== C3 mark dispatched (idempotent guard) ==='
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
UPDATE app.outbox2 SET dispatched_at = now()
WHERE (tenant_id, id) IN (SELECT tenant_id, id FROM app.outbox2 WHERE dispatched_at IS NULL AND dead_at IS NULL LIMIT 100)
  AND dispatched_at IS NULL;
ROLLBACK;
\echo '=== C4 relay cannot read last_error/dedupe_key or other tables ==='
SELECT dedupe_key FROM app.outbox2 LIMIT 1;
SELECT count(*) FROM app.records;
