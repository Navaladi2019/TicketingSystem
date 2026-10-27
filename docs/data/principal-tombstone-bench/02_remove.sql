-- TICK-220: removing 1,000 principals from a 50,000-principal tenant in one
-- sync transaction, hard delete (today) against tombstone (the decision), as
-- nava_groupsync with RLS on. Each is rolled back, so both see the same rows.
\timing on
SELECT set_config('app.tenant_id', md5('t1')::text, false);
-- nava_groupsync cannot create temporary tables, so "gone" is a subquery:
-- the first 1,000 ids of tenant t1.
\set gone '(SELECT id FROM principals WHERE tenant_id = md5(''t1'')::uuid ORDER BY id LIMIT 1000)'
\set one '(SELECT id FROM principals WHERE tenant_id = md5(''t1'')::uuid ORDER BY id LIMIT 1)'
\echo === hard delete, 1,000 principals (cascades to 8,000 memberships) ===
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM principals WHERE tenant_id = md5('t1')::uuid AND id = ANY(ARRAY:gone);
ROLLBACK;
\echo === tombstone, 1,000 principals, then their 8,000 memberships ===
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
UPDATE principals SET enabled = false, deleted_at = now(), synced_at = now()
WHERE tenant_id = md5('t1')::uuid AND id = ANY(ARRAY:gone);
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM group_members WHERE tenant_id = md5('t1')::uuid AND principal_id = ANY(ARRAY:gone);
SELECT n_tup_upd, n_tup_hot_upd FROM pg_stat_xact_user_tables WHERE relname = 'principals';
ROLLBACK;
\echo === resolver on a tombstoned principal (the R1 key lookup) ===
BEGIN;
UPDATE principals SET enabled = false, deleted_at = now() WHERE tenant_id = md5('t1')::uuid AND id = md5('p1:1')::uuid;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT p.id, p.enabled, p.deleted_at FROM principals p
WHERE p.tenant_id = md5('t1')::uuid AND p.keycloak_user_id = md5('kcu1:1')::uuid::text;
ROLLBACK;
\echo === a tombstone that is still enabled is refused ===
\set ON_ERROR_STOP 0
UPDATE principals SET deleted_at = now() WHERE tenant_id = md5('t1')::uuid AND id = :one;
