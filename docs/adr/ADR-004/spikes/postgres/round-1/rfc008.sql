\set ON_ERROR_STOP 0
CREATE ROLE w_app; CREATE ROLE w_onboard;
CREATE TABLE cfg (tenant_id uuid, id uuid, v int);
CREATE TABLE audit_log (tenant_id uuid, txid xid8 NOT NULL DEFAULT pg_current_xact_id(), subject_type text, subject_id uuid);
CREATE UNIQUE INDEX ON audit_log (tenant_id, txid, subject_type, subject_id);
ALTER TABLE cfg ENABLE ROW LEVEL SECURITY; ALTER TABLE cfg FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY; ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY p ON cfg TO w_app, w_onboard USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY s ON audit_log FOR SELECT TO w_app USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY i ON audit_log FOR INSERT TO w_app, w_onboard WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
GRANT SELECT, INSERT, UPDATE ON cfg TO w_app, w_onboard; GRANT SELECT, INSERT ON audit_log TO w_app; GRANT INSERT ON audit_log TO w_onboard;
CREATE FUNCTION chk() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM audit_log a WHERE a.tenant_id=NEW.tenant_id AND a.txid=pg_current_xact_id() AND a.subject_type=TG_ARGV[0] AND a.subject_id=NEW.id)
  THEN RAISE EXCEPTION 'audit missing for % %', TG_ARGV[0], NEW.id; END IF; RETURN NULL; END $$;
CREATE CONSTRAINT TRIGGER cfg_audit AFTER INSERT OR UPDATE ON cfg DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION chk('cfg');
\echo == T1 savepoint: top-level xid same?
BEGIN; SELECT pg_current_xact_id() AS top \gset
SAVEPOINT a; SELECT pg_current_xact_id() = :'top'::xid8 AS same_in_savepoint; ROLLBACK;
\echo == T2 w_app audited write commits
SET ROLE w_app; BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
INSERT INTO cfg VALUES ('11111111-1111-1111-1111-111111111111','22222222-2222-2222-2222-222222222222',1);
INSERT INTO audit_log(tenant_id,subject_type,subject_id) VALUES ('11111111-1111-1111-1111-111111111111','cfg','22222222-2222-2222-2222-222222222222'); COMMIT; RESET ROLE;
\echo == T3 w_app audit inside rolled-back savepoint must fail at commit
SET ROLE w_app; BEGIN; SELECT set_config('app.tenant_id','11111111-1111-1111-1111-111111111111',true);
UPDATE cfg SET v=2; SAVEPOINT s1; INSERT INTO audit_log(tenant_id,subject_type,subject_id) VALUES ('11111111-1111-1111-1111-111111111111','cfg','22222222-2222-2222-2222-222222222222'); ROLLBACK TO s1; COMMIT; RESET ROLE;
\echo == T4 w_onboard: INSERT-only on audit_log (RFC D8)
SET ROLE w_onboard; BEGIN; SELECT set_config('app.tenant_id','33333333-3333-3333-3333-333333333333',true);
INSERT INTO cfg VALUES ('33333333-3333-3333-3333-333333333333','44444444-4444-4444-4444-444444444444',1);
INSERT INTO audit_log(tenant_id,subject_type,subject_id) VALUES ('33333333-3333-3333-3333-333333333333','cfg','44444444-4444-4444-4444-444444444444'); COMMIT; RESET ROLE;
\echo == T5 owner under FORCE RLS reads source table
CREATE ROLE w_owner; ALTER TABLE cfg OWNER TO w_owner; SET ROLE w_owner; SELECT count(*) AS owner_sees FROM cfg; RESET ROLE;
\echo == T6 transient owner policy inside one tx
SET ROLE w_owner; BEGIN; CREATE POLICY backfill_tmp ON cfg FOR SELECT TO w_owner USING (true); SELECT count(*) AS owner_sees_tmp FROM cfg; DROP POLICY backfill_tmp ON cfg; COMMIT;
SELECT count(*) AS after_commit FROM cfg; RESET ROLE; SELECT count(*) AS leftover_policies FROM pg_policies WHERE policyname='backfill_tmp';
