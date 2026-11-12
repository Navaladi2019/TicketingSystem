---
title: "I1: Tenant wall in the database"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, internal/tenancy/isolation/ci2_isolation_test.go, internal/db/dbtest/catalog_test.go, internal/db/dbtest/platform_tables_test.go]
adrs: [ADR-001]
invariants: [I1]
tests: [TestCI2EveryRoleEveryTenantTable, TestT11TenantTablesFailClosed, TestT09PolicyAllowList]
packages: [internal/tenancy, internal/db]
status: delivered
---

**I1** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

Every table that holds tenant data carries a `tenant_id`, has row-level security enabled **and forced**, and has a policy that compares `tenant_id` with the tenant the session declared. If a session declares no tenant, a query returns no rows or an error, never another tenant's rows. Unique keys and foreign keys include `tenant_id`. A policy that does not test the tenant must be on a short, named allow-list.

## Where it is enforced

The database itself: PostgreSQL row-level security on every tenant table (`db/migrations`), plus the per-request tenant setting made by `internal/tenancy` and `internal/db`.

## Proof

ADR-001 Annex B CI-1 (catalogue checks) and CI-2 (no tenant set gives zero rows or an error, for every runtime role and every tenant table).

Tests that prove it, by Go test name: `TestCI2EveryRoleEveryTenantTable`, `TestT11TenantTablesFailClosed`, `TestT09PolicyAllowList`.

## Read next

Enforcement detail: [Row-level security](../database/rls.md). All nine invariants: [the invariants page](invariants.md).

## Note

The shared database and shared tables are **decision G7**, not an invariant: topology is a performance and deployment choice. I1 and I3 are what keep tenants apart, whatever the topology.
