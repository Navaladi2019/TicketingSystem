-- TICK-538 (ADR-004-E5 step (ii)): lock hold and run time of the contract
-- migrations (ii-a, ii-b) and the cost of the RESTRICT check, at 50,000 groups
-- (200 tenants x 250) and 1,000,000 records (5,000 per tenant, 90% with an
-- assignment group, ~400-byte data). records carries the columns the FK and
-- records_tenant_group_open touch. Run on a throwaway
-- postgres:<docs/pins.md tag> container:  psql -U postgres -f bench.sql
\timing on
DROP TABLE IF EXISTS records, groups;
CREATE TABLE groups (
    tenant_id uuid NOT NULL, id uuid NOT NULL, keycloak_group_id text NOT NULL, path text NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT groups_pkey PRIMARY KEY (tenant_id, id));
CREATE TABLE records (
    tenant_id uuid NOT NULL, id uuid NOT NULL, state text NOT NULL, assignment_group_id uuid,
    updated_at timestamptz NOT NULL DEFAULT now(), data jsonb NOT NULL,
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id));
INSERT INTO groups SELECT t, gen_random_uuid(), 'kc-' || g, '/t/' || t || '/g' || g
  FROM (SELECT gen_random_uuid() t FROM generate_series(1, 200)) ts, generate_series(1, 250) g;
CREATE TEMP TABLE g AS SELECT tenant_id, id, (row_number() OVER (PARTITION BY tenant_id ORDER BY id) - 1)::int n FROM groups;
CREATE INDEX ON g (tenant_id, n); ANALYZE g;
INSERT INTO records (tenant_id, id, state, assignment_group_id, data)
SELECT g.tenant_id, gen_random_uuid(), CASE WHEN r % 3 = 0 THEN 'closed' ELSE 'open' END,
       CASE WHEN r % 10 = 0 THEN NULL ELSE g.id END,
       jsonb_build_object('title', repeat('x', 200), 'body', repeat('y', 150), 'n', r)
  FROM g JOIN generate_series(1, 5000) r ON g.n = r % 250;
ALTER TABLE records ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
    REFERENCES groups (tenant_id, id) ON DELETE SET NULL (assignment_group_id);
CREATE INDEX records_tenant_group_open ON records (tenant_id, assignment_group_id, updated_at DESC, id DESC)
    WHERE state <> 'closed';
VACUUM ANALYZE groups; VACUUM ANALYZE records;
SELECT count(*) AS groups FROM groups; SELECT count(*) AS records, count(assignment_group_id) AS with_group FROM records;
SELECT pg_size_pretty(pg_total_relation_size('records')) AS records_size;
SELECT version();

\echo === (ii-a) as written in db/migrations/20261104043000_groups_contract.sql (FK part) ===
BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE records
    DROP CONSTRAINT records_assignment_group_fkey,
    ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
        REFERENCES groups (tenant_id, id) ON DELETE RESTRICT NOT VALID;
SELECT relation::regclass, mode FROM pg_locks WHERE pid = pg_backend_pid() AND locktype = 'relation'
   AND relation::regclass::text IN ('records', 'groups') ORDER BY 1, 2;
COMMIT;

\echo === (ii-b) as written in db/migrations/20261104043100_groups_contract_validate.sql ===
BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;
SELECT relation::regclass, mode FROM pg_locks WHERE pid = pg_backend_pid() AND locktype = 'relation'
   AND relation::regclass::text IN ('records', 'groups') ORDER BY 1, 2;
COMMIT;
SELECT conname, confdeltype, convalidated FROM pg_constraint WHERE conname = 'records_assignment_group_fkey';

\echo === Tombstone a group a record names: no RI work on records (the key does not change) ===
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
UPDATE groups SET deleted_at = now() WHERE (tenant_id, id) = (SELECT tenant_id, assignment_group_id FROM records WHERE assignment_group_id IS NOT NULL LIMIT 1);
ROLLBACK;

\echo === Owner delete of an unreferenced group: the RESTRICT check ===
INSERT INTO groups SELECT tenant_id, gen_random_uuid(), 'kc-unused', '/t/x' FROM groups LIMIT 1;
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM groups WHERE keycloak_group_id = 'kc-unused';
ROLLBACK;
