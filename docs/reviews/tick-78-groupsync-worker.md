# Review: T25 group sync worker (TICK-78, PR #113)

- Reviewer: Quality, Testing & CI Advisor
- Head reviewed: `4b25bce`; finding 1 re-checked at `360846f`
- CI: green at `4b25bce` ([run 40280627013](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40280627013)) and at `360846f` ([run 40281777224](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40281777224)). All 8 jobs pass, and the 6 groupsync integration tests ran against PostgreSQL 16.10.
- Specs: TICK-78 contract and acceptance criteria, the T16 carry-forward (TICK-69), audit TICK-93, the 2026-10-25 ruling on directory membership, and PLAN-001 D-P9, D-P10 and D-P13.
- Out of scope: tenant isolation. That review belongs to the Security & Multi-Tenancy Reviewer (D-P10: `cmd/nava-groupsync/` is a security path). Their adversarial tests are in PR #125, which QA approved at `884ccd1`.
- Merged: PR #113 and PR #125 on 2026-10-26.

## Verdict

Approve. Finding 1 is closed at `360846f`. Findings 2 to 5 are follow-ups and do not block the merge.

| Finding | Status | Tracked in |
|---|---|---|
| 1. CRLF in `review-paths.yaml` | closed at `360846f` | n/a |
| 2. Deleting a referenced principal | open; must close before any new table with a foreign key to `principals` merges | TICK-220 |
| 3. One failing tenant does not stop the others | partly closed by PR #125; the rest is open | TICK-221 |
| 4. Snapshot read inside the open transaction | accepted, no action at current scale | n/a |
| 5. Two replicas and a directory revert | accepted, documented in code | n/a |
| CI harness: integration hang leaves no output | open | TICK-225 |

## Acceptance criteria

| Criterion | Test | Result |
|---|---|---|
| Group removal reflected after one cycle | `TestGroupRemovalReflectedAfterOneCycle` | pass |
| Group outside `/t/<tenant_id>/` never synced | `TestOutsideSubtreeNeverWritten` | pass |
| Two processes never sync one tenant concurrently | `TestTwoSyncersNeverSyncOneTenantConcurrently` | pass |
| OU subtree filter (not `ou/`, another tenant's `ou/`, the container, outside `/t/`) | `TestOutsideSubtreeNeverWritten` | pass |
| A→B→A move refused, `org_units` unchanged | `TestOrgUnitCycleRefused` | pass |
| Walk stops at the depth bound on a seeded cycle | `TestWalkStopsAtDepthBound` (bound 5: `ErrOrgUnitDepth`; bound 20: `ErrOrgUnitCycle`) | pass |
| `'[]'` links; disabled users written `enabled=false` | `TestGroupRemovalReflectedAfterOneCycle` | pass |
| TICK-93: `groupsync_lag_seconds` histogram with no tenant attribute; over-10-min log line | `groupsync_test.go` (unit) | pass |
| Unchanged snapshot skips apply | `TestUnchangedSnapshotSkipsApply` | pass |

## Findings

### 1. Bug: `docs/delivery/review-paths.yaml` committed with CRLF line endings (closed at `360846f`)

`.gitattributes` sets `*.yaml text eol=lf`, but the PR's blob has CRLF line endings. The only real change is one line (`cmd/nava-groupsync/` loses `planned: "PLAN-001 T25"`). The PR diff shows all 163 lines of a D-P10, CTO-gated file rewritten, which hides the real edit from its reviewers and rewrites the blame for the whole file.
Fix: `git add --renormalize docs/delivery/review-paths.yaml`, then commit. The PR diff for that file should then be about 2 lines.
Closed: at `360846f` the file is LF and its diff against `main` is the one real change (3 lines: 2 removed, 1 added).

### 2. Breaks under a schema change: deleting a principal fails once another table references `principals`

`applyPrincipals` hard-deletes principals that leave the directory. Today only `group_members` references `principals`, with `ON DELETE CASCADE`. When a later table references a principal, for example a ticket assignee or an audit actor, with the default NO ACTION, that DELETE fails. That tenant's transaction then rolls back on every cycle, so its sync stops and the lag grows without bound.
Route to the Postgres & Data Modeling Specialist for a decision before the first table that references `principals` lands. The options are soft-delete (`enabled=false` plus a `removed_at`) or a cascade rule. Missing test: an integration test that removes a principal referenced by another row.
Tracked in TICK-220 (backlog).

### 3. Missing test: one failing tenant does not stop the others

The package doc and `Once` promise that "a failing tenant rolls back alone and does not stop the others". No test covers it. Add one: two tenants, where one has a snapshot that fails (for example an org unit whose parent is not in the subtree, which gives `ErrBadSnapshot`). Assert that the failing tenant's tables are unchanged, the other tenant is written, and `Result.Err` is set only on the first. This also covers the `ErrBadSnapshot` rollback, which no test exercises today.

Partly closed by PR #125 (`TestHostileSnapshotNeverTouchesAnotherTenant`, step 2). Tenant A sorts first, its snapshot names an org-unit parent it does not have, and the test asserts that `Result.Err` is `ErrBadSnapshot`, A's rows are unchanged, and B still syncs. That covers "a failing tenant rolls back alone" and the `ErrBadSnapshot` rollback. Still open in TICK-221: an assertion on the `ErrBadSnapshot` log line, a parent missing from the snapshot altogether, and the comment fix at `apply.go:340`.

### 4. Breaks under a slow Keycloak (low): the directory snapshot is read inside the open tenant transaction

`syncTenant` calls `ChangesSince` (HTTP to Keycloak) while the transaction and advisory lock are held, for up to `TenantTimeout` (2 minutes). Under PgBouncer's transaction mode, that keeps one server connection busy for the whole call. Tenants are synced one after another, so this is one connection per replica. It is acceptable at the current scale, and the code comment explains why (the snapshot is read under the lock). Revisit if the `nava_groupsync` pool is ever shared or synced in parallel.

### 5. Breaks under two replicas and a directory revert (accepted)

The per-process cursor can skip a tenant whose tables another replica changed. The impact is bounded by one interval while this replica keeps losing the lock. A code comment in `syncTenant` now documents it, and `TestUnchangedSnapshotSkipsApply` covers the cursor. No action.

## Follow-ups outside this PR

- There is no compose service for `nava-groupsync` yet. T19 (TICK-75) or T26 (TICK-82) adds it with `NAVA_KEYCLOAK_URL`, `NAVA_KEYCLOAK_REALM`, `NAVA_GROUPSYNC_CLIENT_ID` and `NAVA_GROUPSYNC_CLIENT_SECRET`.
- CI harness (owned by Quality): `scripts/ci/test-integration.sh` prints group logs only after every group finishes, and `go test` uses the default 10m `-timeout`, the same as the job timeout. A hang (as on the first run of this PR) therefore leaves no output. Fix: a shorter `-timeout`, and logs printed when the step is cancelled. Tracked in TICK-225 (backlog, Quality).
- Security review risks, filed by the Technical Lead: TICK-222 (refuse to start as a superuser or `BYPASSRLS` role) and TICK-223 (filter `federated_links` to the tenant's bound IdP aliases).

## How to rerun

- Unit: `task test`
- Integration: the GitHub `ci` workflow (`integration` job) on the PR, or `gh workflow run ci --ref <branch>`.
