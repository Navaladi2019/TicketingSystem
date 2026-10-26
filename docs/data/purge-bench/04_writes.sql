-- Write cost of the indexes, as the bootstrap superuser (index maintenance
-- does not depend on the role). Each write is the one its owner runs, over
-- every row it would apply to here, and is rolled back. WAL bytes and the
-- HOT count are read inside the transaction, kept after it. run.sh starts
-- the server with full_page_writes off, so WAL is the change records alone,
-- not whichever page images the last checkpoint happens to force.
\pset pager off
\timing on
CREATE TEMP TABLE w (step text, rows bigint, wal_bytes numeric, hot bigint);

\echo === W1 relay marks every pending outbox row dispatched (relay.go) ===
BEGIN;
SELECT pg_current_wal_insert_lsn() AS l0 \gset
UPDATE outbox SET dispatched_at = now(), attempts = attempts + 1
WHERE dispatched_at IS NULL AND dead_at IS NULL;
SELECT n_tup_upd AS n, pg_current_wal_insert_lsn() - :'l0' AS wal, n_tup_hot_upd AS hot
FROM pg_stat_xact_user_tables WHERE relname = 'outbox' \gset
ROLLBACK;
INSERT INTO w VALUES ('W1 outbox dispatch', :n, :wal, :hot);

\echo === W2 worker closes every open workflow_ref (workerguard/workflowref.go) ===
BEGIN;
SELECT pg_current_wal_insert_lsn() AS l0 \gset
UPDATE workflow_ref SET closed_at = COALESCE(closed_at, now()) WHERE closed_at IS NULL;
SELECT n_tup_upd AS n, pg_current_wal_insert_lsn() - :'l0' AS wal, n_tup_hot_upd AS hot
FROM pg_stat_xact_user_tables WHERE relname = 'workflow_ref' \gset
ROLLBACK;
INSERT INTO w VALUES ('W2 workflow_ref close', :n, :wal, :hot);

-- The seeded heap is packed, so W2 cannot be HOT with or without the index.
-- A live table has free space where purge deleted rows; this copy has 20%.
\echo === W2h the same on a copy with free space on every page (fillfactor 80) ===
CREATE TABLE wr_hot (LIKE workflow_ref INCLUDING INDEXES) WITH (fillfactor = 80, autovacuum_enabled = false);
INSERT INTO wr_hot SELECT * FROM workflow_ref;
BEGIN;
SELECT pg_current_wal_insert_lsn() AS l0 \gset
UPDATE wr_hot SET closed_at = COALESCE(closed_at, now()) WHERE closed_at IS NULL;
SELECT n_tup_upd AS n, pg_current_wal_insert_lsn() - :'l0' AS wal, n_tup_hot_upd AS hot
FROM pg_stat_xact_user_tables WHERE relname = 'wr_hot' \gset
ROLLBACK;
INSERT INTO w VALUES ('W2h workflow_ref close, fillfactor 80', :n, :wal, :hot);
DROP TABLE wr_hot;

\echo === W3 app inserts 10,000 idempotency keys ===
BEGIN;
SELECT pg_current_wal_insert_lsn() AS l0 \gset
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at)
SELECT md5('t1')::uuid, md5('p' || i)::uuid, 'new' || i, decode(md5('n' || i), 'hex'), now() + interval '24 hours'
FROM generate_series(1, 10000) i;
SELECT n_tup_ins AS n, pg_current_wal_insert_lsn() - :'l0' AS wal
FROM pg_stat_xact_user_tables WHERE relname = 'idempotency_keys' \gset
ROLLBACK;
INSERT INTO w VALUES ('W3 idempotency insert', :n, :wal, 0);

SELECT step, rows, pg_size_pretty(wal_bytes) AS wal, round(wal_bytes / rows) AS wal_per_row, hot FROM w;
