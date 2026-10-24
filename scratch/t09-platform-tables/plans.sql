-- T09 query plans, each as the runtime role that runs it, RLS on, tenant set
-- transaction-locally the way InTenantTx does. Plans as the owner or a
-- superuser would skip RLS, so none is taken that way.
-- big = tenant 1 (40% of every table), small = tenant 5 (0.3%).
\set ON_ERROR_STOP 1
\set big   '''' `echo $BIG` ''''
\set small '''' `echo $SMALL` ''''
\pset pager off

\echo '=== storage: heap and each index (bytes) ==='
\c nava pgadmin
SELECT c.relname AS table, pg_size_pretty(pg_relation_size(c.oid)) AS heap,
       i.relname AS index, pg_size_pretty(pg_relation_size(i.oid)) AS index_size
FROM pg_class c JOIN pg_index x ON x.indrelid = c.oid JOIN pg_class i ON i.oid = x.indexrelid
WHERE c.relnamespace = 'public'::regnamespace AND c.relname <> 'goose_db_version'
ORDER BY c.relname, i.relname;

\c nava nava_app
\echo '=== Q1 binding_lookup (A.1 step 4), nava_app, no tenant yet ==='
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT tenant_id, idp_aliases FROM tenant_identity_binding
WHERE issuer = 'https://kc.example/realms/nava' AND org_id = 'org-117';

\echo '=== Q2 idempotency key lookup (T22), nava_app, big tenant ==='
BEGIN; SELECT set_config('app.tenant_id', :big, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT state, request_hash, response_status, response_body FROM idempotency_keys
WHERE tenant_id = :big AND principal_id = md5('principal0')::uuid AND key = 'k500000';
COMMIT;

\echo '=== Q2b same, RLS as the only tenant filter (no tenant_id = $n) ==='
BEGIN; SELECT set_config('app.tenant_id', :big, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT state FROM idempotency_keys WHERE principal_id = md5('principal0')::uuid AND key = 'k500000';
COMMIT;

\echo '=== Q3 outbox dedupe-conflict lookup (T29), nava_app, big tenant ==='
BEGIN; SELECT set_config('app.tenant_id', :big, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id FROM outbox WHERE tenant_id = :big AND dedupe_key = 'd500000';
COMMIT;

\echo '=== Q4 records row lock + version (T29 Enqueue), nava_app, big tenant ==='
BEGIN; SELECT set_config('app.tenant_id', :big, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT version FROM records WHERE tenant_id = :big AND id = md5('rec500000')::uuid FOR UPDATE;
ROLLBACK;

\c nava nava_worker
\echo '=== Q5 open workflow_ref for a record (ErrNoOpenWorkflow), nava_worker, big tenant ==='
BEGIN; SELECT set_config('app.tenant_id', :big, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT outbox_id FROM workflow_ref
WHERE tenant_id = :big AND record_id = md5('rec500000')::uuid AND kind = 'approval' AND closed_at IS NULL
ORDER BY created_at DESC LIMIT 1;
COMMIT;

\echo '=== Q5b same, small tenant ==='
BEGIN; SELECT set_config('app.tenant_id', :small, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT outbox_id FROM workflow_ref
WHERE tenant_id = :small AND record_id = md5('rec3')::uuid AND kind = 'approval' AND closed_at IS NULL
ORDER BY created_at DESC LIMIT 1;
COMMIT;

\c nava nava_relay
\echo '=== Q6 relay claim, one tenant, oldest 100 pending (allow-listed, no tenant set) ==='
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id, tenant_id, target_key, target_seq FROM outbox
WHERE tenant_id = :big AND dispatched_at IS NULL AND dead_at IS NULL AND available_at <= now()
ORDER BY available_at LIMIT 100;

\echo '=== Q6b relay head of each target (FIFO), big tenant ==='
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id, target_seq FROM outbox
WHERE tenant_id = :big AND target_key = 'rec:' || md5('rec500000') AND dispatched_at IS NULL AND dead_at IS NULL
ORDER BY target_seq LIMIT 1;

\c nava nava_app
\echo '=== W1 write cost: 10,000 outbox inserts by nava_app, one tenant transaction ==='
\timing on
BEGIN; SELECT set_config('app.tenant_id', :big, true) \g /dev/null
INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload)
SELECT gen_random_uuid(), :big, 'workflow.signal', 'w' || g, 'rec:w' || g % 100, g, '{}'
FROM generate_series(1, 10000) g;
ROLLBACK;
\echo '=== W2 write cost: 10,000 idempotency_keys inserts by nava_app ==='
BEGIN; SELECT set_config('app.tenant_id', :big, true) \g /dev/null
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at)
SELECT :big, md5('principal' || g % 50)::uuid, 'w' || g, '\x00', now() + interval '1 day'
FROM generate_series(1, 10000) g;
ROLLBACK;
\timing off
