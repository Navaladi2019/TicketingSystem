-- TICK-534 (ADR-004-E5 step (i)): lock hold of the expand migration, VALIDATE
-- time, storage and the membership-insert cost of the D11 FK, at the I5 scale
-- (50,000 groups, 2,500,000 memberships, 200 tenants). Run on a throwaway
-- postgres:<docs/pins.md tag> container:  psql -U postgres -f bench.sql
\timing on
DROP TABLE IF EXISTS group_members, groups;
CREATE TABLE groups (
    tenant_id uuid NOT NULL, id uuid NOT NULL, keycloak_group_id text NOT NULL, path text NOT NULL,
    CONSTRAINT groups_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT groups_keycloak_group_key UNIQUE (tenant_id, keycloak_group_id));
CREATE TABLE group_members (
    tenant_id uuid NOT NULL, group_id uuid NOT NULL, principal_id uuid NOT NULL,
    CONSTRAINT group_members_pkey PRIMARY KEY (tenant_id, group_id, principal_id),
    CONSTRAINT group_members_group_fkey FOREIGN KEY (tenant_id, group_id) REFERENCES groups (tenant_id, id) ON DELETE CASCADE);
CREATE INDEX group_members_principal ON group_members (tenant_id, principal_id);

INSERT INTO groups SELECT t, gen_random_uuid(), 'kc-' || g, '/t/' || t || '/g' || g
  FROM (SELECT gen_random_uuid() t FROM generate_series(1, 200)) ts, generate_series(1, 250) g;
INSERT INTO group_members SELECT tenant_id, id, gen_random_uuid() FROM groups, generate_series(1, 50);
VACUUM ANALYZE groups; VACUUM ANALYZE group_members;
SELECT count(*) AS groups FROM groups; SELECT count(*) AS memberships FROM group_members;
SELECT pg_size_pretty(pg_total_relation_size('groups')) AS groups_before, pg_size_pretty(pg_total_relation_size('group_members')) AS members_before;

-- Baseline: 100,000 membership inserts under the one existing FK.
CREATE TEMP TABLE batch AS SELECT tenant_id, id, gen_random_uuid() p FROM groups, generate_series(1, 2) LIMIT 100000;
BEGIN; INSERT INTO group_members SELECT * FROM batch; ROLLBACK;
BEGIN; INSERT INTO group_members SELECT * FROM batch; ROLLBACK;

-- Migration A (expand), as written in db/migrations.
BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE groups
    ADD COLUMN deleted_at timestamptz,
    ADD COLUMN live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED,
    ADD CONSTRAINT groups_live_key UNIQUE (tenant_id, id, live);
ALTER TABLE group_members
    ADD COLUMN live boolean NOT NULL DEFAULT true,
    ADD CONSTRAINT group_members_live_check CHECK (live) NOT VALID,
    ADD CONSTRAINT group_members_group_live_fkey FOREIGN KEY (tenant_id, group_id, live)
        REFERENCES groups (tenant_id, id, live) NOT VALID;
SELECT relation::regclass, mode FROM pg_locks WHERE pid = pg_backend_pid() AND relation::regclass::text IN ('groups','group_members') ORDER BY 1, 2;
COMMIT;

-- Migration B (validate), its own transaction.
BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_live_check;
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
SELECT relation::regclass, mode FROM pg_locks WHERE pid = pg_backend_pid() AND relation::regclass::text IN ('groups','group_members') ORDER BY 1, 2;
COMMIT;
VACUUM ANALYZE groups; VACUUM ANALYZE group_members;
SELECT pg_size_pretty(pg_total_relation_size('groups')) AS groups_after, pg_size_pretty(pg_total_relation_size('group_members')) AS members_after,
       pg_size_pretty(pg_relation_size('groups_live_key')) AS groups_live_key;

-- After: the same 100,000 inserts under both FKs.
BEGIN; INSERT INTO group_members (tenant_id, group_id, principal_id) SELECT * FROM batch; ROLLBACK;
BEGIN; INSERT INTO group_members (tenant_id, group_id, principal_id) SELECT * FROM batch; ROLLBACK;

-- The two D11 refusals and the tombstone of an emptied group, with plans.
\set g '(SELECT id FROM groups ORDER BY id LIMIT 1)'
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
  DELETE FROM group_members WHERE (tenant_id, group_id) = (SELECT tenant_id, id FROM groups ORDER BY id LIMIT 1);
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
  UPDATE groups SET deleted_at = now() WHERE (tenant_id, id) = (SELECT tenant_id, id FROM groups ORDER BY id LIMIT 1);
ROLLBACK;
