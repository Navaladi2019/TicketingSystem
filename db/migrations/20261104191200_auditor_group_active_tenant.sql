-- P5-23 (TICK-568, ADR-004 with ADR-001-A6, PLAN-005 D-5-4): nava_onboard may
-- create the auditors group of an ACTIVE tenant that has none, and nothing
-- else. Tenants activated before S14 have no tenant_auditor_group row, and the
-- provisioning-only policies (groups_insert_onboard, groups_select_onboard,
-- tenant_auditor_group_insert) refuse the upgrade reconcile (TICK-567), which
-- records the group through onboard.Store.RecordAuditorGroup. Those policies
-- are unchanged; the ones below are permissive and OR with them.
--
-- "The auditors group" is the groups row whose path is /t/<tenant>/auditors
-- (identity.TenantGroupPrefix + identity.AuditorGroupName). The platform id
-- onboard.AuditorGroupID is a UUIDv5 (SHA-1), which PostgreSQL 16 cannot
-- compute without an extension, so the policies pin the path, not the id.
-- The path column is what makes the row the auditors group in Keycloak, and
-- the pointer insert below must name a live row with that path, so the
-- pointer can never be aimed at the admins group or any other group.
--
--   groups INSERT: own tenant, active, path = auditors, and only while the
--     tenant has no tenant_auditor_group row.
--   groups SELECT: own tenant, active, path = auditors. RecordAuditorGroup
--     reads the row back by keycloak_group_id; a reconcile sees whether a
--     group sync created it first.
--   tenant_auditor_group INSERT: own tenant, active, group_id names a live
--     group of the same tenant that nava_onboard can see. The subquery runs
--     under nava_onboard's groups policies, and for an active tenant the only
--     one is groups_select_onboard_auditors, so only the auditors row is
--     visible: the path test lives in that policy's USING, which needs no
--     column grant (nava_onboard has no SELECT on groups.path, TICK-536).
--     A future nava_onboard groups SELECT policy for active tenants would
--     widen this; TestP523OnboardActiveTenantRefusals guards it. "No row yet" is the primary key
--     (tenant_id): a second row is 23505, or nothing under ON CONFLICT DO
--     NOTHING. A NOT EXISTS on its own table would be infinite recursion
--     in policy (42P17), since PostgreSQL expands a table's policies once
--     per query branch.
--   tenant_auditor_group SELECT: own tenant (plain tenant predicate). The
--     groups INSERT NOT EXISTS probe runs as nava_onboard and needs it; the reconcile
--     uses it to detect the no-op case.
--
-- No UPDATE, no DELETE, no membership table. Once the pointer exists, the
-- groups INSERT fails its WITH CHECK (42501) even under ON CONFLICT DO
-- NOTHING, because PostgreSQL checks the policy before the conflict: the
-- reconcile reads tenant_auditor_group first and skips a tenant that has it.
-- RecordAuditorGroup writes both rows in one transaction, so no committed
-- state has the group and not the pointer.
--
-- Outer columns are qualified with the policy's table in every subquery
-- (Security N1). No index: the subqueries hit tenants_pkey,
-- tenant_auditor_group_pkey and groups_pkey by equality. Policies only; no
-- table lock beyond the catalog update. Grants: db/grants.yaml (SELECT
-- tenant_id on tenant_auditor_group; groups grants unchanged).

-- +goose Up

CREATE POLICY groups_insert_onboard_auditors ON groups FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND path = '/t/' || groups.tenant_id::text || '/auditors'
                AND groups.tenant_id IN (SELECT t.id FROM tenants t WHERE t.state = 'active')
                AND NOT EXISTS (SELECT 1 FROM tenant_auditor_group p WHERE p.tenant_id = groups.tenant_id));

CREATE POLICY groups_select_onboard_auditors ON groups FOR SELECT TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND path = '/t/' || groups.tenant_id::text || '/auditors'
           AND groups.tenant_id IN (SELECT t.id FROM tenants t WHERE t.state = 'active'));

CREATE POLICY tenant_auditor_group_select_onboard ON tenant_auditor_group FOR SELECT TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

CREATE POLICY tenant_auditor_group_insert_active ON tenant_auditor_group FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_auditor_group.tenant_id IN (SELECT t.id FROM tenants t WHERE t.state = 'active')
                AND EXISTS (SELECT 1 FROM groups g
                             WHERE g.tenant_id = tenant_auditor_group.tenant_id
                               AND g.id = tenant_auditor_group.group_id
                               AND g.deleted_at IS NULL));

-- +goose Down

DROP POLICY tenant_auditor_group_insert_active ON tenant_auditor_group;
DROP POLICY tenant_auditor_group_select_onboard ON tenant_auditor_group;
DROP POLICY groups_select_onboard_auditors ON groups;
DROP POLICY groups_insert_onboard_auditors ON groups;
