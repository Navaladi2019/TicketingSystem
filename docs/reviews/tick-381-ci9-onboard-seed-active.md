# Review: TICK-381 (P3-06a), PR #328

**Change:** CI-9 path "nava_onboard cannot seed an active tenant" (`internal/tenancy/isolation/role_paths.go`, `onboardCannotSeedActive`) and two fixtures in `ci9_fixture_isolation_test.go`.
**Head reviewed:** `4b0056c`, into `TICK-333/entity-types-versioned`.
**Reviewer:** Quality, Testing & CI Advisor.
**Verdict:** APPROVE.

## What was checked

- **Contract:** the path runs the production `entitytypes.Store.Seed` as `nava_onboard`. Every refusal must be SQLSTATE 42501 (`refused`), so a refusal from a constraint or a type error does not count as PASS. No production code or grant changes.
- **Positive control:** the seed into provisioning tenant P must insert rows, and P's activation must touch exactly 1 row. Both are labelled `control:`, so the fixture harness rejects a FAIL caused by a broken control.
- **ON CONFLICT DO NOTHING:** the retry with the same ids would be a silent no-op if the conflict check ran before WITH CHECK. The path requires 42501 on the retry, and it passes in CI, so a conflict does not get past the policy.
- **Per-table coverage:** Seed stops at the first table that refuses it. With only the full Seed, a missing guard on `entity_definition_versions` would never show. The definition-only and version-only inserts, plus one fixture per table, close that gap.
- **Fixtures fail for the right reason:** in GitHub Actions (`integration` job, run 41257846875) both fixtures turn PASS into FAIL through a successful insert, not through an error:
  - `entity_definitions_onboard stops testing provisioning`: "tenant P after activation, definition only: succeeded"
  - `entity_definition_versions_onboard stops testing provisioning`: "tenant P after activation, version 2 only: succeeded" and "in tenant A, version 2 only: succeeded"
- **Undo is exact:** the harness compares `aclSnapshot` before and after each undo, and the undo text matches `20261031150000_entity_definitions.sql`. Both pass in CI.
- **Isolation of the run:** all writes happen in `inTenant` transactions that roll back, each statement in its own savepoint. Tenant P is created and removed by `withProvisioning`, so the run leaves no rows behind.
- **CI:** `TestRegisteredPaths/nava_onboard_cannot_seed_an_active_tenant` passes, and all required jobs are green on `4b0056c`.

## Findings

None blocking.

1. **Preference:** the "seed tenant B from A's transaction" and "seed tenant A from P's transaction" cases are refused by the `tenant_id = app.tenant_id` term, not by the provisioning test. Neither new fixture exercises them. That is correct as written, because the existing binding fixtures already cover the tenant term. A one-line comment saying so would stop a later reader from expecting a fixture for them.
2. **Preference:** `seedOne` loads `metadata.Template("incident")` on every call (six times per run). It costs little. Loading it once in `onboardCannotSeedActive` would be tidier.

## Isolation note

This change adds tests only. It changes no isolation behaviour, so there is nothing to route to the Security & Multi-Tenancy Reviewer, who authored it.
