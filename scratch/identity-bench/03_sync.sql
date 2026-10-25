-- Run as nava_groupsync (T25): the D-P13 diff reads and the writes, tenant 1.
\pset pager off
BEGIN;
SELECT set_config('app.tenant_id', md5('t1')::uuid::text, true) AS tenant;
\echo === S1 sync snapshot: every principal of the tenant ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id, keycloak_user_id, email, enabled, federated_links FROM principals WHERE tenant_id = md5('t1')::uuid;
\echo === S2 sync snapshot: every membership of the tenant ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT group_id, principal_id FROM group_members WHERE tenant_id = md5('t1')::uuid;
\echo === S3 members of one group ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT principal_id FROM group_members WHERE tenant_id = md5('t1')::uuid AND group_id = md5('g1:77')::uuid;
\echo === S4 children of an org unit ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id, name FROM org_units WHERE tenant_id = md5('t1')::uuid AND parent_id = md5('ou1:3')::uuid;
\echo === S5 delete a principal (cascades to its 8 memberships) ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM principals WHERE tenant_id = md5('t1')::uuid AND id = md5('p1:4242')::uuid;
\echo === S6 delete a group (cascades to its members) ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM groups WHERE tenant_id = md5('t1')::uuid AND id = md5('g1:4242')::uuid;
\echo === S7 delete a leaf org unit (FK check for children) ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM org_units WHERE tenant_id = md5('t1')::uuid AND id = md5('ou1:499')::uuid;
ROLLBACK;

\timing on
\echo === W1 write cost: 10,000 principals, then 80,000 memberships, one tenant transaction ===
BEGIN;
SELECT set_config('app.tenant_id', md5('t2')::uuid::text, true) AS tenant;
INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
SELECT md5('t2')::uuid, md5('np' || p)::uuid, 'new-' || p, 'n' || p || '@x.test', true, '[]', now()
FROM generate_series(1, 10000) p;
INSERT INTO group_members (tenant_id, group_id, principal_id)
SELECT md5('t2')::uuid, md5('g2:' || (1 + (p * 7 + k * 625) % 5000))::uuid, md5('np' || p)::uuid
FROM generate_series(1, 10000) p, generate_series(0, 7) k;
\echo === W2 sync touch: every principal of the tenant (synced_at, federated_links) ===
UPDATE principals SET synced_at = now(), federated_links = '[]' WHERE tenant_id = md5('t2')::uuid;
ROLLBACK;
\timing off
