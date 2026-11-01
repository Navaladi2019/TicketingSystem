# Review: TICK-345 (P3-07), PR #314

Entity-type routes in the isolation suite, and a test that a publish runs no DDL.

- Reviewer: Quality, Testing & CI Advisor
- Head reviewed: `578f042`, on `TICK-345/entitytype-isolation-no-ddl`, which targets `TICK-333/entity-types-versioned`
- Reviewed against: the TICK-345 contract and acceptance criteria, PLAN-003 I8 and I1, and ADR-001 I2

**VERDICT: APPROVE.** Nothing in the PR must change. Finding 1 is a confirmed bug in the isolation seed, outside this diff. It goes to S8 and does not block.

## Acceptance criteria

| Criterion | Result | Evidence |
|---|---|---|
| New cases appear in the isolation report | Met | CI `integration` job (PR run 41246340862): all 14 SQL paths and 6 DB paths `PASS` in `TestRegisteredPaths` |
| The no-DDL test fails on `ALTER TABLE`, `CREATE FUNCTION`, `CREATE TRIGGER` or `CREATE TYPE` | Met | Red runs in the PR description. The test also has a positive control: each statement inside a rolled-back transaction must show up in the snapshot. So an empty or wrong snapshot query cannot pass |
| Forged writes are checked with `Super()` reads, and go red if the handler trusts a body `tenant_id` | Met | `tenantsOf` reads past RLS as the superuser. Red evidence for both routes is in the PR description |
| Existing isolation cases are still green | Met | `internal/tenancy/isolation` ok, `internal/server` ok, 0 `--- FAIL` in the job |
| Only adds (D-P10) | Met | 618 additions and 0 deletions. The `registry.go` change only appends paths and constants |

The test is not parallel, and the risk of a flaky snapshot is low. `TestPublishIssuesNoDDL` does not call `t.Parallel()`. `scripts/ci/test-integration.sh` runs the shared-cluster packages with `-p 1`, and `dbtest` runs on its own shard clusters. `pg_class` is per-database, and VACUUM and ANALYZE update it in place, which keeps `xmin`. So no other test can change the snapshot during the window.

## Findings (most severe first)

### 1. Bug (in the fixture, outside this PR): seeded versions return 500 when their owner reads them

- **Where:** `internal/tenancy/isolation/testdata/seed.sql:106` stores `fixed_columns` and `states` as `'[]'`. `internal/entitytypes/entitytypes.go:107` scans them into `map[string]string` and `metadata.StateModel`. `internal/server/entitytypes.go:153-154` returns the scan error unmapped.
- **Failing scenario:** tenant B, the owner, calls `GET /api/v1/entity-types/ticket` or `GET /api/v1/entity-types/ticket/versions/2` on the isolation seed. The scan fails and the response is 500 `internal_error` instead of 200.
- **Proof:** I ran pgx v5's `jsonb` codec on `[]` with both target types, and both fail with `json: cannot unmarshal array into Go value of type ...`. I did not make the HTTP call itself. The handler path from the scan error to the 500 is a direct read of the code above.
- **Scope:** the API always writes objects (`json.Marshal` of a map or struct, `entitytypes.go:186` and `:268`), and this seed is the only writer of `[]` in the repo. Production data is not affected today. But the columns have no shape `CHECK`, so any writer that is not the API (a future migration, a seed, a manual fix) can store a row that breaks every read of that type.
- **Fix (two parts):** change the seed to valid objects, for example `'{}'` and `'{"initial":"new","states":["new"],"transitions":[]}'`. Then decide with the Postgres & Data Modeling Specialist whether to add `CHECK (jsonb_typeof(fixed_columns) = 'object' AND jsonb_typeof(states) = 'object')`.
- **Test that would catch it:** finding 2.

### 2. Missing test (not blocking): no case reads a seeded version as its owner

The isolation cases only check that tenant A gets 404 on B's seeded versions. No positive control reads them as B. That gap is why finding 1 went unnoticed. Once the seed is fixed, add a control to `etReadOtherTenantsKey`: B reads `ticket/versions/2` and gets 200. Without it, the 404 comparison would still pass even if the route returned 404 for everyone.

### 3. Should-change (not blocking): I8 is proven for the synchronous publish only

`TestPublishIssuesNoDDL` covers the HTTP publish, including the NeedsBackfill classification. The backfill that NeedsBackfill schedules is not in this window. When the backfill worker lands, it needs the same catalogue snapshot around a full backfill run. Track this on the backfill task, not here.

### 4. Preference: the name check in `etListOwnOnly` is redundant

`strings.Contains(it.Name, " B")` (`entitytype_paths.go`, `etListOwnOnly`) is a heuristic. The `slices.Equal(keys, own)` check after it already proves the list exactly. The name check can be removed.

## CI

`gh pr checks 314` at review time, head `578f042`. I watched once for about 10 minutes.

- **Pass:** `changes`, `lint`, `unit`, `integration`, `temporal-server`.
- **Fail:** `story-branches`, on the `pull_request` run only. The cause is repo state: `TICK-334/workflow-interpreter: orphaned`. It is not caused by this diff, and the same check passed on the `push` run of the same SHA.
- **Still pending when the watch ended:** `stack`, `stack-db`, `stack-internal`.
