-- T18a (TICK-194): the email domains each tenant owns. InviteUser creates a
-- Keycloak user only for an email in a domain the tenant owns
-- (keycloak.Registry.TenantOwnsEmailDomain), so tenant A cannot pre-create
-- anyone@tenant-b.test and lock B's federated user out of their first
-- brokered login (security finding 3 on PR #101).
--
-- One index. The primary key is on domain alone because a domain belongs to
-- one tenant realm-wide, as UNIQUE (issuer, org_id) is global on
-- tenant_identity_binding; it INCLUDEs tenant_id, so the one read
-- (tenant_id = $1 AND domain = $2) is an index-only scan and the tenant
-- predicate is answered from the index. A (tenant_id, domain) key next to it
-- would only add write and storage cost (docs/data/identity.md).
--
-- nava_onboard writes the rows in T21's onboarding transaction, next to the
-- binding, and like the binding only for its own tenant while it is
-- provisioning (catalog.PinnedTenantPolicies). nava_app reads them under the
-- tenant predicate: another tenant's domain is no row, so it reads as "not
-- owned", never as a hint of who owns it. Grants: db/grants.yaml.
--
-- Empty new table, so plain CREATE INDEX holds no lock worth planning for.

-- +goose Up

-- domain is the part after the "@", lower case, ASCII (an IDN in its
-- punycode form), at most 253 bytes, at least two labels. The CHECK keeps a
-- mixed-case or malformed row from ever being written and then never matched.
CREATE TABLE tenant_email_domain (
    tenant_id uuid NOT NULL,
    domain    text NOT NULL,
    CONSTRAINT tenant_email_domain_pkey PRIMARY KEY (domain) INCLUDE (tenant_id),
    CONSTRAINT tenant_email_domain_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT tenant_email_domain_domain_check CHECK (
        octet_length(domain) <= 253
        AND domain ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$')
);
ALTER TABLE tenant_email_domain ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_email_domain FORCE ROW LEVEL SECURITY;

CREATE POLICY email_domain_select ON tenant_email_domain FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- tenants_onboard_select shows nava_onboard only the tenant of its
-- transaction, so the subquery adds "and that tenant is provisioning".
CREATE POLICY email_domain_onboard ON tenant_email_domain FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));
