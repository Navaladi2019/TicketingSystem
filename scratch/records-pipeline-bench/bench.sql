-- P3-09 (TICK-353) evidence. Run by run.sh against a throwaway postgres (the
-- docs/pins.md tag; 16.10 when recorded). Prerequisite tables are reduced to
-- the keys the migration references.
-- 1. The migration on 800,000 bare (tenant_id, id) rows, 200 tenants: time
--    (the ACCESS EXCLUSIVE window) and relfilenode before and after.
-- 2. 800,000 pipeline rows after: plans for the audit history, the replay
--    lookup, the counter upsert and the group-delete foreign-key action.
\timing off
CREATE ROLE nava_app; CREATE ROLE nava_worker;
CREATE TABLE tenants (id uuid PRIMARY KEY, state text NOT NULL);
CREATE TABLE principals (tenant_id uuid NOT NULL, id uuid NOT NULL, PRIMARY KEY (tenant_id, id));
CREATE TABLE groups (tenant_id uuid NOT NULL, id uuid NOT NULL, PRIMARY KEY (tenant_id, id));
CREATE TABLE entity_definitions (tenant_id uuid NOT NULL, id uuid NOT NULL, PRIMARY KEY (tenant_id, id));
CREATE TABLE records (tenant_id uuid NOT NULL, id uuid NOT NULL, version bigint NOT NULL DEFAULT 0,
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id));
ALTER TABLE records ENABLE ROW LEVEL SECURITY;
ALTER TABLE records FORCE ROW LEVEL SECURITY;

INSERT INTO tenants SELECT md5('t' || g)::uuid, 'active' FROM generate_series(1, 200) g;
INSERT INTO principals SELECT t.id, md5(t.id::text || 'p' || p)::uuid FROM tenants t, generate_series(1, 50) p;
INSERT INTO groups SELECT t.id, md5(t.id::text || 'g' || g)::uuid FROM tenants t, generate_series(1, 20) g;
INSERT INTO entity_definitions SELECT id, id FROM tenants;
INSERT INTO records (tenant_id, id) SELECT t.id, gen_random_uuid() FROM tenants t, generate_series(1, 4000);
VACUUM ANALYZE;

\echo '== 1. migration on 800,000 bare rows'
SELECT relfilenode AS before_relfilenode, pg_size_pretty(pg_table_size('records')) AS heap FROM pg_class WHERE relname = 'records';
\timing on
BEGIN;
\i /bench/migration.sql
COMMIT;
\timing off
SELECT relfilenode AS after_relfilenode, pg_size_pretty(pg_table_size('records')) AS heap FROM pg_class WHERE relname = 'records';

\echo '== 2. pipeline rows'
TRUNCATE records CASCADE;
INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, requester_id,
                     assignment_group_id, created_by, created_at, updated_at, data)
SELECT t.id, gen_random_uuid(), t.id, 1, 'TCK' || lpad(n::text, 7, '0'),
       CASE WHEN n % 5 = 0 THEN 'open' ELSE 'closed' END,
       md5(t.id::text || 'p' || (n % 50 + 1))::uuid, md5(t.id::text || 'g' || (n % 20 + 1))::uuid,
       md5(t.id::text || 'p' || (n % 50 + 1))::uuid,
       now() - n * interval '1 minute', now() - n * interval '1 minute', '{"title": "x"}'
FROM tenants t, generate_series(1, 4000) n;
-- Five audit rows per record, the first keyed.
INSERT INTO audit_log (tenant_id, id, record_id, action, source, after, idempotency_key, at)
SELECT r.tenant_id, gen_random_uuid(), r.id, CASE a WHEN 1 THEN 'create' ELSE 'update' END,
       'workflow', '{}', CASE a WHEN 1 THEN 'wf:' || r.id || ':s1' END, r.created_at + a * interval '1 second'
FROM records r, generate_series(1, 5) a;
INSERT INTO record_counters SELECT id, id, 4001 FROM tenants;
VACUUM ANALYZE;
SELECT c.relname, pg_size_pretty(pg_relation_size(c.oid)) AS size
FROM pg_class c WHERE c.relname IN ('records', 'records_pkey', 'records_number_key', 'records_tenant_updated',
    'records_tenant_requester', 'records_tenant_group_open', 'audit_log', 'audit_log_pkey', 'audit_log_record',
    'audit_log_idempotency_key', 'record_counters')
ORDER BY c.relname;

SELECT tenant_id AS t, id AS r FROM records ORDER BY tenant_id, id LIMIT 1 \gset
\echo '-- a record''s history'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY ON)
SELECT * FROM audit_log WHERE tenant_id = :'t' AND record_id = :'r' ORDER BY at, id;
\echo '-- the replay lookup (D-3-13)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY ON)
SELECT id FROM audit_log WHERE tenant_id = :'t' AND idempotency_key = 'wf:' || :'r' || ':s1';
\echo '-- the counter upsert (D-3-9)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY ON)
INSERT INTO record_counters VALUES (:'t', :'t', 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
\echo '-- what the ON DELETE SET NULL action runs per deleted group (4,000-record tenant)'
SELECT md5(:'t'::text || 'g1')::uuid AS g \gset
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY ON)
UPDATE ONLY records SET assignment_group_id = NULL WHERE tenant_id = :'t' AND assignment_group_id = :'g';
\echo '-- the group delete itself, as group sync issues it'
\timing on
BEGIN;
DELETE FROM groups WHERE tenant_id = :'t' AND id = md5(:'t'::text || 'g2')::uuid;
ROLLBACK;
\timing off
