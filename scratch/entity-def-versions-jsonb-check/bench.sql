-- TICK-378. 2,000 tenants x 10 types x 5 versions = 100,000 versions with
-- object-shaped fixed_columns and states (as the API writes them), each
-- schema ~1.6 kB. Then: the cost of an insert batch without the CHECKs, the
-- lock-holding ADD ... NOT VALID, the VALIDATE scan, and the same insert
-- batch with the CHECKs.
CREATE ROLE nava_app; CREATE ROLE nava_worker; CREATE ROLE nava_onboard;
CREATE TABLE tenants (id uuid PRIMARY KEY, state text NOT NULL);
CREATE TABLE principals (tenant_id uuid NOT NULL, id uuid NOT NULL, PRIMARY KEY (tenant_id, id));
\i /bench/m1.sql
INSERT INTO tenants SELECT md5('t' || g)::uuid, 'active' FROM generate_series(1, 2000) g;
INSERT INTO principals SELECT id, id FROM tenants;
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
SELECT t.id, md5(t.id::text || k)::uuid, 'type_' || k, 'Type ' || k, 'T' || chr(64 + k) || 'X', 5
FROM tenants t, generate_series(1, 10) k;
CREATE TABLE schemas AS
SELECT v, '{"type": "object", "properties": {' ||
       string_agg('"f' || f || '_' || v || '": {"type": "string", "maxLength": 200}', ', ') || '}}' AS schema
FROM generate_series(1, 10) v, generate_series(1, 40) f GROUP BY v;
-- versions lo..hi of every type
PREPARE ins(int, int) AS
INSERT INTO entity_definition_versions
    (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
SELECT d.tenant_id, md5(d.id::text || s.v)::uuid, d.id, s.v, s.schema::json,
       sha256(convert_to(s.schema, 'UTF8')), '{"state": "state", "assignee": "assignee_id"}',
       '{"initial": "open", "states": ["open", "closed"], "transitions": [{"from": "open", "to": "closed"}]}',
       CASE WHEN s.v = 1 THEN 'initial' ELSE 'compatible' END, d.tenant_id
FROM entity_definitions d JOIN schemas s ON s.v BETWEEN $1 AND $2;
EXECUTE ins(1, 5);
VACUUM ANALYZE;
SELECT count(*) AS versions FROM entity_definition_versions;

\timing on
\echo '== insert 20,000 versions (v6), no CHECK: 3 runs, rolled back'
BEGIN; EXECUTE ins(6, 6); ROLLBACK;
BEGIN; EXECUTE ins(6, 6); ROLLBACK;
BEGIN; EXECUTE ins(6, 6); ROLLBACK;

\echo '== migration 1: ADD CONSTRAINT ... NOT VALID (ACCESS EXCLUSIVE held for this long)'
BEGIN;
\i /bench/m2.sql
SELECT mode FROM pg_locks WHERE relation = 'entity_definition_versions'::regclass AND pid = pg_backend_pid();
COMMIT;

\echo '== migration 2: VALIDATE (SHARE UPDATE EXCLUSIVE)'
BEGIN;
\i /bench/m3.sql
SELECT mode FROM pg_locks WHERE relation = 'entity_definition_versions'::regclass AND pid = pg_backend_pid();
COMMIT;

\echo '== insert 20,000 versions (v6), with both CHECKs: 3 runs, rolled back'
BEGIN; EXECUTE ins(6, 6); ROLLBACK;
BEGIN; EXECUTE ins(6, 6); ROLLBACK;
BEGIN; EXECUTE ins(6, 6); ROLLBACK;

\echo '== the CHECK expression alone over 100,000 rows'
EXPLAIN (ANALYZE, BUFFERS) SELECT count(*) FROM entity_definition_versions
WHERE jsonb_typeof(fixed_columns) = 'object' AND jsonb_typeof(states) = 'object';
\timing off
\echo '== a [] insert is refused'
\set ON_ERROR_STOP 0
INSERT INTO entity_definition_versions
    (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
SELECT tenant_id, gen_random_uuid(), id, 99, '{}', sha256('{}'), '[]', '{}', 'compatible', tenant_id
FROM entity_definitions LIMIT 1;
\echo '== sizes (no index added)'
SELECT pg_size_pretty(pg_total_relation_size('entity_definition_versions')) total;
