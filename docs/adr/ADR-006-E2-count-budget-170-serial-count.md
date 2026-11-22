---
type: adr-erratum
id: ADR-006-E2
title: "Erratum E2 to ADR-006: CountBudget is 170 ms (the spill-then-cancel term), the budgeted count runs without parallel workers, and the (3a) leak check uses pg_ls_tmpdir()"
status: ACCEPTED
date: 2026-11-22
issue: TICK-1094
rfc: none (erratum from a spike's fail rule; source thread TICK-1083, PR #851)
supersedes: none
amends: "[[ADR-006-A2-count-time-budget]] A2-2 (the number), A2-3 (the savepoint Exec), A2-8 (3a), (3c) and a new (3e)"
source: "S-A2-cancel runs 5 and 6 (TICK-1083, PR #851 at 037e4438), archived in docs/adr/ADR-006/spikes/s-a2-cancel/; the Technical Architect's (3a) review on TICK-1082 (comment 01a20063-d698, findings 5 and 6)"
deciders: Solution Architect (author; verdict ACCEPTED 2026-11-22); Technical Architect (rev 0 critique, TICK-1094 comment 01a202d5-e1cb: no blocking items, findings 3-6 taken in rev 1, "with those changes in, I have no objection to ACCEPTED"); CTO (no tie-break needed: nothing contested)
---

# ADR-006 erratum E2: a 170 ms count budget, a serial count, and a real leak check

> **Status: ACCEPTED, rev 1** (2026-11-22). The Solution Architect gave the verdict on TICK-1094 after the Technical Architect's rev 0 critique, which had no blocking items. Nothing was contested, so no CTO tie-break was needed. Number E2 is assumed: E1 is the highest ADR-006 erratum in `docs/adr/`.
>
> **Rev 1 changelog (TA critique, comment 01a202d5-e1cb).**
> - **TA 3:** E2-1 now gives the real reason for using run 5's +45.9 ms: a conservative upper estimate. Run 5's spill cell was not gated, so it is not "the worse run" under (2a). The rejected-option row is updated to match.
> - **TA 4, 5:** (3e) is replaced with the Technical Architect's text: ABBA blocks with gates around each block, a pass line based on the standard error, a p95 check, and a plan check on `a50, pk = 3` in both arms. Ties go to arm A. The explainer's section 4 matches.
> - **TA 6:** a new "Interim" section covers PR #827 until E2's plan lands.
> - TA 1, 2, 7 and 8 confirmed rev 0 as written.

> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-006 and E1 are.

## In plain words

- **What changes:** a list's count gets 170 ms, not 200 ms. It runs without parallel workers. And the test that looks for leftover temp files now looks at the files themselves, not at a counter.
- **Why:** A2 promised that a cancelled count stops within 15 ms of its budget. Runs 5 and 6 found two cases where it doesn't. (1) A count that has spilled to disk takes up to 46 ms to stop, because the database must delete the files it wrote. (2) A count running on parallel workers took up to 71 ms, because it must wait for the workers to stop. A2's own fail rule says to subtract the measured overshoot from the budget. We do that for the spill. For the parallel case we remove the cause instead of paying 66 ms on every count.
- **What you'll see when it's done:** band (ii) lists still finish within 350 ms p95. Slightly more of them may show "N+" instead of a number.
- **Codes used below:** A2-2, A2-3 and A2-8 are decisions in [[ADR-006-A2-count-time-budget]]. *Spill*: a sort or hash too big for `work_mem` writes temporary files. *`Gather`*: the plan node that runs part of a query on parallel worker processes. *Overshoot*: how long the count phase runs past its budget. Band (ii): a filter matching ≥ 40% of a type, with an ACL that narrows it (A1).

## Decisions

| # | Decision |
|---|---|
| **E2-1** | **A2-2 re-derived (mechanical, per A2-8 (3a)'s fail rule).** The extra term is the spill cell's count-phase p99 overshoot at 8 clients, minus the 6 ms of round trips A2-2 already budgets: 45.9 − 6 = 39.9, rounded up to **40 ms**. **350 − 94 − 6 − 35 − 40 = 175, rounded down to `CountBudget = 170 ms`.** The term uses run 5's +45.9 ms as a conservative upper estimate. Run 5's spill cell was not gated (the gate after the spike failed: floor 7.86 ms, control p95 300.1 ms), so by (2a) it is inconclusive. The only gate-clean figure is run 6's +27.8 ms (a 22 ms term, 190 ms). An error in this direction only raises the `unknown` share. If R13 or R14 fires, the lever is a gated spill-only rerun (`NAVA_PERF_SPIKE_CELLS=spill`, about 10 min), which could give back up to about 20 ms. The term was measured with `max_parallel_workers_per_gather = 0`, so it prices the serial spill, which is the shape E2-2 makes universal. The pointwise bound now reads: latency ≤ non-count + 170 + 6 + 40 on every request. The 94 ms term stays the formula's input, as A2 measured it (TICK-1048 run 2: 50.7 ms, so there is 43 ms of real headroom). The floor (100 ms) and the CTO escalation are unchanged. |
| **E2-2** | **The budgeted count runs without parallel workers.** A2-3's one no-argument `Exec` becomes `SAVEPOINT list_count; SET LOCAL statement_timeout = '<ms>'; SET LOCAL max_parallel_workers_per_gather = 0`. It is still one round trip. `ROLLBACK TO SAVEPOINT list_count` already ends the count on both paths, and it undoes every `SET LOCAL` issued after the savepoint (PostgreSQL 16 `SET`: "the effects of SET or SET LOCAL are also canceled by rolling back to a savepoint that is earlier than the command"). So the page statement and the rest of the transaction keep the server's settings. This removes the parallel overshoot (run 6: +71.2 ms p99, a +66 ms term) instead of pricing it into every count, which would give 140 ms. It applies only to budgeted counts (lists with a data filter); the page statement and unbudgeted counts are untouched. **E2-2 is conditional on A2-8 (3e).** |
| **E2-3** | **A2-8 (3a)'s leak check is reworded** with the Technical Architect's text (TICK-1082, finding 5): "`pg_ls_tmpdir()` lists no `pgsql_tmp` files after the run (and 60 s after); `temp_files` is reported for spill context only." The harness also lists `pg_ls_dir('base/pgsql_tmp')`, which shows a parallel hash's shared fileset directories that `pg_ls_tmpdir()` skips, and counts `parallel worker` backends. Both stay. `pg_stat_database.temp_files` counts files created, so it can show that a spill happened but never that a file was left behind. |
| **E2-4** | **A2-8 additions.** (3c) is extended: inside the savepoint `current_setting('max_parallel_workers_per_gather') = '0'`, and after `ROLLBACK TO` it is back to its earlier value, on both the success and the 57014 path. The next transaction on the same pooled connection has the server default. **(3e) new, the serial-count check** (see "Context"). It compares arm A (E2: workers off, 170 ms) with arm B (the fallback: server settings, 140 ms). On the kit's fresh stack, at 8 clients, run blocks of 15 s in the order A, B, B, A, with the (2a) host gate before and after each block. A block with a failed gate is rerun. Pool each arm (about 800 requests). Cells: the kit's band (ii) count, and `a50, pk = 3`. Record, per arm and cell: n, the `unknown` share, the count phase p95/p99, and the peak number of `parallel worker` backends. **Pass:** on each cell, arm A's `unknown` share ≤ arm B's + max(1 point, 2 × SE of the difference), where SE = √(pA(1−pA)/nA + pB(1−pB)/nB). Arm A's band (ii) p95 is also ≤ 350 ms. **Plan check:** on `a50, pk = 3`, `EXPLAIN` run on the same connection after the savepoint `Exec` shows no `Gather` under arm A and a `Gather` under arm B. **Inconclusive** if arm B's `a50` cell launched no workers: report it, and E2-2 stands on its bounded-tail argument. **Fail:** E2-2 falls away and "If E2-2 falls away" applies. Ties go to arm A, because it removes a cancel path that no clean run has bounded (+71.2 ms in run 6, gate-flagged; +195.6 ms in run 5, on a failed host). |
| **E2-5** | **Residual (new R14).** A count that would have finished inside the budget only because it ran on workers now ends `unknown`. A2-8 (7)'s recorded `unknown` share and (3e) show how often. A2-7's alert (more than 5% `unknown` over 1 h) is the production signal. |

**If E2-2 falls away** ((3e) fails): the parallel term applies. A parallel-only S-A2-cancel rerun with clean host gates (about 10 min, `NAVA_PERF_SPIKE_CELLS=parallel`) sets it, because run 6's parallel cell was gate-flagged. At run 6's figure, `CountBudget` is about 140 ms (350 − 94 − 6 − 35 − 66 = 149). E2-1 and E2-3 stand either way.

## Interim (until E2's plan lands)

S21's PR #827 sets `CountBudget` = 140 ms before it merges, or it waits for E2's plan. At its current 200 ms, the serial spill shape is over the bound even on the clean run (+27.8 ms against +15). 170 ms ships only together with E2-2 and after (3e) passes. If (3e) fails, 140 ms stays and the parallel-only rerun sets the final term.

**Not changed:** the 57014 contract, `total_state`, the 1,001 cap, RLS, the page statement, `CountBudgetFor`'s `min(CountBudget, q.Timeout)`, the A2-4 shortcut, A2-6's isolation argument and every other A2-8 item.

## Context it rests on

| Fact | Evidence | Known / inferred |
|---|---|---|
| Spill-then-cancel overshoot is systematic, not noise | Spill cell, 8 clients p99: run 5 **+45.9 ms**, run 6 **+27.8 ms** (host gates passed on both sides in run 6). Single-client count statement p50: 208.2 / 207.6 ms against `base`'s 201.6 ms, so every spilled cancel costs about 6 ms more | Known (`spikes/s-a2-cancel/run-5/raw.md`, `run-6/raw.md`) |
| Parallel cancel overshoot is larger | Run 6 parallel cell: +39.5 ms (1 client), **+71.2 ms** (8 clients). The statement p50 is 229.1 ms, 29 ms over on every cancel. The host gate failed narrowly (control p95 80.6 and 84.3 against 80.3) | Known; the 8-client figure is the less certain one |
| The cancel path holds in every cell | 600 of 600 answers were 200 + `unknown` + `total: null` + 50 items, then `commit`. No `pgsql_tmp` file and no parallel worker was left after the run or 60 s later | Known |
| **The production planner already parallelizes a band (ii)-shaped count** | Run 6, `a50, pk = 3` under server settings: a `Gather` with 2 workers launched, plain median 94.5 ms. With workers off the serial time at the default `work_mem` was not measured. The only serial figure, 191.8 ms, was taken at `work_mem = 64kB` with an external sort, so it overstates | Known (plan); the serial time is **not known**, hence (3e) |
| Workers are scarce under concurrency | The server's `max_parallel_workers` defaults to 8. At 8 clients × 2 planned workers, some `Gather`s get fewer workers or none, so the parallel speedup shrinks exactly when the budget matters | Inferred (PG 16 docs on `max_parallel_workers`); (3e)'s arm B shows the size |

I don't have data on how many production counts finish inside 170 ms only because of workers. (3e) gives the first number before merge, and A2-8 (7) records the share on the kit.

## Consequences

- **Code (story S21, branch `TICK-1046/count-budget`):** `internal/records/store/list.go`: `CountBudget = 170 * time.Millisecond`, and the savepoint `Exec` gains `SET LOCAL max_parallel_workers_per_gather = 0`. The tests for (3c) and the harness check for (3e) are added. The CTO creates `Plan ADR-006-E2` on ACCEPTED.
- **Acceptance:** A2-8 (7), the S21 kit's ≥ 40% ACL cell ≤ 350 ms p95 on two fresh stacks, was met at 200 ms (TICK-1049). A lower budget can only lower the count phase, but the `unknown` share changes, so the kit is rerun once and records it.
- **Register:** AR-TICK-1022-01 reads "170 ms" on ACCEPTED. New row AR-TICK-1094-01 (R14).
- **Handbook:** the A2 count-budget page and the A2 gate benchmark page change subject (Technical Lead).

## Invariant check (incomplete: no project invariants)

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. Same statement, same RLS. `ROLLBACK TO` undoes only the `SET LOCAL`s after the savepoint; `app.tenant_id` is set earlier and survives (A2-8 (3b), unchanged). |
| I3 roles | Unchanged. `max_parallel_workers_per_gather` is a user-settable GUC; no grant. |
| I5 one authorization layer | Unchanged. `total` is exact, capped or explicitly unknown. |
| I8 configuration over code | `CountBudget` stays a platform constant. |
| I2, I4, I6, I7, I9 | Not touched. |

## Risks and residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R14 | A budgeted count that would finish inside 170 ms only with parallel workers ends `unknown`. The page is complete; `total` is null. No isolation effect (A2-6 holds: it is the same duration threshold). | Technical Lead | (3e) fails; A2-8 (7) records more than 5% `unknown` on the kit; A2-7's alert fires in production |
| R13 (A2) | Unchanged in kind. The threshold moves from 200 to 170 ms, so the `unknown` share rises somewhat. | Technical Lead | As in A2 |

## Rejected options

| Option | Why |
|---|---|
| Keep 200 ms, call the spill a tail case | A2-8 (3a)'s fail rule is mechanical: re-derive A2-2 with the measured overshoot. Run 6 confirmed the spill fails on a clean host. |
| Price the parallel term into the budget (140 ms) | It charges 30 ms more of every count's budget for a shape we can turn off for free, on a figure from a gate-flagged run. Kept as the fallback. |
| Use run 6's spill figure alone (+22 ms, 190 ms) | It is the only gate-clean figure, but it is one run. 170 ms is conservative while a single clean run is all we have. An error in this direction only raises the `unknown` share. A gated spill-only rerun is the lever if R13 or R14 fires. |
| Turn parallel off server-wide or for the `nava_app` role | It would also slow page statements and unbudgeted queries that A2 never measured. The `SET LOCAL` covers only the count. |
| Cancel from Go (a context deadline) | A2-3 already rejected it: it turns every budget hit into a 5xx and adds a second oracle. |
| Raise `work_mem` for the count to avoid spills | It trades a bounded cancel cost for unbounded memory per connection under 8+ clients. Rejected in A2 too. |

**Dissent:** none.

## Competitive Gate (ADR-008)

**Result:** PASS.
1. **Axis moved:** X3 (speed): band (ii)'s pointwise bound is restored at 350 ms p95, including spilled counts (it was 200 + 46 ms over A2's term).
2. **Pain points:** keeps CP-FW-01 ("slow loading at high ticket volume") reduced. Risks: a slightly higher "N+" share, accepted as R14.
3. **Measured by:** spike S-A2-cancel (runs 5, 6); A2-8 (3e) and (7) (planned; (3e) as rewritten in rev 1).
4. **Parity:** I1–I9 unchanged.
