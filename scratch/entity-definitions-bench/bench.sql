-- P3-02 (TICK-340) plan evidence for entity_definitions and
-- entity_definition_versions. Run by run.sh against a throwaway postgres
-- (the docs/pins.md tag; 16.10 when recorded).
-- 2,000 tenants x 10 types x 5 versions = 20,000 types, 100,000 versions,
-- each schema ~1.6 kB of json (stored as written).
CREATE ROLE nava_app; CREATE ROLE nava_worker; CREATE ROLE nava_onboard;
CREATE TABLE tenants (id uuid PRIMARY KEY, state text NOT NULL);
CREATE TABLE principals (tenant_id uuid NOT NULL, id uuid NOT NULL, PRIMARY KEY (tenant_id, id));
\i /bench/migration.sql
GRANT SELECT ON tenants TO nava_onboard;
GRANT SELECT, INSERT ON entity_definitions, entity_definition_versions TO nava_app;
GRANT UPDATE (current_version, updated_at) ON entity_definitions TO nava_app;

INSERT INTO tenants SELECT md5('t' || g)::uuid, 'active' FROM generate_series(1, 2000) g;
INSERT INTO principals SELECT id, id FROM tenants;
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
SELECT t.id, md5(t.id::text || k)::uuid, 'type_' || k, 'Type ' || k,
       'T' || chr(64 + k) || 'X', 5
FROM tenants t, generate_series(1, 10) k;
INSERT INTO entity_definition_versions
    (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
SELECT d.tenant_id, md5(d.id::text || v)::uuid, d.id, v, s.schema::json,
       sha256(convert_to(s.schema, 'UTF8')), '[]', '["open","closed"]',
       CASE WHEN v = 1 THEN 'initial' ELSE 'compatible' END, d.tenant_id
FROM entity_definitions d, generate_series(1, 5) v,
     LATERAL (SELECT '{"type": "object", "properties": {' ||
        string_agg('"f' || f || '_' || v || '": {"type": "string", "maxLength": 200}', ', ') || '}}' AS schema
        FROM generate_series(1, 40) f) s;
VACUUM ANALYZE;

\echo '== sizes'
SELECT relname, pg_size_pretty(pg_total_relation_size(oid)) total, pg_size_pretty(pg_relation_size(oid)) heap
FROM pg_class WHERE relname IN ('entity_definitions', 'entity_definition_versions');
SELECT indexrelid::regclass, pg_size_pretty(pg_relation_size(indexrelid))
FROM pg_index WHERE indrelid IN ('entity_definitions'::regclass, 'entity_definition_versions'::regclass);

SET ROLE nava_app;
SELECT set_config('app.tenant_id', md5('t1000')::uuid::text, false);
\echo '== Q1 type by key, with its current version (record create/read path)'
EXPLAIN (ANALYZE, BUFFERS)
SELECT v.id, v.schema, v.schema_sha256 FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id AND v.version = d.current_version
WHERE d.tenant_id = md5('t1000')::uuid AND d.key = 'type_7';
\echo '== Q2 version by id (interpreter, D-3-12)'
EXPLAIN (ANALYZE, BUFFERS)
SELECT schema FROM entity_definition_versions
WHERE tenant_id = md5('t1000')::uuid AND id = md5(md5(md5('t1000')::uuid::text || 7)::uuid::text || 3)::uuid;
\echo '== Q3 list a tenant types'
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, key, name, number_prefix, current_version FROM entity_definitions
WHERE tenant_id = md5('t1000')::uuid ORDER BY key;
\echo '== Q4 version history of one type'
EXPLAIN (ANALYZE, BUFFERS)
SELECT version, compat, published_at FROM entity_definition_versions
WHERE tenant_id = md5('t1000')::uuid AND definition_id = md5(md5('t1000')::uuid::text || 7)::uuid ORDER BY version;
