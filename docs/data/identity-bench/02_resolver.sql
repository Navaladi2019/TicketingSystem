-- Run as nava_app, the resolver's role (T17), with RLS on and the tenant set
-- transaction-locally, as InTenantTx does. Tenant 1 is large (50k
-- principals, 5k groups); tenant 150 is small.
\pset pager off
BEGIN;
SELECT set_config('app.tenant_id', md5('t1')::uuid::text, true) AS tenant;

\echo === R1 resolver lookup (A.1 step 3 + Principal.Groups), big tenant, typical user (8 groups) ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT p.id, p.enabled, p.federated_links,
       ARRAY(SELECT gm.group_id FROM group_members gm
             WHERE gm.tenant_id = p.tenant_id AND gm.principal_id = p.id ORDER BY gm.group_id) AS groups
FROM principals p
WHERE p.tenant_id = md5('t1')::uuid AND p.keycloak_user_id = md5('kcu1:31337')::uuid::text;

\echo === R1b same, worst-case user (200 groups) ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT p.id, p.enabled, p.federated_links,
       ARRAY(SELECT gm.group_id FROM group_members gm
             WHERE gm.tenant_id = p.tenant_id AND gm.principal_id = p.id ORDER BY gm.group_id) AS groups
FROM principals p
WHERE p.tenant_id = md5('t1')::uuid AND p.keycloak_user_id = md5('kcu1:1')::uuid::text;

\echo === R1c same as R1, RLS as the only tenant filter ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT p.id, p.enabled, p.federated_links,
       ARRAY(SELECT gm.group_id FROM group_members gm
             WHERE gm.principal_id = p.id ORDER BY gm.group_id) AS groups
FROM principals p
WHERE p.keycloak_user_id = md5('kcu1:31337')::uuid::text;

\echo === R1d unknown user (no row yet: the resolver returns 401) ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT p.id, p.enabled, p.federated_links
FROM principals p
WHERE p.tenant_id = md5('t1')::uuid AND p.keycloak_user_id = 'no-such-user';
COMMIT;

BEGIN;
SELECT set_config('app.tenant_id', md5('t150')::uuid::text, true) AS tenant;
\echo === R1e resolver lookup, small tenant ===
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT p.id, p.enabled, p.federated_links,
       ARRAY(SELECT gm.group_id FROM group_members gm
             WHERE gm.tenant_id = p.tenant_id AND gm.principal_id = p.id ORDER BY gm.group_id) AS groups
FROM principals p
WHERE p.tenant_id = md5('t150')::uuid AND p.keycloak_user_id = md5('kcu150:42')::uuid::text;
COMMIT;
