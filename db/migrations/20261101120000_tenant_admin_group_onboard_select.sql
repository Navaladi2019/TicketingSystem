-- P3-04h (TICK-373, PLAN-003 D-3-3): "active implies the tenant has an admin"
-- is enforced by the store, not only by the order of calls in onboard.Service.
-- PGStore.Activate's UPDATE carries
--   AND EXISTS (SELECT 1 FROM tenant_admin_group WHERE tenant_id = $1)
-- so it changes no row of a tenant without an admins row. nava_onboard runs
-- that subquery under FORCE RLS, so it needs a SELECT policy and a grant.
--
-- The policy is as narrow as the other nava_onboard reads (groups_select_
-- onboard): the transaction's own tenant, and only while it is provisioning.
-- Activate's subquery runs on the statement's snapshot, in which the tenant is
-- still provisioning. The grant is SELECT (tenant_id) only: the subquery reads
-- no other column, and group_id stays hidden from onboarding.
--
-- A policy on an existing small table: catalog-only, no rewrite, no index. The
-- primary key on tenant_id serves the EXISTS. Grants: db/grants.yaml.

-- +goose Up

CREATE POLICY tenant_admin_group_select_onboard ON tenant_admin_group FOR SELECT TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));
