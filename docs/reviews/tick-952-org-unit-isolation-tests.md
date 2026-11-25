# Review: PR #810, TICK-952 (S31 P11-19) org-unit isolation tests

Reviewer: Quality, Testing & CI Advisor. Head reviewed: `TICK-952/org-unit-isolation-tests` at the single commit "adversarial org-unit isolation tests and review". Specs read first: the TICK-952 attack list, `db/migrations/20261121000000_org_unit_members.sql`, `store.primaryUnitSQL`, and the `Denied` and `OwnRowsOnly` semantics in `internal/tenancy/isolation/registry.go` and `paths_isolation_test.go`.

**Verdict: CHANGES.** Two new registry paths cannot go Red. One guard named in attack 1 has no path that can go Red. The isolation suite, which is the only thing that runs the 12 new paths, did not run in CI on this PR.

## Findings, most severe first

### 1. Missing test: two attack-5 and attack-1 paths pass with their guard removed

The `Denied` mode treats SQLSTATE class 23 as a denial. The comment on `Denied` (`registry.go:85-89`) warns about this: "aim an UPDATE or DELETE where an integrity error cannot hide a reached row". These two new paths break that rule:

- **`org_units: group sync deletes its own units`** (attack 5, the DELETE grant). This path deletes `OrgUnitA1`. In the seed, `OrgUnitA1` has a membership row (`seed.sql:121`) and record a2 points at it (`seed.sql:174`). Neither foreign key cascades. If someone restores the DELETE grant and the `org_units_delete` policy (what the migration's Down does), the DELETE reaches the row, gets 23503, and the path reports "denied". So the path passes whether or not the guard is there.
  **Fix:** have the path delete a unit with no members, no records and no children, and seed one for it (for example a leaf `OrgUnitA2`). Or assert the grant directly: `SELECT has_table_privilege('nava_groupsync','org_units','DELETE')` must be false.
- **`org_unit_members: the app moves tenant B's primary units`** (attack 1, nava_app write). The statement `SET is_primary = NOT is_primary` on tenant B's three rows sets b1's `5b1` row to primary while b1's `5b2` row is still primary. The non-deferrable index `org_unit_members_one_primary` raises 23505, which counts as "denied". Without the guard the path stays green, depending on scan order (a sequential scan visits the `5b1` row first). The same pattern is already in the TICK-947 path `group sync moves tenant B's primary units` (`registry.go:367`), which is a pre-existing bug of the same kind.
  **Fix (both paths):** use `UPDATE org_unit_members SET is_primary = false WHERE tenant_id = $1 AND is_primary`. It can only clear rows, so no constraint can hide a reached row.

How to show Red for each: temporarily `GRANT DELETE ON org_units TO nava_groupsync` and recreate `org_units_delete`, or give nava_app an UPDATE policy with `USING (true)`. The path must fail. Today it passes.

### 2. Missing test: nothing tests group sync's own INSERT WITH CHECK on `org_unit_members`

Attack 1 says tenant B's memberships are "unwritable from tenant A's session ... for the groupsync role". nava_groupsync is the one role that has INSERT, so `org_unit_members_insert WITH CHECK` is the only guard. Both groupsync INSERT paths fail on the composite foreign key before RLS is ever checked:

- `group sync adds tenant A's principal to tenant B's unit` (TICK-947) uses tenant A with B's unit, so it always gets 23503.
- `group sync adds tenant B's principal to tenant A's unit` (new) uses tenant A with B's principal, so it always gets 23503.

If the WITH CHECK is dropped, both stay green. The new path that does write `(TenantB, OrgUnitB1, PrincipalB2)` runs as nava_worker, and nava_worker has no INSERT grant, so it is the grant that refuses it, not RLS.
**Fix:** add the same tuple for nava_groupsync: `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $2, $3)` with `TenantB, OrgUnitB1, PrincipalB2`, expecting `Denied`. Its foreign keys are valid, so only the policy can refuse it (42501).

### 3. Missing test (CI): the isolation job was skipped on #810, so the 12 new paths have never run in CI

`gh pr checks 810` shows `isolation skipping`. The job runs only when `needs.changes.outputs.full == 'true'` (`.github/workflows/ci.yml:615`). That output is the workflow-level `env.CI_FULL` (`ci.yml:177`), which is true only with `ci:full`, a manual run or the daily run. The story-branch rule (`ci.yml:279`) feeds only the stages step, not the output. `full-suite` then accepts the skip as a "fast run" (TICK-1003). `docs/delivery/review-paths.yaml:91` lists `registry.go` as a path that needs `ci:full`, and #798 ran with it. **I have added `ci:full` to #810.**

This is also **spec drift in the CI definition I own.** The workflow header and the TICK-952 issue both say a PR into a `TICK-*` story branch "runs every job", but since TICK-1003 the isolation suite is skipped on those PRs unless they carry the label. That is how #801 and #805 merged without it running. I will raise the fix (derive `full` from the same expression as `ci.yml:279`) as a CI task of my own. It does not block this PR beyond the label.

### 4. Missing test: attack 7 checks a copy of the pipeline lookup, not the lookup itself

`pipelinePrimaryUnit` (`internal/groupsync/orgunit_isolation_integration_test.go:108`) copies `store.primaryUnitSQL`. It is identical today (`internal/records/store/batch.go:134`). The guard that actually goes Red here is the syncer: no `is_primary` row in A. The "the ticket's `org_unit_id` stays NULL" half of the attack checks only the copy, so a later change to the real lookup will not fail this test. Add one case in `internal/records/pipeline` that runs a real create for a principal with no primary row in its tenant and asserts `org_unit_id IS NULL`. Or reference a shared exported query. Severity is low, because the syncer half is the real guard.

### 5. Preference: the attack-2 create asserts only that it got some error

In `TestRecordNeverGetsAnotherTenantsOrgUnit`, `err == nil` is the only check on the forged-primary create. An unrelated failure (a bad fixture, a validation error) would pass for the guard. Assert the tenant-carrying FK: `errors.As` to `*pgconn.PgError` with code 23503 and the `records` org-unit constraint name, or whatever typed error the pipeline wraps it in.

## What holds

- **Red for the rest of the attacks.** I traced each guard to an assertion that fails when the guard goes.
  - Attack 1: the reads have tenant B rows in the seed, so `OwnRowsOnly` is not vacuous.
  - Attack 2: the records FK carries the tenant, and the `org_unit_id` write refusal is tested.
  - Attack 3: a forged membership in tombstoned T. Dropping the `deleted_at` filter puts T and T1 in S.
  - Attack 4: a forged membership in o's X. Seeding from `org_unit_members` without joining live `org_units` puts X in S.
  - Attack 5: the groupsync payload test compares B's id, deleted_at, membership and full rows before and after.
  - Attack 6: a UUID regex over the cap error.
  - Attack 7: the syncer sets no primary from a path that exists only in B.
  - In attacks 3 and 4 the "tenant filter at every step" in the recursive query sits behind RLS. The test cannot tell the query's filter apart from RLS, which is fine as defense in depth, but the PR text claims a little more than the test proves.
- **Forged fixtures cannot leak between tests.** Each test calls `migrated(t)` (pipeline) or `cluster(t)` (groupsync). Both `DROP DATABASE ... WITH (FORCE)`, drop the manifest roles, and migrate from scratch. Neither file uses `t.Parallel`. `forge` uses `SET LOCAL session_replication_role = replica` inside its own transaction, with a deferred rollback, so the setting dies at commit or on `t.Fatal`, and the shared `super` connection never keeps it. The 10,000-unit cap seed goes into the same throwaway database. Tenant ids are random (`uuid.New`) in both packages.

## To re-review

Fix 1 and 2 in `registry.go`, and preferably 4 and 5. Then push. With `ci:full` on the PR, the next run includes `isolation`, and I will confirm every job green, `isolation` included.
