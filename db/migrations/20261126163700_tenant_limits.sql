-- TICK-751 (S22 P7-11, folded into TICK-916; ADR-006 D12, I8): a tenant's
-- per-type filterable-field limit, raised by an operator from the default 5
-- up to 10 with no per-tenant DDL. One optional row per tenant: no row is the
-- default (fieldindex.DefaultPerTypeLimit); publish reads it in its own
-- transaction (tenantlimits.PerType). The tenant cap of 50 is unchanged.
--
-- Who may do what (db/grants.yaml): nava_app SELECT only (publish). The one
-- writer is nava_onboard (AR-TICK-916-01, Security review of PR #921): the
-- operator route PUT /api/v1/operator/tenants/{tenant}/limits on
-- nava-onboard authenticates an operator token (D-P12) and writes through its
-- nava_onboard pool in the target tenant's transaction, SELECT, INSERT and
-- UPDATE of the three value columns. So no tenant-scoped nava_app
-- transaction, whatever route, bug or injection runs it, can raise its own
-- tenant's limit. Nobody deletes a row or moves its tenant_id.
--
-- nava_onboard's generated audit_log policies admit a provisioning tenant
-- only (ADR-004 R12), and limits are raised on active tenants. Two pinned
-- policies (catalog.PinnedTenantPolicies) let it write and read back, for the
-- commit check, the tenant_limits entries of its transaction's tenant, in any
-- state, and nothing else.
--
-- Audit (db/audited-tables.yaml): every write needs a config entry under the
-- tenant_limits subject (subject_id the tenant id) in its transaction;
-- tenant_limits_audit_commit_check enforces it at commit.
--
-- A new, empty table: only the foreign key to tenants takes a lock on an
-- existing table (SHARE ROW EXCLUSIVE, milliseconds).

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE TABLE tenant_limits (
    tenant_id                  uuid        NOT NULL,
    filterable_fields_per_type smallint    NOT NULL,
    updated_at                 timestamptz NOT NULL DEFAULT now(),
    updated_by                 text        NOT NULL,
    CONSTRAINT tenant_limits_pkey PRIMARY KEY (tenant_id),
    CONSTRAINT tenant_limits_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT tenant_limits_filterable_fields_per_type_check CHECK (filterable_fields_per_type BETWEEN 5 AND 10),
    CONSTRAINT tenant_limits_updated_by_check CHECK (length(updated_by) BETWEEN 1 AND 255)
);
ALTER TABLE tenant_limits ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_limits FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_limits_select_app ON tenant_limits FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_limits_select_onboard ON tenant_limits FOR SELECT TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_limits_insert_onboard ON tenant_limits FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_limits_update_onboard ON tenant_limits FOR UPDATE TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY audit_log_limits_onboard_insert ON audit_log FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND subject_type = 'tenant_limits');
CREATE POLICY audit_log_limits_onboard_select ON audit_log FOR SELECT TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND subject_type = 'tenant_limits');

CREATE CONSTRAINT TRIGGER tenant_limits_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF filterable_fields_per_type, updated_at, updated_by ON tenant_limits
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('tenant_limits', 'tenant_id');

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP POLICY audit_log_limits_onboard_select ON audit_log;
DROP POLICY audit_log_limits_onboard_insert ON audit_log;
DROP TABLE tenant_limits;
