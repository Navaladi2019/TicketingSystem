-- RFC-001 §4.4 role model, as written. Bootstrap superuser creates roles.
\set ON_ERROR_STOP 0
REVOKE CONNECT ON DATABASE nava FROM PUBLIC;
CREATE ROLE nava_migrator LOGIN PASSWORD 'm';
CREATE ROLE nava_app      LOGIN PASSWORD 'a';
CREATE ROLE nava_worker   LOGIN PASSWORD 'w';
CREATE ROLE nava_relay    LOGIN PASSWORD 'r';
CREATE ROLE nava_onboard  LOGIN PASSWORD 'o';
CREATE ROLE nava_groupsync LOGIN PASSWORD 'g';
GRANT CONNECT ON DATABASE nava TO nava_migrator,nava_app,nava_worker,nava_relay,nava_onboard,nava_groupsync;
CREATE SCHEMA app AUTHORIZATION nava_migrator;
GRANT USAGE ON SCHEMA app TO nava_app,nava_worker,nava_relay,nava_onboard,nava_groupsync;

SET ROLE nava_migrator;
CREATE TABLE app.tenants (id uuid PRIMARY KEY, slug text UNIQUE NOT NULL, onboarding_request_id uuid UNIQUE);
CREATE TABLE app.tenant_identity_binding (issuer text, org_id uuid, tenant_id uuid NOT NULL REFERENCES app.tenants, PRIMARY KEY (issuer, org_id));
CREATE TABLE app.records (tenant_id uuid NOT NULL, id uuid NOT NULL, data jsonb, PRIMARY KEY (tenant_id,id));
-- outbox: the RFC names only these columns
CREATE TABLE app.outbox (id uuid PRIMARY KEY, tenant_id uuid NOT NULL, kind text NOT NULL, ref text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), dispatched_at timestamptz, attempts int NOT NULL DEFAULT 0, last_error text);

-- I1 tenant policy + FORCE on every table with tenant_id (CI-1 rule)
DO $do$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['records','outbox','tenant_identity_binding'] LOOP
  EXECUTE format('ALTER TABLE app.%I ENABLE ROW LEVEL SECURITY', t);
  EXECUTE format('ALTER TABLE app.%I FORCE ROW LEVEL SECURITY', t);
  EXECUTE format('CREATE POLICY tenant_isolation ON app.%I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)', t);
 END LOOP; END $do$;
CREATE POLICY outbox_relay ON app.outbox TO nava_relay USING (true);

GRANT SELECT,INSERT,UPDATE,DELETE ON app.records TO nava_app, nava_worker;
GRANT SELECT ON app.tenant_identity_binding TO nava_app, nava_groupsync;
GRANT INSERT ON app.outbox TO nava_app, nava_worker;
GRANT SELECT (id, tenant_id, kind, ref, created_at) ON app.outbox TO nava_relay;
GRANT UPDATE (dispatched_at, attempts, last_error) ON app.outbox TO nava_relay;
GRANT INSERT ON app.tenants, app.tenant_identity_binding TO nava_onboard;
RESET ROLE;
-- seed (superuser bypasses RLS)
INSERT INTO app.tenants VALUES ('00000000-0000-0000-0000-00000000000a','a',null),('00000000-0000-0000-0000-00000000000b','b',null);
INSERT INTO app.tenant_identity_binding VALUES ('https://kc/realms/nava','11111111-0000-0000-0000-000000000001','00000000-0000-0000-0000-00000000000a');
INSERT INTO app.records VALUES ('00000000-0000-0000-0000-00000000000a',gen_random_uuid(),'{}'),('00000000-0000-0000-0000-00000000000b',gen_random_uuid(),'{}');
INSERT INTO app.outbox(id,tenant_id,kind,ref) VALUES (gen_random_uuid(),'00000000-0000-0000-0000-00000000000a','wf.start','r1'),(gen_random_uuid(),'00000000-0000-0000-0000-00000000000b','wf.start','r2');
