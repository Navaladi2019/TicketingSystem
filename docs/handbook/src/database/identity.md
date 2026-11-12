---
title: "Identity and directory"
section: database
status: delivered
tables: [principals, groups, group_members, org_units, tenant_identity_binding, tenant_email_domain, pending_invite, tenant_admin_group, tenant_auditor_group]
adrs: [ADR-001, ADR-004-E5]
sources: [docs/data/identity.md, docs/data/platform-tables.md, docs/data/groups-tombstone.md, docs/data/principal-tombstone.md]
---

These tables are a copy of the identity provider (Keycloak), plus the
bindings that turn a token into a tenant. Keycloak is the source of truth for
users and groups. Group sync (`nava_groupsync`) writes `principals`, `groups`,
`group_members` and `org_units` from the Keycloak admin API, one transaction
per tenant. The API reads them under RLS and takes Keycloak user and group ids
only from these rows.

- **Tenant resolution.** `tenant_identity_binding` maps a token's
  `(issuer, org_id)` to a tenant. It is read before a tenant is known, so its
  read policy, `binding_lookup`, is on the allow-list and has no tenant test
  (ADR-001 A.1).
- **Tombstones, not deletes.** A group or principal removed in Keycloak keeps
  its row with `deleted_at` set (ADR-004-E5). Audit entries and records that
  name it keep a valid foreign key. `group_members` rows are inserted and
  deleted.
- **Onboarding.** `tenant_email_domain`, `pending_invite`, `tenant_admin_group`
  and `tenant_auditor_group` are written by onboarding, under policies pinned
  to the tenant while it is `provisioning`. The two group tables point at the
  tenant's admins and auditors groups. Group sync may delete a pointer only
  once the group it names is tombstoned.

## Diagram

<!-- gen:er area=identity -->

## Tables

<!-- gen:tables area=identity -->
