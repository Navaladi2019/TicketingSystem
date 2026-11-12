-- TICK-715: foreign-key VALIDATE CONSTRAINT under FORCE ROW LEVEL SECURITY.
-- 1. Reproduces the bug: the table owner's bare VALIDATE marks a foreign key
--    valid over an orphan.
-- 2. Shows the P3-40 pattern (SELECT policies for the migrator that live only
--    inside the transaction) catches the orphan.
-- 3. Times the re-check migrations' SQL as written, and the locks they hold,
--    at 50,000 groups (200 tenants x 250), 2,500,000 memberships and
--    1,000,000 records (900,000 with an assignment group).
-- Run on a throwaway postgres:16.10 container, from this directory:
--   psql -U postgres -f bench.sql
\timing on
DROP TABLE IF EXISTS records, group_members, groups;
DROP ROLE IF EXISTS bench_migrator;
CREATE ROLE bench_migrator NOSUPERUSER NOBYPASSRLS;
CREATE TABLE groups (
    tenant_id uuid NOT NULL, id uuid NOT NULL, path text NOT NULL, deleted_at timestamptz,
    live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED,
    CONSTRAINT groups_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT groups_live_key UNIQUE (tenant_id, id, live));
CREATE TABLE group_members (
    tenant_id uuid NOT NULL, group_id uuid NOT NULL, principal_id uuid NOT NULL,
    live boolean NOT NULL DEFAULT true,
    CONSTRAINT group_members_pkey PRIMARY KEY (tenant_id, group_id, principal_id));
CREATE TABLE records (
    tenant_id uuid NOT NULL, id uuid NOT NULL, state text NOT NULL, assignment_group_id uuid,
    updated_at timestamptz NOT NULL DEFAULT now(), data jsonb NOT NULL,
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id));
INSERT INTO groups (tenant_id, id, path) SELECT t, gen_random_uuid(), '/t/' || t || '/g' || g
  FROM (SELECT gen_random_uuid() t FROM generate_series(1, 200)) ts, generate_series(1, 250) g;
CREATE TEMP TABLE g AS SELECT tenant_id, id, (row_number() OVER (PARTITION BY tenant_id ORDER BY id) - 1)::int n FROM groups;
CREATE INDEX ON g (tenant_id, n); ANALYZE g;
INSERT INTO group_members (tenant_id, group_id, principal_id)
SELECT tenant_id, id, gen_random_uuid() FROM g, generate_series(1, 50);
INSERT INTO records (tenant_id, id, state, assignment_group_id, data)
SELECT g.tenant_id, gen_random_uuid(), CASE WHEN r % 3 = 0 THEN 'closed' ELSE 'open' END,
       CASE WHEN r % 10 = 0 THEN NULL ELSE g.id END,
       jsonb_build_object('title', repeat('x', 200), 'body', repeat('y', 150), 'n', r)
  FROM g JOIN generate_series(1, 5000) r ON g.n = r % 250;
ALTER TABLE group_members ADD CONSTRAINT group_members_group_live_fkey FOREIGN KEY (tenant_id, group_id, live)
    REFERENCES groups (tenant_id, id, live) NOT VALID;
ALTER TABLE records ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
    REFERENCES groups (tenant_id, id) ON DELETE RESTRICT NOT VALID;
ALTER TABLE groups OWNER TO bench_migrator;
ALTER TABLE group_members OWNER TO bench_migrator;
ALTER TABLE records OWNER TO bench_migrator;
ALTER TABLE groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE groups FORCE ROW LEVEL SECURITY;
ALTER TABLE group_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE group_members FORCE ROW LEVEL SECURITY;
ALTER TABLE records ENABLE ROW LEVEL SECURITY;
ALTER TABLE records FORCE ROW LEVEL SECURITY;
-- One orphan each, written past the RI triggers (the superuser bypasses RLS).
SET session_replication_role = replica;
INSERT INTO group_members (tenant_id, group_id, principal_id)
SELECT tenant_id, gen_random_uuid(), gen_random_uuid() FROM groups LIMIT 1;
INSERT INTO records (tenant_id, id, state, assignment_group_id, data)
SELECT tenant_id, gen_random_uuid(), 'open', gen_random_uuid(), '{}' FROM groups LIMIT 1;
RESET session_replication_role;
VACUUM ANALYZE groups;
VACUUM ANALYZE group_members;
VACUUM ANALYZE records;
SELECT (SELECT count(*) FROM groups) AS groups, (SELECT count(*) FROM group_members) AS memberships,
       (SELECT count(*) FROM records) AS records, (SELECT count(assignment_group_id) FROM records) AS with_group;
SELECT pg_size_pretty(pg_total_relation_size('records')) AS records_size,
       pg_size_pretty(pg_total_relation_size('group_members')) AS members_size;
SELECT version();

\echo === Orphan query, RLS bypassed (superuser): one orphan per key ===
\i orphans.sql

\echo === BUG: the table owner, bare VALIDATE, passes over the orphans ===
SET ROLE bench_migrator;
BEGIN;
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;
SELECT conname, convalidated FROM pg_constraint
 WHERE conname IN ('group_members_group_live_fkey', 'records_assignment_group_fkey') ORDER BY 1;
ROLLBACK;

\echo === FIX, group_members: the validate under migrator policies fails on the orphan ===
BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE POLICY group_members_migrator_validate ON group_members FOR SELECT TO bench_migrator USING (true);
CREATE POLICY groups_migrator_validate ON groups FOR SELECT TO bench_migrator USING (true);
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
ROLLBACK;
\echo === FIX, records: the validate under migrator policies fails on the orphan ===
BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE POLICY records_migrator_validate ON records FOR SELECT TO bench_migrator USING (true);
CREATE POLICY groups_migrator_validate ON groups FOR SELECT TO bench_migrator USING (true);
ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;
ROLLBACK;
RESET ROLE;

\echo === Remove the orphans and mark both keys valid, as the vacuous migrations left them ===
SET session_replication_role = replica;
DELETE FROM group_members gm WHERE NOT EXISTS (SELECT 1 FROM groups g
    WHERE g.tenant_id = gm.tenant_id AND g.id = gm.group_id AND g.live = gm.live);
DELETE FROM records r WHERE r.assignment_group_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM groups g
    WHERE g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id);
RESET session_replication_role;
SET ROLE bench_migrator;
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;

\echo === Re-check migration 1 as written: group_members_group_live_fkey ===
SELECT clock_timestamp() AS t0 \gset
BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE POLICY group_members_migrator_validate ON group_members FOR SELECT TO bench_migrator USING (true);
CREATE POLICY groups_migrator_validate ON groups FOR SELECT TO bench_migrator USING (true);
ALTER TABLE group_members
    DROP CONSTRAINT group_members_group_live_fkey,
    ADD CONSTRAINT group_members_group_live_fkey FOREIGN KEY (tenant_id, group_id, live)
        REFERENCES groups (tenant_id, id, live) NOT VALID;
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
DROP POLICY groups_migrator_validate ON groups;
DROP POLICY group_members_migrator_validate ON group_members;
SELECT relation::regclass, mode FROM pg_locks WHERE pid = pg_backend_pid() AND locktype = 'relation'
   AND relation::regclass::text IN ('records', 'groups', 'group_members') AND granted ORDER BY 1, 2;
COMMIT;
SELECT round(extract(epoch FROM clock_timestamp() - :'t0'::timestamptz) * 1000) AS group_members_tx_ms;

\echo === Re-check migration 2 as written: records_assignment_group_fkey ===
SELECT clock_timestamp() AS t0 \gset
BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE POLICY records_migrator_validate ON records FOR SELECT TO bench_migrator USING (true);
CREATE POLICY groups_migrator_validate ON groups FOR SELECT TO bench_migrator USING (true);
ALTER TABLE records
    DROP CONSTRAINT records_assignment_group_fkey,
    ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
        REFERENCES groups (tenant_id, id) ON DELETE RESTRICT NOT VALID;
ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;
DROP POLICY groups_migrator_validate ON groups;
DROP POLICY records_migrator_validate ON records;
SELECT relation::regclass, mode FROM pg_locks WHERE pid = pg_backend_pid() AND locktype = 'relation'
   AND relation::regclass::text IN ('records', 'groups', 'group_members') AND granted ORDER BY 1, 2;
COMMIT;
SELECT round(extract(epoch FROM clock_timestamp() - :'t0'::timestamptz) * 1000) AS records_tx_ms;
RESET ROLE;
SELECT conname, convalidated FROM pg_constraint
 WHERE conname IN ('group_members_group_live_fkey', 'records_assignment_group_fkey') ORDER BY 1;
SELECT count(*) AS leftover_policies FROM pg_policies WHERE policyname LIKE '%migrator_validate';

\echo === Orphan query on the clean database ===
\i orphans.sql
