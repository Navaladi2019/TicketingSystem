-- Round 2 cost: FIFO-per-target claim, and workflow_ref. Run as nava_relay / nava_app.
\set ON_ERROR_STOP 0
\echo '=== C1 fair + FIFO claim, normal state (flood tenant 45 deep x 1,000 targets) ==='
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
WITH RECURSIVE tenants AS (
  (SELECT tenant_id FROM app.outbox3 WHERE dispatched_at IS NULL AND dead_at IS NULL ORDER BY tenant_id LIMIT 1)
  UNION ALL
  SELECT (SELECT o.tenant_id FROM app.outbox3 o WHERE o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.tenant_id > tenants.tenant_id ORDER BY o.tenant_id LIMIT 1)
  FROM tenants WHERE tenants.tenant_id IS NOT NULL)
SELECT c.* FROM tenants CROSS JOIN LATERAL (
  SELECT o.id, o.tenant_id FROM app.outbox3 o
  WHERE o.tenant_id = tenants.tenant_id AND o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.available_at <= now()
    AND NOT EXISTS (SELECT 1 FROM app.outbox3 p WHERE p.tenant_id = o.tenant_id AND p.target_key = o.target_key
                    AND p.dispatched_at IS NULL AND p.dead_at IS NULL AND p.id < o.id)
  ORDER BY o.available_at LIMIT 2 FOR UPDATE OF o SKIP LOCKED) c LIMIT 400;
ROLLBACK;
