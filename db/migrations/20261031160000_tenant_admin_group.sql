-- P3-04 (TICK-342, PLAN-003 D-3-3): the platform id of each tenant's `admins`
-- group. authz.RequireTenantAdmin(Tx) compares a principal's group ids with
-- this id and never reads groups.path, which group sync rewrites on every
-- rename (ADR-002 section 1.4). One row per tenant, written once by onboarding
-- in its tenant transaction; no runtime role may UPDATE or DELETE it, so the
-- admin group cannot be re-pointed at another group by a rename or a bug.
--
-- (tenant_id, group_id) references groups (tenant_id, id): the group is a
-- group of the same tenant, so a row naming tenant B's group is refused.
--
-- Onboarding also inserts the groups row itself (same transaction), with the
-- platform id it records here and the Keycloak group id it just created. Group
-- sync then finds the row by (tenant_id, keycloak_group_id) and keeps its id,
-- only keeping path current. groups_insert_onboard lets nava_onboard insert
-- only the provisioning tenant's own groups, as the other nava_onboard writes.
--
-- Empty new table; no lock worth planning for. No second index: the primary
-- key serves the only lookup (WHERE tenant_id = $1), and the foreign key's
-- referenced side is groups' primary key.
-- Grants: db/grants.yaml.

-- +goose Up

CREATE TABLE tenant_admin_group (
    tenant_id uuid NOT NULL,
    group_id  uuid NOT NULL,
    CONSTRAINT tenant_admin_group_pkey PRIMARY KEY (tenant_id),
    CONSTRAINT tenant_admin_group_group_fkey FOREIGN KEY (tenant_id, group_id)
        REFERENCES groups (tenant_id, id)
);
ALTER TABLE tenant_admin_group ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_admin_group FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_admin_group_select ON tenant_admin_group FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_admin_group_insert ON tenant_admin_group FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));

CREATE POLICY groups_insert_onboard ON groups FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));
