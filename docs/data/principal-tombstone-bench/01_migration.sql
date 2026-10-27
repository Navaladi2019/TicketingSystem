-- TICK-220: the cost of 00006 and 00007 on the identity-bench data (519,000
-- principals), run as pgadmin after 01_seed.sql. Both already ran on the
-- empty table, so they are undone and replayed here, then the rejected
-- one-step variant is timed twice.
\timing on
\echo === 00006 + 00007 replayed: ADD COLUMN + CHECK NOT VALID, then VALIDATE ===
ALTER TABLE principals DROP CONSTRAINT principals_tombstone_check, DROP COLUMN deleted_at;
VACUUM (ANALYZE) principals;
SELECT count(*) FROM principals;
BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE principals ADD COLUMN deleted_at timestamptz;
ALTER TABLE principals ADD CONSTRAINT principals_tombstone_check CHECK (deleted_at IS NULL OR NOT enabled) NOT VALID;
COMMIT;
ALTER TABLE principals VALIDATE CONSTRAINT principals_tombstone_check;
-- DROP COLUMN above took nava_groupsync's column grant with it; nava-db
-- migrate would re-apply it from db/grants.yaml.
GRANT UPDATE (deleted_at) ON principals TO nava_groupsync;
\echo === variant rejected: validated CHECK under ACCESS EXCLUSIVE, two runs (warm) ===
ALTER TABLE principals DROP CONSTRAINT principals_tombstone_check;
ALTER TABLE principals ADD CONSTRAINT principals_tombstone_check CHECK (deleted_at IS NULL OR NOT enabled);
ALTER TABLE principals DROP CONSTRAINT principals_tombstone_check;
ALTER TABLE principals ADD CONSTRAINT principals_tombstone_check CHECK (deleted_at IS NULL OR NOT enabled);
\echo === size before tombstones ===
VACUUM (ANALYZE) principals;
SELECT pg_size_pretty(pg_table_size('principals')) AS heap, pg_size_pretty(pg_indexes_size('principals')) AS indexes,
       (SELECT avg(pg_column_size(p.*))::int FROM principals p) AS avg_row_bytes;
