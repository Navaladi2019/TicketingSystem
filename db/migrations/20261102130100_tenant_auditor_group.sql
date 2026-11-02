-- P5-18 (TICK-434, PLAN-005 D-5-4): the platform id of each tenant's `auditors`
-- group. authz.RequireAuditPermission compares a principal's group ids with
-- this id and never reads groups.path. Mirrors tenant_admin_group (D-3-3): one
-- row per tenant, written once by onboarding in its tenant transaction, no
-- runtime role may UPDATE or DELETE it.
--
-- (tenant_id, group_id) references groups (tenant_id, id): a row naming tenant
-- B's group is refused. ON DELETE CASCADE as for tenant_admin_group: group sync
-- must still commit when Keycloak loses the group; the tenant then has no
-- auditors and the check denies (fail closed).
--
-- Existing tenants have no row and are back-filled by a one-off script run as
-- the migrator (docs/acceptance/guide/tools/seed.sh), not by this migration.
-- Empty new table; the primary key serves the only lookup.
-- Grants: db/grants.yaml.

-- +goose Up

CREATE TABLE tenant_auditor_group (
    tenant_id uuid NOT NULL,
    group_id  uuid NOT NULL,
    CONSTRAINT tenant_auditor_group_pkey PRIMARY KEY (tenant_id),
    CONSTRAINT tenant_auditor_group_group_fkey FOREIGN KEY (tenant_id, group_id)
        REFERENCES groups (tenant_id, id) ON DELETE CASCADE
);
ALTER TABLE tenant_auditor_group ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_auditor_group FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_auditor_group_select ON tenant_auditor_group FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_auditor_group_insert ON tenant_auditor_group FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));
