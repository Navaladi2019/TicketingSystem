---
title: "I3: Least-privilege database roles"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, db/grants.yaml, internal/db/dbtest/d28_test.go, internal/db/dbtest/grants_fault_test.go, internal/db/dbtest/roles_test.go, internal/db/dbtest/catalog_test.go]
adrs: [ADR-001]
invariants: [I3]
tests: [TestD28NoSuperuserOrBypassRLS, TestManifestDetectsAndReportsGrantDrift, TestDBRolesNewPlatformDatabaseHasNoPublicGrants, TestT10GateEmptySecurityDefinerExceptions]
packages: [internal/db]
status: delivered
---

**I3** is one of the nine invariants in ADR-001 section 1. The wording below follows that table, paraphrased for readability; the ADR is the decision.

## What it says

No account the platform runs as is a superuser, can skip row-level security, owns an object, or is a member of any other role. Nothing is granted to `PUBLIC`, including through default privileges. Every permission comes from one manifest, and CI compares the live database with it. No `SECURITY DEFINER` function or policy-bypassing view is reachable from a runtime account.

## Where it is enforced

The one-shot migrator and the roles job (`internal/db`), the grants manifest `db/grants.yaml`, and the catalogue checks run against a live database.

## Proof

Annex B CI-3 (role membership and ownership), CI-4 (privileges equal the manifest exactly) and CI-9 (each system actor can do only its own job).

Tests that prove it, by Go test name: `TestD28NoSuperuserOrBypassRLS`, `TestManifestDetectsAndReportsGrantDrift`, `TestDBRolesNewPlatformDatabaseHasNoPublicGrants`, `TestT10GateEmptySecurityDefinerExceptions`.

## Read next

Enforcement detail: [Database](../database/index.md) and [Security](../security/index.md). All nine invariants: [the invariants page](invariants.md).
