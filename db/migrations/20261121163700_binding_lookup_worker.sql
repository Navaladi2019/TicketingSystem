-- TICK-1070 (S31; PLAN-011-A2 D-11A2-2): the N-hat_e refresh job in nava-worker
-- lists tenants without a cross-tenant read on records or tenants. It reads
-- tenant_identity_binding.tenant_id through binding_lookup, the ADR-001 A.3
-- allow-listed SELECT policy (USING (true)) that tenant resolution and group
-- sync already use. db/grants.yaml grants nava_worker that one column, so the
-- issuer, org id and aliases stay out of its reach.
--
-- Every tenant the job must refresh has a binding: onboarding writes it
-- while the tenant is provisioning (onboard.PGStore.Bind), before it can be
-- activated and before a token can resolve to it; no role holds DELETE on
-- the table. A tenant may hold more than one binding, so the job reads
-- DISTINCT tenant_id.
--
-- The policy's roles are pinned in catalog.AllowList (PLAN-001 D-P10, CTO
-- sign-off). It is a separate migration so a test run that stops at an
-- earlier version can add it out of order, as it does for the counter purge
-- policies.
--
-- Locks: ALTER POLICY takes an ACCESS EXCLUSIVE lock on the binding table for
-- the catalog update only. Tenant resolution reads it on every request, so
-- lock_timeout fails the migration rather than queue those reads.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER POLICY binding_lookup ON tenant_identity_binding TO nava_app, nava_groupsync, nava_worker;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER POLICY binding_lookup ON tenant_identity_binding TO nava_app, nava_groupsync;
