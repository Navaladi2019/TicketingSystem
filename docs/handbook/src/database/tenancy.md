---
title: "Tenancy"
section: database
status: delivered
tables: [tenants]
adrs: [ADR-001, ADR-001-E3]
sources: [docs/data/platform-tables.md, docs/adr/ADR-001-platform-blueprint.md]
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

## Diagram

<!-- gen:er area=tenancy -->

## Tables

<!-- gen:tables area=tenancy -->
