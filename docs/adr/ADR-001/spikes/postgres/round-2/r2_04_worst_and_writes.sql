-- Round 2 cost, part 2. Run as superuser.
\set ON_ERROR_STOP 0
\echo '=== C2 worst case: all 1,000 flood-tenant heads backing off (available_at +1h) ==='
UPDATE app.outbox3 o SET available_at = now() + interval '1 hour'
WHERE o.dispatched_at IS NULL AND o.tenant_id = (SELECT id FROM t WHERE n = 1)
  AND NOT EXISTS (SELECT 1 FROM app.outbox3 p WHERE p.tenant_id = o.tenant_id AND p.target_key = o.target_key
                  AND p.dispatched_at IS NULL AND p.id < o.id);
VACUUM ANALYZE app.outbox3;
SET ROLE nava_relay;
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT o.id FROM app.outbox3 o
WHERE o.tenant_id = (SELECT id FROM t WHERE n = 1) AND o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.available_at <= now()
  AND NOT EXISTS (SELECT 1 FROM app.outbox3 p WHERE p.tenant_id = o.tenant_id AND p.target_key = o.target_key
                  AND p.dispatched_at IS NULL AND p.dead_at IS NULL AND p.id < o.id)
ORDER BY o.available_at LIMIT 2 FOR UPDATE OF o SKIP LOCKED;
ROLLBACK;
RESET ROLE;

\echo '=== W1 insert cost: 100,000 outbox rows WITH the FIFO index ==='
\timing on
BEGIN;
INSERT INTO app.outbox3 (id, tenant_id, kind, dedupe_key, target_key)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 1 + (g % 200)), 'workflow.signal', 'w'||g, 'rec:w'||g FROM generate_series(1,100000) g;
ROLLBACK;
\timing off
DROP INDEX app.outbox3_fifo;
\echo '=== W2 insert cost: 100,000 outbox rows WITHOUT the FIFO index ==='
\timing on
BEGIN;
INSERT INTO app.outbox3 (id, tenant_id, kind, dedupe_key, target_key)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 1 + (g % 200)), 'workflow.signal', 'w'||g, 'rec:w'||g FROM generate_series(1,100000) g;
ROLLBACK;
\echo '=== W3 workflow_ref insert cost: 100,000 rows ==='
BEGIN;
INSERT INTO app.workflow_ref (tenant_id, workflow_id, record_id, kind)
SELECT (SELECT id FROM t WHERE n = 1 + (g % 200)), 't:x:approval:'||gen_random_uuid(), gen_random_uuid(), 'approval' FROM generate_series(1,100000) g;
ROLLBACK;
\timing off

\echo '=== L1 workflow_ref lookup as nava_app under RLS (latest workflow for a record) ==='
SELECT tenant_id::text AS tid, record_id::text AS rid FROM app.workflow_ref WHERE record_id = md5('777'::text)::uuid LIMIT 1 \gset
SET ROLE nava_app;
BEGIN; SELECT set_config('app.tenant_id', :'tid', true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT workflow_id FROM app.workflow_ref WHERE record_id = :'rid' AND kind = 'approval' ORDER BY created_at DESC LIMIT 1;
ROLLBACK;
RESET ROLE;
