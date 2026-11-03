# Review: TICK-509, commit-check budget rows in CI (PR #420)

Reviewer: Quality, Testing & CI Advisor. Head reviewed: `25ca403`. Base: `TICK-419/one-audit-log`.

**Verdict: CHANGES REQUESTED.** The code, the method and the gate are correct. The blocker is that the required job is red on the head commit. Two runs of `commit-check-budget` on the same SHA disagreed: the import row read 129.5 µs on one runner and 190.9 µs on another, against a ceiling of 175. As wired, the gate would turn `main` red on roughly one run in three. Deciding the ceiling against the runner class is the Technical Lead's escalation to the CTO, not a code change.

## CI evidence on `25ca403`

Both runs used `postgres:16.10`, `ubuntu-latest` with 4 vCPU, `TestCommitCheckBudget` alone on the runner, and 38 partitions.

| Run | Per call, clause, N = 200 / 1k / 5k (max/min) | Ref, N = 200 | Ratio, N = 200 | Import, clause / ref (median of 3) | Import ratio | Verdict |
|---|---|---|---|---|---|---|
| 41676703902 (push) | 170.4 / 170.6 / 165.1 (1.03) | 117.5 | 1.45 | 129.5 / 13.3 | 9.76 | pass |
| 41676713796 (pull_request) | 233.2 / 234.6 / 228.8 (1.03) | 151.4 | 1.54 | **190.9** / 19.1 | 9.98 | **rerun** (import over 175) |
| 41673342312 (earlier, `2073e3c`) | 170.5 / 163.6 / 164.8 (1.04) | 133.5 | 1.28 | 127.7 / 13.1 | 9.75 | pass |

Every other job is green, on both runs.

## Findings, most severe first

1. **Breaks under runner-host variance: the import row flaps on a dedicated runner.** `docs/budgets.md` import row and `.github/workflows/ci.yml`, job `commit-check-budget`. The failing run had the runner to itself. Both of its arms were about 1.4× slower than in the passing run: ref import 19.1 vs 13.3 µs, and clause per call 233 vs 170 µs. The ratios did not move: 9.98 vs 9.76. So the spread comes from the host, not from contention, and the dedicated job does not remove it. This partly falsifies the contention-only diagnosis. The headroom is 175 / 129 = 1.36×, which is below the observed host spread of about 1.45×. The single-subject row has 300 / 234 = 1.28× headroom on the slow host, so it can flap too. Because the job is required and sits in `main-red`'s `needs`, a slow host opens a P0 "main is red" issue with nothing wrong. This is not S3: both ratios are in range. **Route:** Technical Lead to CTO (ceiling vs runner class, as the TL already offered). Options for them: pin a fixed larger-runner class, or keep the absolute ceilings as report-only in CI and let the job fail only on S3 (ratio out of range) or on flatness. The second option changes the ADR-004-E3 verdict rule, which belongs to the Solution Architect. Nobody changes the ceilings in this PR.
2. **Breaks under a rename: the required job can pass vacuously.** `scripts/ci/commit-check-budget.sh:58`. If `TestCommitCheckBudget` is renamed, `-run '^TestCommitCheckBudget$'` matches nothing and `go test` exits 0 with "no tests to run". The job stays green and measures nothing. The `grep -vx` exclusion in `scripts/ci/test-integration.sh` no longer matches either, so the renamed test goes back into the contended shards. `check-ci-steps` only checks the `make` step, so it does not catch this. **Fix (engineer):** after `go test`, have the script fail unless the output contains both `COMMIT-CHECK-BUDGET row=single_subject` and `COMMIT-CHECK-BUDGET row=import`, for example by `tee`ing to a file and running `grep -q`.
3. **Missing observability: the runner's CPU is not logged.** `scripts/ci/commit-check-budget.sh:53`. The script logs only `nproc 4`, which is identical on the fast host and the slow one. Finding 1 cannot be attributed without the CPU model. **Fix (engineer):** log `grep -m1 'model name' /proc/cpuinfo` on the "cluster healthy" line, so the CTO's decision rests on data.
4. **Preference: the S2 tenant prefix differs.** The test writes as `0190b000-…-04`, the fixture tenant that is in the young partition's statistics, which is the intended shape. The archived `scratch/audit-commit-check/budget.pgbench` uses `0190d000-…-04`. That is harmless, but the comment in `commit_check_budget_test.go` says "as in the TA's budget.pgbench". Optional: note the difference there.

## Checks requested by the Technical Lead

| Check | Result | Evidence |
|---|---|---|
| Pinned `postgres:16.10` | ✅ | The script reads the tag from `docs/pins.md`. Both logs show `commit-check-budget (postgres:16.10)` |
| Same runner class as `integration` | ✅ | Both jobs use `runs-on: ubuntu-latest`. The logs show `nproc 4` |
| Both arms in the dedicated job | ✅ | `budgetArms` (clause and ref) runs inside the one `TestCommitCheckBudget` call, on the same fixture and the same cluster |
| Nothing else alongside it | ✅ | The job's only test step is `make test-commit-check-budget`. The script starts one cluster and runs `go test -run '^TestCommitCheckBudget$'` against it. The compile step runs first with `-run '^$'` and finishes before the test |
| Excluded from the `integration` shards | ✅ | `grep -vx 'TestCommitCheckBudget'` in `scripts/ci/test-integration.sh`. The integration log shows 195 of 195 dbtest tests (see finding 2 for the rename case) |
| Method matches the TA's `recheck-38-partitions.md` | ✅ | `retainedMonths` matches the TA's builder statement for statement: 36 months × 200 rows, `n_distinct = 1`, RLS enabled and forced, `session_replication_role = replica`, the same xid8 and occurred_at formulas, and the same tenant expression. The young partition is rebuilt and `pg_stat_reset()` runs before every per-call run. Per-call runs use one connection (`-c 1`) at N = 200 / 1,000 / 5,000, with the same transaction body as `budget.pgbench` in simple protocol. The import runs as a fresh `nava_worker` session per run, median of 3, with the arms alternating. The 38 partitions are asserted from `pg_inherits` |
| `required.yaml` gates the job | ✅ | Mutation-tested locally with `go run ./tools/ci/cisteps`. Replacing the step's `run` makes it fail ("step … is missing"), and adding `continue-on-error: true` to the job makes it fail ("may fail unnoticed"). Unmodified, it reports `ok: 16 required steps` |
| `main-red` wiring | ✅ | `commit-check-budget` is in `main-red`'s `needs`, and `if: failure()` covers it. `check-ci-steps` does not guard that list, which is the same for every other job |
| All GitHub Actions jobs green (`gh pr checks 420`) | ❌ | `commit-check-budget` failed in run 41676713796 (finding 1). Every other job passed on both runs |
| Verdict rule unit tests | ✅ | `go test -run 'Budget\|Verdict' ./internal/db/dbtest`: 11 cases pass, including the boundaries and a zero reference. `go vet -tags integration` is clean |
| Runtime | ✅ | The test takes 13–19 s and the job 1m42s–1m49s, in parallel with `integration` (3m27s–4m04s), so the critical path does not grow |

`main` branch protection does not list `commit-check-budget` yet. The TL owns that item, and it does not block this review.

## What closes this review

- A CTO or Technical Lead decision on finding 1. Until then, a re-run is the rule's answer for the red run, but a green re-run does not make a gate trustworthy when it flaps on a third of its runs.
- Findings 2 and 3 fixed by the engineer: small script changes, no change to the method.
