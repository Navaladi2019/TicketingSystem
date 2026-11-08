\set ON_ERROR_STOP 0
-- Round 2: does D10 step (i) (D1 order + D5 policy + D11 FK) work as written,
-- and does the pre-(i) sync binary still work against the step-(i) schema?
CREATE ROLE own NOLOGIN; CREATE ROLE nava_groupsync LOGIN; CREATE ROLE nava_app LOGIN;
GRANT USAGE, CREATE ON SCHEMA public TO own;
SET ROLE own;
-- Schema as on the branch today (00003, tenant_admin_group, tenant_auditor_group, records FK SET NULL).
CREATE TABLE groups (tenant_id uuid NOT NULL, id uuid NOT NULL, keycloak_group_id text NOT NULL, path text NOT NULL,
  PRIMARY KEY (tenant_id, id), UNIQUE (tenant_id, keycloak_group_id));
CREATE TABLE group_members (tenant_id uuid NOT NULL, group_id uuid NOT NULL, principal_id uuid NOT NULL,
  PRIMARY KEY (tenant_id, group_id, principal_id),
  CONSTRAINT group_members_group_fkey FOREIGN KEY (tenant_id, group_id) REFERENCES groups (tenant_id, id) ON DELETE CASCADE);
CREATE INDEX group_members_principal ON group_members (tenant_id, principal_id);
CREATE TABLE tenant_admin_group (tenant_id uuid PRIMARY KEY, group_id uuid NOT NULL,
  FOREIGN KEY (tenant_id, group_id) REFERENCES groups (tenant_id, id) ON DELETE CASCADE);
CREATE TABLE tenant_auditor_group (tenant_id uuid PRIMARY KEY, group_id uuid NOT NULL,
  FOREIGN KEY (tenant_id, group_id) REFERENCES groups (tenant_id, id) ON DELETE CASCADE);
CREATE TABLE records (tenant_id uuid NOT NULL, id uuid NOT NULL, assignment_group_id uuid, PRIMARY KEY (tenant_id, id),
  FOREIGN KEY (tenant_id, assignment_group_id) REFERENCES groups (tenant_id, id) ON DELETE SET NULL (assignment_group_id));
ALTER TABLE groups ENABLE ROW LEVEL SECURITY; ALTER TABLE groups FORCE ROW LEVEL SECURITY;
ALTER TABLE group_members ENABLE ROW LEVEL SECURITY; ALTER TABLE group_members FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_admin_group ENABLE ROW LEVEL SECURITY; ALTER TABLE tenant_admin_group FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_auditor_group ENABLE ROW LEVEL SECURITY; ALTER TABLE tenant_auditor_group FORCE ROW LEVEL SECURITY;
CREATE POLICY groups_select ON groups FOR SELECT TO nava_app, nava_groupsync USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY groups_insert ON groups FOR INSERT TO nava_groupsync WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY groups_update ON groups FOR UPDATE TO nava_groupsync USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY groups_delete ON groups FOR DELETE TO nava_groupsync USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY gm_select ON group_members FOR SELECT TO nava_app, nava_groupsync USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY gm_insert ON group_members FOR INSERT TO nava_groupsync WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY gm_delete ON group_members FOR DELETE TO nava_groupsync USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tag_select ON tenant_admin_group FOR SELECT TO nava_app, nava_groupsync USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- branch: tenant_auditor_group_select is TO nava_app, nava_worker only; nava_groupsync has no grant on it
CREATE POLICY taud_select ON tenant_auditor_group FOR SELECT TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT, DELETE ON groups, group_members TO nava_groupsync; GRANT UPDATE (path) ON groups TO nava_groupsync;
GRANT SELECT ON tenant_admin_group TO nava_groupsync;
RESET ROLE;

-- Data: 10 tenants x 5,000 groups x 50 members (2.5M memberships), plus tenant T.
INSERT INTO groups SELECT md5('t'||t)::uuid, md5('t'||t||'g'||g)::uuid, 'kc'||t||'-'||g, '/t/x/'||g FROM generate_series(1,10) t, generate_series(1,5000) g;
INSERT INTO group_members SELECT md5('t'||t)::uuid, md5('t'||t||'g'||g)::uuid, md5('p'||t||'-'||((g*7+m) % 20000))::uuid
  FROM generate_series(1,10) t, generate_series(1,5000) g, generate_series(1,50) m ON CONFLICT DO NOTHING;
INSERT INTO groups VALUES
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a','kc-admins','/t/x/admins'),
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000b','kc-auditors','/t/x/auditors'),
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000c','kc-old','/t/x/old'),
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000d','kc-live','/t/x/live');
INSERT INTO group_members VALUES
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a','00000000-0000-0000-0000-0000000000e1'),
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000b','00000000-0000-0000-0000-0000000000e1'),
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000c','00000000-0000-0000-0000-0000000000e1'),
  ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000d','00000000-0000-0000-0000-0000000000e1');
INSERT INTO tenant_admin_group VALUES ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a');
INSERT INTO tenant_auditor_group VALUES ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000b');
INSERT INTO records VALUES ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-0000000000f1','00000000-0000-0000-0000-00000000000c');
ANALYZE;
SELECT (SELECT count(*) FROM groups) AS groups_rows, (SELECT count(*) FROM group_members) AS member_rows;

\echo '================ STEP (i) MIGRATION, as D10 lists it (timed) ================'
\timing on
SET ROLE own;
BEGIN;
SET LOCAL lock_timeout = '2s';
ALTER TABLE groups ADD COLUMN deleted_at timestamptz;
ALTER TABLE groups ADD COLUMN live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED;
SELECT relfilenode AS groups_relfilenode_after_live FROM pg_class WHERE relname = 'groups';
ALTER TABLE groups ADD CONSTRAINT groups_tenant_id_live_key UNIQUE (tenant_id, id, live);
ALTER TABLE group_members ADD COLUMN live boolean NOT NULL DEFAULT true CONSTRAINT group_members_live_check CHECK (live);
ALTER TABLE group_members ADD CONSTRAINT group_members_group_live_fkey FOREIGN KEY (tenant_id, group_id, live)
  REFERENCES groups (tenant_id, id, live) NOT VALID;
SELECT mode, relation::regclass FROM pg_locks WHERE pid = pg_backend_pid() AND relation::regclass::text IN ('groups','group_members') AND granted ORDER BY 2,1;
COMMIT;
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
\timing off
-- D5 policies + grants (D5 as written: DELETE + policy; SELECT on tenant_auditor_group NOT added, per rev 1 text)
GRANT UPDATE (deleted_at) ON groups TO nava_groupsync;
GRANT DELETE ON tenant_admin_group, tenant_auditor_group TO nava_groupsync;
CREATE POLICY tenant_admin_group_delete ON tenant_admin_group FOR DELETE TO nava_groupsync
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
         AND EXISTS (SELECT 1 FROM groups g WHERE g.tenant_id = tenant_admin_group.tenant_id AND g.id = group_id AND g.deleted_at IS NOT NULL));
CREATE POLICY tenant_auditor_group_delete ON tenant_auditor_group FOR DELETE TO nava_groupsync
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
         AND EXISTS (SELECT 1 FROM groups g WHERE g.tenant_id = tenant_auditor_group.tenant_id AND g.id = group_id AND g.deleted_at IS NOT NULL));
RESET ROLE;

\echo '================ R1: OLD binary after step (i): hard DELETE of kc-old (members still present; old sync deletes memberships first, here we also try without) ================'
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
DELETE FROM groups WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND id = ANY(ARRAY['00000000-0000-0000-0000-00000000000c']::uuid[]);
COMMIT;
\c - postgres
SELECT (SELECT count(*) FROM groups WHERE id='00000000-0000-0000-0000-00000000000c') AS old_group_rows,
       (SELECT count(*) FROM group_members WHERE group_id='00000000-0000-0000-0000-00000000000c') AS old_member_rows,
       (SELECT assignment_group_id FROM records WHERE id='00000000-0000-0000-0000-0000000000f1') AS record_group;

\echo '================ R2: NEW binary, D1 order for kc-admins: (1) delete members (2) tombstone (3) delete pointer ================'
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
DELETE FROM group_members WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = ANY(ARRAY['00000000-0000-0000-0000-00000000000a']::uuid[]);
UPDATE groups SET deleted_at = now() WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND id = ANY(ARRAY['00000000-0000-0000-0000-00000000000a']::uuid[]);
DELETE FROM tenant_admin_group WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = ANY(ARRAY['00000000-0000-0000-0000-00000000000a']::uuid[]);
COMMIT;
\c - postgres
SELECT (SELECT count(*) FROM tenant_admin_group) AS admin_ptr_rows_after_R2;

\echo '================ R2b: same order for kc-auditors, D5 grant+policy only (rev 1 text): no SELECT for nava_groupsync on tenant_auditor_group ================'
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
DELETE FROM group_members WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = '00000000-0000-0000-0000-00000000000b';
UPDATE groups SET deleted_at = now() WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND id = '00000000-0000-0000-0000-00000000000b';
DELETE FROM tenant_auditor_group WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = ANY(ARRAY['00000000-0000-0000-0000-00000000000b']::uuid[]);
ROLLBACK;

\echo '================ R2c: add SELECT grant only (no SELECT policy for nava_groupsync) ================'
\c - postgres
GRANT SELECT ON tenant_auditor_group TO nava_groupsync;
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
DELETE FROM group_members WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = '00000000-0000-0000-0000-00000000000b';
UPDATE groups SET deleted_at = now() WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND id = '00000000-0000-0000-0000-00000000000b';
DELETE FROM tenant_auditor_group WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = ANY(ARRAY['00000000-0000-0000-0000-00000000000b']::uuid[]);
ROLLBACK;

\echo '================ R3: SELECT grant + SELECT policy TO nava_groupsync on tenant_auditor_group ================'
\c - postgres
ALTER POLICY taud_select ON tenant_auditor_group TO nava_app, nava_groupsync;
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
DELETE FROM group_members WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = '00000000-0000-0000-0000-00000000000b';
UPDATE groups SET deleted_at = now() WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND id = '00000000-0000-0000-0000-00000000000b';
DELETE FROM tenant_auditor_group WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND group_id = ANY(ARRAY['00000000-0000-0000-0000-00000000000b']::uuid[]);
COMMIT;
\c - postgres
SELECT (SELECT count(*) FROM tenant_auditor_group) AS auditor_ptr_rows_after_R3;

\echo '================ R4: D5 policy refuses a LIVE pointer (kc-live recorded as admin) ================'
INSERT INTO tenant_admin_group VALUES ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000d');
\c - nava_groupsync
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
DELETE FROM tenant_admin_group WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001';
COMMIT;

\echo '================ R5: wrong order (tombstone kc-live before deleting its member) is refused by D11 ================'
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
UPDATE groups SET deleted_at = now() WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND id = '00000000-0000-0000-0000-00000000000d';
ROLLBACK;

\echo '================ R6: wrong order on revive (member insert before deleted_at = NULL) is refused; right order succeeds ================'
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a','00000000-0000-0000-0000-0000000000e1');
ROLLBACK;
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
UPDATE groups SET deleted_at = NULL WHERE tenant_id = 'd5f1e2b0-0000-0000-0000-000000000001' AND id = '00000000-0000-0000-0000-00000000000a';
INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a','00000000-0000-0000-0000-0000000000e1');
COMMIT;

\echo '================ R7: OLD binary during rollout: tombstoned kc-auditors back in Keycloak; old code does not revive, inserts member ================'
BEGIN; SELECT set_config('app.tenant_id','d5f1e2b0-0000-0000-0000-000000000001',true) \g /dev/null
INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ('d5f1e2b0-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000b','00000000-0000-0000-0000-0000000000e1');
ROLLBACK;

\echo '================ R8: resolver reader with D11 + D4 filter ================'
\c - postgres
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT g.path FROM group_members gm JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id
 WHERE gm.tenant_id = md5('t1')::uuid AND gm.principal_id = md5('p1-77')::uuid AND g.deleted_at IS NULL;
SELECT version();
