\set ON_ERROR_STOP 0
\echo '=== seed (superuser) ==='
INSERT INTO groups (tenant_id, id, path) SELECT '11111111-1111-1111-1111-111111111111', ('00000000-0000-0000-0000-00000000000' || n)::uuid, '/g' || n FROM generate_series(1,6) n;
INSERT INTO records SELECT '11111111-1111-1111-1111-111111111111', ('00000000-0000-0000-0000-0000000000f' || n)::uuid, 7,
       ('00000000-0000-0000-0000-00000000000' || n)::uuid, '2026-01-01' FROM generate_series(1,6) n;
SELECT id, version, assignment_group_id, updated_at FROM records ORDER BY id;

\echo '=== S0: no commit check attached. nava_groupsync deletes g1 (no grant on records; no RLS policy for the owner on records under FORCE) ==='
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
SELECT has_table_privilege('records','UPDATE') AS groupsync_may_update_records;
DELETE FROM groups WHERE id = '00000000-0000-0000-0000-000000000001';
COMMIT;
\c - postgres
SELECT id, version, assignment_group_id, updated_at FROM records WHERE id = '00000000-0000-0000-0000-0000000000f1';

\echo '=== attach commit check as 20261104100000 (no WHEN) ==='
CREATE CONSTRAINT TRIGGER records_audit_commit_check AFTER INSERT OR DELETE OR UPDATE OF version, assignment_group_id, updated_at
  ON records DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION audit_commit_check();
\echo '=== S1: nava_groupsync deletes g2 with the check attached ==='
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
DELETE FROM groups WHERE id = '00000000-0000-0000-0000-000000000002';
\echo '--- DELETE statement returned; now COMMIT'
COMMIT;
\c - postgres
SELECT id, version, assignment_group_id FROM records WHERE id = '00000000-0000-0000-0000-0000000000f2';
SELECT count(*) AS g2_still_exists FROM groups WHERE id = '00000000-0000-0000-0000-000000000002';

\echo '=== option (c): same trigger WHEN (pg_trigger_depth() = 0) ==='
DROP TRIGGER records_audit_commit_check ON records;
CREATE CONSTRAINT TRIGGER records_audit_commit_check AFTER INSERT OR DELETE OR UPDATE OF version, assignment_group_id, updated_at
  ON records DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (pg_trigger_depth() = 0) EXECUTE FUNCTION audit_commit_check();
\echo '=== S2: nava_groupsync deletes g3 under (c) ==='
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
DELETE FROM groups WHERE id = '00000000-0000-0000-0000-000000000003';
COMMIT;
\c - postgres
SELECT id, version, assignment_group_id, updated_at, (SELECT count(*) FROM audit_log) AS audit_rows FROM records WHERE id = '00000000-0000-0000-0000-0000000000f3';

\echo '=== S3: under (c), a raw nava_app UPDATE at depth 0 is still caught ==='
\c - nava_app
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
UPDATE records SET assignment_group_id = '00000000-0000-0000-0000-000000000004' WHERE id = '00000000-0000-0000-0000-0000000000f3';
COMMIT;

\echo '=== S4: under (c), ANY write issued from inside a trigger escapes: nava_app inserts into an unaudited table whose trigger updates records ==='
\c - postgres
SET ROLE own;
CREATE TABLE side (tenant_id uuid, record_id uuid);
ALTER TABLE side ENABLE ROW LEVEL SECURITY; ALTER TABLE side FORCE ROW LEVEL SECURITY;
CREATE POLICY s_app ON side FOR ALL TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT INSERT ON side TO nava_app;
CREATE FUNCTION side_fn() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER AS $f$
BEGIN UPDATE records SET assignment_group_id = NULL WHERE tenant_id = NEW.tenant_id AND id = NEW.record_id; RETURN NULL; END $f$;
CREATE TRIGGER side_t AFTER INSERT ON side FOR EACH ROW EXECUTE FUNCTION side_fn();
RESET ROLE;
\c - nava_app
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
INSERT INTO side VALUES ('11111111-1111-1111-1111-111111111111', '00000000-0000-0000-0000-0000000000f4');
COMMIT;
\c - postgres
SELECT id, version, assignment_group_id, (SELECT count(*) FROM audit_log) AS audit_rows FROM records WHERE id = '00000000-0000-0000-0000-0000000000f4';

\echo '=== R29 query (proposed), before the FK change ==='
SELECT c.conrelid::regclass AS audited_table, c.conname, c.confdeltype, c.confupdtype, c.confdelsetcols
  FROM pg_constraint c
 WHERE c.contype = 'f' AND c.conrelid = ANY (ARRAY['records'::regclass])
   AND (c.confdeltype NOT IN ('a','r') OR c.confupdtype NOT IN ('a','r'));

\echo '=== option (e): FK -> RESTRICT, trigger without WHEN; tombstone instead of delete ==='
DROP TRIGGER records_audit_commit_check ON records;
CREATE CONSTRAINT TRIGGER records_audit_commit_check AFTER INSERT OR DELETE OR UPDATE OF version, assignment_group_id, updated_at
  ON records DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION audit_commit_check();
BEGIN;
SET LOCAL lock_timeout = '2s';
ALTER TABLE records DROP CONSTRAINT records_assignment_group_fkey,
  ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
    REFERENCES groups (tenant_id, id) ON DELETE RESTRICT NOT VALID;
COMMIT;
ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;
REVOKE DELETE ON groups FROM nava_groupsync; GRANT UPDATE (deleted_at) ON groups TO nava_groupsync;
\echo '--- R29 query after the change (expect 0 rows)'
SELECT c.conrelid::regclass, c.conname, c.confdeltype, c.confupdtype FROM pg_constraint c
 WHERE c.contype = 'f' AND c.conrelid = 'records'::regclass AND (c.confdeltype NOT IN ('a','r') OR c.confupdtype NOT IN ('a','r'));
\echo '=== S5: nava_groupsync tombstones g5 (commits; no records event) and tries DELETE g6 (denied) ==='
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
UPDATE groups SET deleted_at = now() WHERE id = '00000000-0000-0000-0000-000000000005';
COMMIT;
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
DELETE FROM groups WHERE id = '00000000-0000-0000-0000-000000000006';
ROLLBACK;
\c - postgres
\echo '=== S6: superuser hard-deletes referenced g6 -> RESTRICT ==='
DELETE FROM groups WHERE id = '00000000-0000-0000-0000-000000000006';
\echo '=== S7: FK accepts a NEW reference to tombstoned g5 (only the D6 pipeline check can stop it) ==='
\c - nava_app
BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
INSERT INTO audit_log VALUES ('11111111-1111-1111-1111-111111111111', pg_current_xact_id(), now(), 'record', '00000000-0000-0000-0000-0000000000f6');
UPDATE records SET assignment_group_id = '00000000-0000-0000-0000-000000000005', version = version + 1 WHERE id = '00000000-0000-0000-0000-0000000000f6';
COMMIT;
\c - postgres
SELECT id, version, assignment_group_id FROM records ORDER BY id;
SELECT version();
