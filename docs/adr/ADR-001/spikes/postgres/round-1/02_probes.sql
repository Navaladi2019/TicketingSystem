\set ON_ERROR_STOP 0
\echo '=== P1 relay: claim pending rows (needs dispatched_at in WHERE) ==='
SET ROLE nava_relay;
SELECT id, tenant_id FROM app.outbox WHERE dispatched_at IS NULL ORDER BY created_at LIMIT 10 FOR UPDATE SKIP LOCKED;
\echo '=== P2 relay: mark dispatched, idempotent guard on dispatched_at IS NULL ==='
UPDATE app.outbox SET dispatched_at = now() WHERE id = (SELECT id FROM app.outbox LIMIT 1) AND dispatched_at IS NULL;
\echo '=== P2b relay: attempts = attempts + 1 (reads attempts) ==='
UPDATE app.outbox SET attempts = attempts + 1 WHERE id = (SELECT id FROM app.outbox LIMIT 1);
\echo '=== P3 relay: plain read of granted columns (sees all tenants, by design) ==='
SELECT count(*) FROM app.outbox;
RESET ROLE;

\echo '=== P4 app: identity lookup BEFORE tenant is known (binding table under CI-1 FORCE RLS) ==='
SET ROLE nava_app;
SELECT tenant_id FROM app.tenant_identity_binding WHERE issuer='https://kc/realms/nava' AND org_id='11111111-0000-0000-0000-000000000001';
RESET ROLE;

\echo '=== P5 worker: idempotent outbox insert (INSERT-only grant) ==='
SET ROLE nava_worker;
BEGIN; SELECT set_config('app.tenant_id','00000000-0000-0000-0000-00000000000a',true);
INSERT INTO app.outbox(id,tenant_id,kind,ref) VALUES ('22222222-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a','wf.start','x') ON CONFLICT DO NOTHING;
INSERT INTO app.outbox(id,tenant_id,kind,ref) VALUES ('22222222-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a','wf.start','x') ON CONFLICT DO NOTHING;
COMMIT;
\echo '=== P5b worker: INSERT ... RETURNING id with INSERT-only grant ==='
BEGIN; SELECT set_config('app.tenant_id','00000000-0000-0000-0000-00000000000a',true);
INSERT INTO app.outbox(id,tenant_id,kind,ref) VALUES (gen_random_uuid(),'00000000-0000-0000-0000-00000000000a','wf.start','x') RETURNING id;
ROLLBACK;
\echo '=== P6 worker: insert outbox row for ANOTHER tenant (WITH CHECK) ==='
BEGIN; SELECT set_config('app.tenant_id','00000000-0000-0000-0000-00000000000a',true);
INSERT INTO app.outbox(id,tenant_id,kind,ref) VALUES (gen_random_uuid(),'00000000-0000-0000-0000-00000000000b','wf.start','x');
ROLLBACK;
RESET ROLE;

\echo '=== P7 A4: setting value after SET LOCAL txn ends, same session ==='
SET ROLE nava_app;
SELECT quote_nullable(current_setting('app.tenant_id', true)) AS before_any;
BEGIN; SELECT set_config('app.tenant_id','00000000-0000-0000-0000-00000000000a',true); SELECT count(*) AS in_txn FROM app.records; COMMIT;
SELECT quote_nullable(current_setting('app.tenant_id', true)) AS after_local, (SELECT count(*) FROM app.records) AS rows_visible;
\echo '=== P8 app: garbage tenant string -> error (fail closed) ==='
BEGIN; SELECT set_config('app.tenant_id','not-a-uuid',true); SELECT count(*) FROM app.records; ROLLBACK;
\echo '=== P9 app: runtime role can re-point its own tenant mid-txn (GUC is user-settable) ==='
BEGIN; SELECT set_config('app.tenant_id','00000000-0000-0000-0000-00000000000a',true);
SELECT set_config('app.tenant_id','00000000-0000-0000-0000-00000000000b',true); SELECT tenant_id FROM app.records; ROLLBACK;
RESET ROLE;

\echo '=== P10 PUBLIC grant: a stray GRANT TO PUBLIC reaches nava_relay; CI-3 membership query sees nothing ==='
SET ROLE nava_migrator; GRANT SELECT ON app.records TO PUBLIC; RESET ROLE;
WITH RECURSIVE m AS (SELECT roleid, member FROM pg_auth_members WHERE member = 'nava_relay'::regrole
  UNION SELECT a.roleid, a.member FROM pg_auth_members a JOIN m ON a.member = m.roleid)
SELECT count(*) AS ci3_membership_rows_for_relay FROM m;
SET ROLE nava_relay; BEGIN; SELECT set_config('app.tenant_id','00000000-0000-0000-0000-00000000000b',true); SELECT count(*) AS relay_reads_records FROM app.records; ROLLBACK; RESET ROLE;
SET ROLE nava_migrator; REVOKE SELECT ON app.records FROM PUBLIC; RESET ROLE;
\echo '=== P11 default EXECUTE on functions to PUBLIC ==='
SET ROLE nava_migrator; CREATE FUNCTION app.f() RETURNS int LANGUAGE sql AS 'select 1'; RESET ROLE;
SELECT has_function_privilege('nava_relay','app.f()','EXECUTE') AS relay_can_exec_new_fn;
\echo '=== P12 migrator (owner) under FORCE, no tenant: rows visible? can it turn FORCE off? ==='
SET ROLE nava_migrator; SELECT count(*) AS owner_rows FROM app.records;
BEGIN; ALTER TABLE app.records NO FORCE ROW LEVEL SECURITY; SELECT count(*) AS owner_rows_after_noforce FROM app.records; ROLLBACK; RESET ROLE;
\echo '=== P13 app session tries SET ROLE nava_migrator ==='
SET ROLE nava_app; SET ROLE nava_migrator; RESET ROLE;
