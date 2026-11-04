\set ON_ERROR_STOP 1
CREATE ROLE own NOLOGIN;            -- table owner (stands in for nava_migrator)
CREATE ROLE nava_groupsync LOGIN;
CREATE ROLE nava_app LOGIN;
GRANT USAGE, CREATE ON SCHEMA public TO own;
SET ROLE own;
CREATE TABLE groups (tenant_id uuid NOT NULL, id uuid NOT NULL, path text NOT NULL,
  deleted_at timestamptz, PRIMARY KEY (tenant_id, id));
CREATE TABLE records (tenant_id uuid NOT NULL, id uuid NOT NULL, version int NOT NULL DEFAULT 1,
  assignment_group_id uuid, updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, id),
  CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
    REFERENCES groups (tenant_id, id) ON DELETE SET NULL (assignment_group_id));
CREATE TABLE audit_log (tenant_id uuid, txid xid8, occurred_at timestamptz, subject_type text, subject_id uuid);
-- records: FORCE RLS with a canonical policy for nava_app only; the owner has NO policy at all.
ALTER TABLE records ENABLE ROW LEVEL SECURITY; ALTER TABLE records FORCE ROW LEVEL SECURITY;
CREATE POLICY records_app ON records FOR ALL TO nava_app
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
ALTER TABLE groups ENABLE ROW LEVEL SECURITY; ALTER TABLE groups FORCE ROW LEVEL SECURITY;
CREATE POLICY g_sync ON groups FOR ALL TO nava_groupsync
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY; ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY a_app ON audit_log FOR ALL TO nava_app
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- grants as db/grants.yaml: groupsync has nothing on records or audit_log
GRANT SELECT, INSERT, DELETE ON groups TO nava_groupsync; GRANT UPDATE (path) ON groups TO nava_groupsync;
GRANT SELECT, INSERT, UPDATE ON records TO nava_app; GRANT SELECT, INSERT ON audit_log TO nava_app;
GRANT SELECT ON groups TO nava_app;
-- probe: who runs the RI action's UPDATE, and is RLS applied to it?
CREATE FUNCTION probe() RETURNS trigger LANGUAGE plpgsql AS $f$
BEGIN
  RAISE NOTICE 'PROBE % % on records: current_user=% session_user=% depth=% row_security=% old.grp=% new.grp=%',
    TG_WHEN, TG_OP, current_user, session_user, pg_trigger_depth(), current_setting('row_security'),
    OLD.assignment_group_id, NEW.assignment_group_id;
  RETURN NEW;
END $f$;
CREATE TRIGGER records_probe BEFORE UPDATE ON records FOR EACH ROW EXECUTE FUNCTION probe();
-- the real commit check, reduced to its decision (same query shape as 20261103100000)
CREATE FUNCTION audit_commit_check() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER AS $f$
BEGIN
  RAISE NOTICE 'CHECK fires: current_user=% depth=%', current_user, pg_trigger_depth();
  PERFORM 1 FROM audit_log WHERE tenant_id = NEW.tenant_id AND txid = pg_current_xact_id()
     AND subject_type = 'record' AND subject_id = NEW.id LIMIT 1;
  IF NOT FOUND THEN RAISE EXCEPTION 'audit_missing: record %', NEW.id USING ERRCODE = '23514'; END IF;
  RETURN NULL;
END $f$;
RESET ROLE;
