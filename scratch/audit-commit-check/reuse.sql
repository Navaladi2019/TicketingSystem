\set ON_ERROR_STOP 1
DROP TABLE IF EXISTS p511_audited;
CREATE TABLE p511_audited (tenant_id uuid NOT NULL, id uuid NOT NULL, val text, PRIMARY KEY (tenant_id, id));
GRANT SELECT, INSERT ON p511_audited TO nava_worker;
CREATE CONSTRAINT TRIGGER p511_audited_audit_commit_check AFTER INSERT OR DELETE OR UPDATE OF val ON p511_audited
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record', 'id');
DELETE FROM audit_log WHERE tenant_id::text LIKE '0190d000%'; DELETE FROM tenants WHERE id::text LIKE '0190d000%';
INSERT INTO tenants (id, state) SELECT ('0190d000-0000-7000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 'active' FROM generate_series(1, 4) g;
DROP TABLE audit_log_p511_young;
CREATE TABLE audit_log_p511_young PARTITION OF audit_log FOR VALUES FROM (date_trunc('month', now())) TO (date_trunc('month', now()) + interval '1 month') WITH (autovacuum_enabled = false);
ALTER TABLE audit_log_p511_young ALTER COLUMN occurred_at SET (n_distinct = 1);
SET session_replication_role = replica;
INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, actor_system, source)
 SELECT ('0190b000-0000-7000-8000-' || lpad(to_hex(1 + g % 20), 12, '0'))::uuid, gen_random_uuid(), date_trunc('month', now()) + (now() - date_trunc('month', now())) * (g::float8/61), (3000000+g)::text::xid8, 'record','update','record', gen_random_uuid(), 'workflow','workflow' FROM generate_series(1,60) g;
SET session_replication_role = DEFAULT;
ANALYZE audit_log;
SELECT relpages, reltuples FROM pg_class WHERE relname = 'audit_log_p511_young';
SET ROLE nava_worker;
SET plan_cache_mode = force_generic_plan;
\timing on
-- warm: a 1-subject write caches the trigger's generic plan while the partition is 2 pages
BEGIN; SELECT set_config('app.tenant_id', '0190d000-0000-7000-8000-000000000001', true) \g /dev/null
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
  VALUES ('0190d000-0000-7000-8000-000000000001', gen_random_uuid(), 'record', 'create', 'record', '00000000-0000-4000-8000-000000000001', 1, 'workflow', 'workflow');
INSERT INTO p511_audited VALUES ('0190d000-0000-7000-8000-000000000001', '00000000-0000-4000-8000-000000000001', 'v');
COMMIT;
\echo '--- same session: import of 1,000 then 4,000 subjects (COMMIT times)'
BEGIN; SELECT set_config('app.tenant_id', '0190d000-0000-7000-8000-000000000002', true) \g /dev/null
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
  SELECT '0190d000-0000-7000-8000-000000000002', gen_random_uuid(), 'record', 'create', 'record', md5('b' || g)::uuid, 1, 'workflow', 'workflow' FROM generate_series(1, 1000) g;
INSERT INTO p511_audited SELECT '0190d000-0000-7000-8000-000000000002', md5('b' || g)::uuid, 'v' FROM generate_series(1, 1000) g;
COMMIT;
BEGIN; SELECT set_config('app.tenant_id', '0190d000-0000-7000-8000-000000000003', true) \g /dev/null
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
  SELECT '0190d000-0000-7000-8000-000000000003', gen_random_uuid(), 'record', 'create', 'record', md5('c' || g)::uuid, 1, 'workflow', 'workflow' FROM generate_series(1, 4000) g;
INSERT INTO p511_audited SELECT '0190d000-0000-7000-8000-000000000003', md5('c' || g)::uuid, 'v' FROM generate_series(1, 4000) g;
COMMIT;
