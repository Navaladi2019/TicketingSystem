-- TICK-751 (S22 P7-11, folded into TICK-916; ADR-006 D12, I8): a tenant's
-- per-type filterable-field limit, raised by an operator from the default 5
-- up to 10 with no per-tenant DDL. One optional row per tenant: no row is the
-- default (fieldindex.DefaultPerTypeLimit); publish reads it in its own
-- transaction (tenantlimits.PerType). The tenant cap of 50 is unchanged.
--
-- Who may do what (db/grants.yaml): nava_app SELECT (publish), INSERT and
-- UPDATE of the three value columns. The only writer is the operator route
-- PUT /api/v1/operator/tenants/{tenant}/limits on nava-onboard, which
-- authenticates an operator token (D-P12) and writes through its nava_app
-- pool in the target tenant's transaction; no tenant route writes it. Nobody
-- deletes a row or moves its tenant_id.
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
CREATE POLICY tenant_limits_insert_app ON tenant_limits FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_limits_update_app ON tenant_limits FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

CREATE CONSTRAINT TRIGGER tenant_limits_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF filterable_fields_per_type, updated_at, updated_by ON tenant_limits
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('tenant_limits', 'tenant_id');

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP TABLE tenant_limits;
