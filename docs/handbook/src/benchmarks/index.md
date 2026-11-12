---
title: "Benchmarks and budgets"
section: benchmarks
status: delivered
sources: [docs/budgets.md, docs/quality/acl-perf-800k.md, docs/quality/pgbouncer-load.md, docs/data/audit-migration-b-bench/RESULT.md, scratch/audit-backfill-s8/RESULT.md, bench/acl-perf/run.sh, Taskfile.yml, docs/handbook/src/benchmarks/benchmarks.yaml]
adrs: [ADR-004-E3, ADR-004-E4, ADR-004-E6, ADR-005]
tests: [TestCommitCheckBudget, TestConnectionBudgetFits, TestAllocatorPoolSizing, TestPatchHistoryIsCapped]
---

A budget is a cost the platform keeps in bounds. [docs/budgets.md](../../../budgets.md) is the normative list: what is measured, how, the limit, the baseline and the owner. A change that breaks a budget needs a new measurement and the owner's sign-off. This page puts each budget next to its **latest result of record**.

## How to read the tables

- **One row per budget and host class.** Results from different hosts cannot be compared. The commit-check trigger reads 80 µs per probe on one CI CPU model and 195 µs on another, in the same job.
- **Host classes.** *CI runner* is a GitHub-hosted `ubuntu-latest` runner, named with its CPU model. *Idle reference machine* is the Docker Desktop on Windows host with nothing else running. *Reference host* is the same kind of machine with other containers present. *Laptop* is a developer machine.
- **Verdicts.** **met**: within the limit. **warn**: over a report-only threshold; the CI job passes with a warning. **waived** and **known miss**: over the target, with the decision that accepts it linked in the row. **no result yet**: the budget exists but no measured result is committed. The row says "no recorded result" and is not left out.
- **Commit.** The commit the result was measured at. "not recorded" means the source did not record it. It is never guessed.
- **Nothing is fetched.** The figures come from `docs/handbook/src/benchmarks/benchmarks.yaml`, written by hand from the sources. Run links are plain links. The build fails when a row of `docs/budgets.md` has no entry there, or when a source file or heading no longer exists.

## Results

<!-- gen:benchmarks -->

## Where each number comes from

| Measurement | How to run it | Where it runs |
|---|---|---|
| Commit-check trigger | `task test:commit-check-budget` (`TestCommitCheckBudget`) | CI job `commit-check-budget`, report-only (ADR-004-E4). The baseline is the median of at least 3 runs of the job on one CPU model (ADR-004-E6). A laptop run is not a baseline |
| Lists, single reads and creates at 800k records (P3-36) | `task bench:acl-perf` (`bench/acl-perf/run.sh`), about 32 minutes | Manually, on the reference host; not in CI. Each create attempt counts only when it passes the transport, host-speed, storage and container gates |
| Migration lock holds | `scratch/audit-backfill-s8/run.sh`, `docs/data/audit-migration-b-bench/run.sh` | Manually, on a developer host, against `postgres:16.10` in Docker |
| Connection sizing | `TestConnectionBudgetFits`, `TestAllocatorPoolSizing` in `task test` | Every test run: a static sum, nothing is timed |
| Relay claim latency (T33) | `task bench:relay` (`BenchmarkClaim`) | Report-only; no result is committed |

## Not on this page

- **The 100/s open-loop create run** (TICK-700) is informational: it has no target and no verdict. It is in [acl-perf-800k.md](../../../quality/acl-perf-800k.md#100s-open-loop-create-informational-tick-700-2026-11-11-0443-utc).
- **The PgBouncer load test** (T06) checks correctness under load, not a latency budget: 50 clients on 5 server connections with zero prepared-statement errors. See [pgbouncer-load.md](../../../quality/pgbouncer-load.md).
- **Production numbers.** Every result here comes from a CI runner or a local Docker host. The go-live benchmark on production storage is the trigger of accepted risk `AR-S10-02` in [accepted-risks.md](../../../security/accepted-risks.md).

## Budget owners

<!-- gen:budgets -->
