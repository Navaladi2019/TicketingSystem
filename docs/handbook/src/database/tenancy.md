---
title: "Tenancy"
section: database
status: delivered
tables: [tenants, tenant_limits]
stories: [S22]
operations: [setTenantLimits]
adrs: [ADR-001, ADR-001-E3]
sources: [docs/data/platform-tables.md, docs/adr/ADR-001-platform-blueprint.md, internal/server/operator_limits.go, internal/tenantlimits, db/grants.yaml]
tests: [TestSetTenantLimitsRaisesOnceAndAudits, TestSetTenantLimitsRefusesNonOperators, TestPublishHonoursTheTenantsRaisedLimit]
---

`tenants` is the root of every other table. It is the one table whose tenant
key is `id`, not `tenant_id`, so its policies test `id` against
`app.tenant_id`. Every other table points at it, directly or through a
composite key. Every composite key starts with `tenant_id`, so a foreign key
cannot point at another tenant's row.

A tenant starts as `provisioning` and becomes `active`. Onboarding
(`nava_onboard`) inserts the row and moves it to `active`. Its pinned policies
(see [RLS](rls.md)) let it write the tenant only while the tenant is
`provisioning`, and only for the tenant in `app.tenant_id`.
`onboarding_request_hash` makes a repeated onboarding request idempotent.

## Per-tenant limits: `tenant_limits` (S22)

One optional row per tenant holds limits an operator has raised. Today it has
one limit, `filterable_fields_per_type`, between 5 and 10 (ADR-006 D12). No row
means the default, 5. The entity-type publish reads it through
`tenantlimits.PerType` when it counts a type's flagged fields.

- **Who writes it.** Only an operator, through `PUT
  /api/v1/operator/tenants/{tenant}/limits` (`setTenantLimits`) on
  nava-onboard. A tenant token gets `403`. An unknown tenant is `404`. A value
  out of range, or a request with no reason, is `400`; a body that is not
  strict JSON is `422 invalid_tenant_limits`. The write runs in the **target
  tenant's** transaction on the existing pool, with no new database role.
- **Audit.** Every change is one audit entry under the `tenant_limits` subject,
  naming the operator and the reason. Writing the value already in force
  writes nothing, so a replay leaves one entry.
- **Grants.** `nava_onboard` is the only role that writes it (AR-TICK-916-01,
  fixed in TICK-916). `nava_app` reads its own tenant's row only. No tenant
  role can write its own limit or another tenant's (S22 attack row 3, see
  [S22](../stories/s22.md)).

## Diagram

<!-- gen:er area=tenancy -->

## Tables

<!-- gen:tables area=tenancy -->
