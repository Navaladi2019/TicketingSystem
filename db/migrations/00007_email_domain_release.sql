-- TICK-280 (follow-up from TICK-276 item 1, AR-036): nava_onboard may
-- release the email domains of its own tenant while that tenant is still
-- provisioning, so a stuck onboarding stops holding domains against their
-- real owner. Whether the abandon operation calls it is AR-036's design.
--
-- The release is DELETE FROM tenant_email_domain WHERE tenant_id = $1 in the
-- tenant transaction. email_domain_release carries email_domain_onboard's
-- guard on USING: the tenant of the transaction, and only while it is
-- provisioning (pinned in catalog.PinnedTenantPolicies). An active tenant's
-- domains, and another tenant's, are no rows.
--
-- The WHERE reads tenant_id, so nava_onboard has SELECT on that column alone
-- (db/grants.yaml) and email_domain_onboard_select, the canonical tenant
-- predicate. A DELETE that reads a column runs under both policies, so the
-- provisioning guard still holds; the SELECT shows only its own tenant's
-- tenant_id, never a domain, so onboarding learns nothing it did not know.
-- The CI-2 isolation suite needs the key readable to prove another tenant's
-- rows are out of reach.
--
-- No index. Both policies read tenant_id, which tenant_email_domain_pkey only
-- INCLUDEs, so the delete scans the table: 2.4 ms and 210 buffers at 25,003
-- rows (docs/data/identity.md), on a path that runs once per abandoned
-- tenant. A (tenant_id) index brings it to 0.13 ms for 504 kB and a second
-- index write on every claim.
--
-- CREATE POLICY takes an ACCESS EXCLUSIVE lock on the table for the length
-- of this transaction only; no data is rewritten.

-- +goose Up

CREATE POLICY email_domain_onboard_select ON tenant_email_domain FOR SELECT TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- tenants_onboard_select shows nava_onboard only the tenant of its
-- transaction, so the subquery adds "and that tenant is provisioning".
CREATE POLICY email_domain_release ON tenant_email_domain FOR DELETE TO nava_onboard
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));
