-- RFC-004 spike: does a state predicate on nava_onboard's SELECT policy break
-- activation? PostgreSQL 16.10 (docs/pins.md). Run as superuser on a throwaway
-- database. Mirrors db/migrations/00002_platform_tables.sql for `tenants`.
\set ON_ERROR_STOP off
\set VERBOSITY terse
CREATE ROLE nava_onboard LOGIN NOSUPERUSER NOBYPASSRLS;
CREATE TABLE tenants (
    id uuid NOT NULL,
    state text NOT NULL,
    onboarding_request_hash bytea,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tenants_pkey PRIMARY KEY (id),
    CONSTRAINT tenants_state_check CHECK (state IN ('provisioning', 'active'))
);
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT (id, state, onboarding_request_hash), UPDATE (state) ON tenants TO nava_onboard;

CREATE POLICY tenants_onboard_insert ON tenants FOR INSERT TO nava_onboard
    WITH CHECK (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND state = 'provisioning');
CREATE POLICY tenants_activate ON tenants FOR UPDATE TO nava_onboard
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND state = 'provisioning')
    WITH CHECK (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND state = 'active');

\echo
\echo '=== Case 1: SELECT policy as shipped (own row, no state predicate) ==='
CREATE POLICY tenants_onboard_select ON tenants FOR SELECT TO nava_onboard
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
SET ROLE nava_onboard;
BEGIN;
SELECT set_config('app.tenant_id', '0190a000-0000-7000-8000-000000000001', true);
INSERT INTO tenants (id, state, onboarding_request_hash) VALUES ('0190a000-0000-7000-8000-000000000001', 'provisioning', '\x01');
\echo '-- activation with WHERE id = own tenant (the T09 shape)'
UPDATE tenants SET state = 'active' WHERE id = '0190a000-0000-7000-8000-000000000001';
\echo '-- own row after activation: visible (this is the drift TICK-170 reports)'
SELECT id, state, encode(onboarding_request_hash, 'hex') AS hash FROM tenants;
COMMIT;
RESET ROLE;
DROP POLICY tenants_onboard_select ON tenants;
DELETE FROM tenants;

\echo
\echo '=== Case 2: SELECT policy restricted to provisioning (A.3 read literally) ==='
CREATE POLICY tenants_onboard_select ON tenants FOR SELECT TO nava_onboard
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND state = 'provisioning');
SET ROLE nava_onboard;
BEGIN;
SELECT set_config('app.tenant_id', '0190a000-0000-7000-8000-000000000002', true);
INSERT INTO tenants (id, state, onboarding_request_hash) VALUES ('0190a000-0000-7000-8000-000000000002', 'provisioning', '\x02');
\echo '-- 2a: activation with WHERE id = own tenant: expected ERROR 42501 (new row fails the SELECT policy, Table 292 footnote a)'
UPDATE tenants SET state = 'active' WHERE id = '0190a000-0000-7000-8000-000000000002';
\echo '-- SQLSTATE of the last error:'
\echo :LAST_ERROR_SQLSTATE
ROLLBACK;
BEGIN;
SELECT set_config('app.tenant_id', '0190a000-0000-7000-8000-000000000002', true);
INSERT INTO tenants (id, state, onboarding_request_hash) VALUES ('0190a000-0000-7000-8000-000000000002', 'provisioning', '\x02');
\echo '-- 2b: activation with RETURNING: expected ERROR 42501'
UPDATE tenants SET state = 'active' RETURNING id;
\echo :LAST_ERROR_SQLSTATE
ROLLBACK;
BEGIN;
SELECT set_config('app.tenant_id', '0190a000-0000-7000-8000-000000000002', true);
INSERT INTO tenants (id, state, onboarding_request_hash) VALUES ('0190a000-0000-7000-8000-000000000002', 'provisioning', '\x02');
\echo '-- 2c: activation with no WHERE and no RETURNING: no read access needed, so the SELECT policy is not applied; expected UPDATE 1'
UPDATE tenants SET state = 'active';
\echo '-- own row after activation under 2c: expected 0 rows (onboard can no longer read its own tenant)'
SELECT id, state FROM tenants;
COMMIT;
RESET ROLE;
DROP POLICY tenants_onboard_select ON tenants;
DELETE FROM tenants;

\echo
\echo '=== Case 3: two permissive SELECT policies (provisioning, active) ==='
CREATE POLICY tenants_onboard_select ON tenants FOR SELECT TO nava_onboard
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND state = 'provisioning');
CREATE POLICY tenants_onboard_select_active ON tenants FOR SELECT TO nava_onboard
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND state = 'active');
SET ROLE nava_onboard;
BEGIN;
SELECT set_config('app.tenant_id', '0190a000-0000-7000-8000-000000000003', true);
INSERT INTO tenants (id, state) VALUES ('0190a000-0000-7000-8000-000000000003', 'provisioning');
\echo '-- activation with WHERE: expected UPDATE 1 (the policies OR together, which equals "own row")'
UPDATE tenants SET state = 'active' WHERE id = '0190a000-0000-7000-8000-000000000003';
SELECT id, state FROM tenants;
COMMIT;
RESET ROLE;
SELECT version();
