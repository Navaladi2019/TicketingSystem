# ACL list and read performance at 800k (P3-36)

Scope: TICK-621, PLAN-003 P3-36 (`docs/delivery/plan-records-path.md`) and
the D25 escalation. The question: with the real compiled ACL predicate
(P3-31, `authz.CompileRead`) inside the real list SQL (P3-34,
`records/store.List`), is list p95 ≤ 150 ms and single-read p95 ≤ 50 ms at
800k records?

## Create re-run with the host-speed gate (`15da77d6`, 16:00 UTC)

**Result: two valid rows miss, so the outcome rule releases P3-36i (TICK-692).**

| Valid row | p95 | Target | Verdict |
| --- | --- | --- | --- |
| Agent, 8 clients | **72.5 ms** | 50 ms | **MISS** |
| Admin, 8 clients | **85.0 ms** | 50 ms | **MISS** |
| 1 client | **43.2 ms** | 50 ms | meets |

The small-tenant row got no valid attempt in 3: the host-speed probe failed each time.

**Method.** This harness at `b4d7b512` (story tip `15da77d6`), with `NAVA_PERF_ONLY='^create'`. The raw report is `final-2026-11-10-fold-hostgate-run1.md`, started 16:00 UTC. Three gates apply (CTO, 2026-11-10 15:58 UTC):

| Gate | How it is measured | Limit |
| --- | --- | --- |
| Transport floor | Measured once, before the scenarios | p50 ≤ 3 ms |
| Host speed (new) | 200 round trips of a 1-client `BEGIN; COMMIT` on the handler's pool, run just before each attempt and not taken from the row's own trace | p50 ≤ 2.0 ms |
| Storage window | The commit round trip, from the row's own trace | p50 ≤ 10 ms |

The run's transport floor was p50 1.46 ms, p95 2.58 ms. A `503 retry_later` in a valid attempt counts as a miss; in a gated attempt it is only reported. There were none.

| Row | Attempt | p95 | Host probe p50 | Commit p50 | Counter wait p50 / p95 | Alloc p50 / p95 | Round trips | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 client | 1 | 43.2 | 1.67 | 7.81 | 2.43 / 4.42 | 2.55 / 4.87 | 6 | **meets** |
| 8 clients, agent | 1 | **72.5** | 1.20 | 9.33 | 3.75 / 8.20 | 4.05 / 9.53 | 6 | **MISS** |
| 8 clients, admin | 1 | **85.0** | 1.66 | 9.32 | 3.88 / 8.76 | 4.18 / 9.76 | 6 | **MISS** |
| 8 clients, small tenant | 1 | 98.2 | 3.03 | 9.98 | 4.22 / 11.05 | 4.43 / 15.53 | 6 | gated (host probe) |
| 8 clients, small tenant | 2 | 64.6 | 3.03 | 9.41 | 3.67 / 6.93 | 3.97 / 9.16 | 6 | gated (host probe) |
| 8 clients, small tenant | 3 | 72.5 | 2.52 | 9.18 | 3.76 / 8.08 | 4.06 / 9.61 | 6 | gated (host probe), no valid attempt in 3 |
| 8 clients, `gapless` | 1 | 189.0 | 2.75 | 6.42 | 36.5 / 166.1 | – | 5 | reported only |

In every attempt:
- the allocator's acquire wait was p95 ≤ 0.49 ms;
- `nava_numbering_alloc_timeouts` stayed at 0;
- there was no `409 idempotency_in_flight`.

**Where a valid 8-client create spends its time.** Agent, attempt 1: request p50 42.5 ms, p95 72.5 ms. The host probe just before it was 1.20 ms, so the host was fast.

| Round trip | p50 / p95 (ms) |
| --- | --- |
| `BEGIN` + set tenant | 3.1 / 8.1 |
| lock_timeout, claim and prefetch | 7.6 / 17.8 |
| Checks | 4.0 / 8.5 |
| `AllocTx` | 3.8 / 8.2 |
| `INSERT` | 6.1 / 10.7 |
| Completion and `COMMIT` | 9.3 / 14.9 |

- The statements sum to 33.9 ms at p50; the other 8.6 ms is handler time.
- **With 8 clients, even a trivial `BEGIN` costs 3.1 ms p50, while one client on the same host costs 1.2 ms.** That is the database and VM under eight concurrent writers, not host noise: the host-speed gate passed.
- The p95 comes from tails stacking across 6 round trips. P3-36i removes two of them: the standalone `BEGIN`, folded into the claim batch, and the separate insert, folded into the completion batch. At p50 that saves about 3 + 6 ms; at p95 it removes two of the six tails.

**The small-tenant row** is host-invalid at 16:00 UTC. Per the outcome rule it is repeated once, alone, at `15da77d6` in a different hour (scheduled for 17:10 UTC). Neither outcome of that repeat changes the verdict above.

**Repeat in a different hour: still host-invalid.**

| Start (UTC) | Floor p50 | Outcome |
| --- | --- | --- |
| 17:23 | 3.47 ms | Aborted at the floor gate, so no numbers |
| 17:48 | 1.85 ms | Ran the small-tenant row alone; every attempt failed the host-speed probe (table below) |

During both runs another task's integration Postgres (`nava-it-pg-tick692`) was at 9 to 24% CPU. The raw report for the 17:48 run is `final-2026-11-10-small-tenant-repeat-1748.md`.

| Attempt | p95 | Host probe p50 | Commit p50 | `retry_later` | Verdict |
| --- | --- | --- | --- | --- | --- |
| 1 | 113.6 | 2.98 | 10.76 | 0 | gated (probe, commit) |
| 2 | 89.4 | 2.96 | 9.90 | 0 | gated (probe) |
| 3 | 87.6 | 2.87 | 9.73 | 0 | gated (probe) |

The row is reported as host-invalid. That is not a pass and not a miss; the reference host goes to the CTO.

Not verified: in both runs, every probe after the 16:00 run's first three rows read 2.5 to 3.0 ms. That includes the very first probe of the 17:48 run, taken right after the seed. Background work in the bench database itself, such as autovacuum or analyze after the seed and the creates, may be raising the probe as well as the host. Sampling `pg_stat_activity` during the probe would settle it.

## Gated create re-run after TICK-688 (claim folded into the create transaction)

**Result: FAIL.** At story tip `15da77d6`, two rows that passed the storage-window gate still miss:
- small tenant, 8 clients: 70.6 ms;
- 1 client: 50.6 ms and 59.9 ms.

Agent and admin at 8 clients got no valid window in 3 attempts in either run. That makes the host invalid for those two rows, not a miss.

**Method.** This PR's harness, merged with the story tip (`9878baea`), creates only, with `NAVA_PERF_ONLY='^create'`.

| Gate | Rule (CTO, 2026-11-10 13:33 UTC) |
| --- | --- |
| Storage window | A scored create row counts only if its commit p50 is ≤ 10 ms. Commit p50 is taken from the create's own commit round trip in its trace. After the fold, that round trip is the completion batch ending in `COMMIT`. |
| Attempts | A gated attempt is neither a pass nor a fail. A row gets at most 3 attempts. |
| Transport floor | p50 must be ≤ 3 ms. The harness now enforces this with `t.Fatalf`. |

Each create makes **6 round trips** (it was 9 before the fold): BEGIN, claim and prefetch batch, checks, `AllocTx`, insert, then completion and commit. There were no `409 idempotency_in_flight` responses.

| File | Start (UTC) | Build | Floor p50 / p95 / p99 |
| --- | --- | --- | --- |
| `final-2026-11-10-fold-gated-run1.md` | 15:33 | `15da77d6` and this harness | 1.17 / 2.20 / 5.78 |
| `final-2026-11-10-fold-gated-run2.md` | 15:41 | `15da77d6` and this harness | 1.09 / 1.66 / 2.58 |
| `final-2026-11-10-prefold-control.md` | 15:48 | `c65064cd` (`9da6f7e1`, pre-fold), same-hour control, not gated | 2.26 / 3.46 / 4.14 |

### Every attempt (ms)

| Row | Run / attempt | p95 | Commit p50 | Counter wait p50 / p95 | Alloc p50 / p95 | Failed | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 client | 1 / 1 | 70.5 | 10.96 | – | 2.89 / 8.44 | none | gated |
| 1 client | 1 / 2 | **59.9** | 8.15 | – | 3.37 / 8.53 | none | **MISS** |
| 1 client | 2 / 1 | **50.6** | 7.81 | 2.86 / 5.29 | 3.15 / 7.49 | none | **MISS** |
| 8 clients, agent | 1 / 1, 2, 3 | 127.5, 101.5, 83.5 | 12.11, 10.36, 10.08 | – | 5.59 / 21.79, 4.19 / 17.74, 3.93 / 9.76 | 1× 503 `retry_later` (attempt 1) | gated ×3 |
| 8 clients, agent | 2 / 1, 2, 3 | 73.5, 160.5, 113.0 | 10.16, 11.25, 11.87 | 3.48 / 8.44, 4.08 / 17.82, 4.72 / 12.80 | 3.83 / 9.69, 4.45 / 21.79, 4.87 / 19.37 | none | gated ×3, **host invalid** |
| 8 clients, admin | 1 / 1, 2, 3 | 133.7, 204.8, 186.7 | 11.08, 13.17, 22.72 | – | 4.57 / 20.25, 7.68 / 24.66, 6.65 / 23.68 | none | gated ×3 |
| 8 clients, admin | 2 / 1, 2, 3 | 136.7, 125.6, 74.1 | 11.23, 27.01, 10.14 | 4.02 / 15.81, 3.89 / 11.22, 3.59 / 8.33 | 4.36 / 19.74, 4.27 / 17.24, 3.98 / 9.61 | none | gated ×3, **host invalid** |
| 8 clients, small tenant | 1 / 1, 2, 3 | 293.2, 187.6, 120.0 | 14.62, 13.97, 11.89 | – | 8.51 / 40.77, 7.91 / 24.63, 5.64 / 20.03 | 1× 503 `retry_later` (attempt 1) | gated ×3 |
| 8 clients, small tenant | 2 / 1 | 76.6 | 10.26 | 3.60 / 8.56 | 3.99 / 9.66 | none | gated |
| 8 clients, small tenant | 2 / 2 | **70.6** | 9.83 | 3.29 / 7.50 | 3.71 / 9.24 | none | **MISS** |
| 8 clients, `gapless` (not scored) | 1, 2 | 383.0, 228.7 | 12.55, 7.66 | 39.5 / 199.9 (run 2) | – | none | reported only |

Notes on the table:
- **Run 1 counter waits:** they are in that run's report.
- **Acquire wait on `nava_alloc`:** p95 ≤ 0.5 ms, with one 1.33 ms exception.
- **Both `retry_later` responses** were allocations past the 200 ms budget: `nava_numbering_alloc_timeouts` was 1 in each of those attempts. Both came while the commit p50 was 12 to 15 ms.

### The host was in a slow period: same-hour control

At 15:48 the pre-fold build (`c65064cd`) ran on the same host. Its p95 was 56.9 / 95.3 / 93.7 / 144.6 ms for 1 client / agent / admin / small tenant. The same build gave 41.4 / 52.1 / 64.0 / 97.5 at 12:58.

So at 15:33 to 15:48:
- **The host was slower than at 13:00.** Even a trivial single-client `BEGIN` took 2.4 to 3.0 ms p50, against 1.6 to 1.8.
- **The fold helped.** In the same hour the folded build is 6 to 74 ms faster at p95 than the pre-fold build.

**The gate does not catch this kind of slowness.** It reads commit fsync only. A single client never contends on fsync, so the 1-client row passes the gate with commit p50 8 ms while every round trip costs about 1.7 times its 13:00 value.

### Where a valid 8-client create spends its time

Small tenant, run 2, attempt 2: request p50 40.2 ms, p95 70.6 ms.

| Round trip | p50 / p95 (ms) |
| --- | --- |
| `BEGIN` + set tenant | 2.4 / 4.8 |
| Claim + prefetch batch | 6.8 / 17.4 |
| Checks batch | 3.6 / 8.5 |
| `AllocTx` | 3.3 / 7.5 |
| `INSERT` batch | 5.3 / 10.0 |
| Completion + `COMMIT` | 9.8 / 15.9 |

- The statements sum to 31 ms at p50. The other 9 ms is handler time between round trips.
- The p95 comes from tails stacking across 6 round trips, not from one statement.
- The claim batch's tail (p95 17.4 ms) is the largest after the commit. That batch now carries the `lock_timeout` setting, the claim `INSERT` and the prefetch.

**For the CTO.** The commit count is down to one, so the remaining levers are design questions:
1. **Fewer round trips per create.** For example, merge the checks batch into the prefetch, or send the insert and the completion in one batch with the commit.
2. **A gate that also covers per-round-trip host speed.** For example, score a create row only when its `BEGIN` p50 is ≤ 2 ms as well. That is needed before a 1-client verdict can be trusted on this host.

## Final re-run after TICK-656 and TICK-657 (gaps-allowed numbering)

**Runs.** All runs are at story tip `9da6f7e1` with this PR's harness, on the reference host (Docker 29.9.2 on WSL2, 16 vCPU, 12.5 GB; `postgres:16.10`). The raw reports are in `docs/quality/acl-perf-800k/`. Only an idle `postgres` container from another task and the multica stack were up, both under 1% CPU. Every run passed the floor gate (p50 ≤ 3 ms).

| File | Start (UTC) | Rows | Floor p50 / p95 / p99 |
| --- | --- | --- | --- |
| `final-2026-11-10-gaps-create-run1-log.txt` | 12:54 | creates. Aborted at the gapless row's setup (a raw `UPDATE` hit the audit trigger), so only the log's `RESULT` lines exist | 0.57 / 1.19 / 1.81 |
| `final-2026-11-10-gaps-create-run2.md` | 12:58 | creates, gapless row included | 1.09 / 1.44 / 1.81 |
| `final-2026-11-10-gaps-create-run3.md` | 13:02 | creates | 1.17 / 2.35 / 5.88 |
| `final-2026-11-10-gaps-create-run4-checkpoint.md` | 13:06 | creates, `NAVA_PERF_CHECKPOINT=1` | 1.07 / 1.71 / 2.59 |
| `final-2026-11-10-lists-9da6f7e1.md` | 13:11 | `state=new`, agent page 1 and page 20, admin `state in` | 1.05 / 1.55 / 1.77 |

### p95 against target (ms)

| Row | Run 1 | Run 2 | Run 3 | Run 4 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- |
| Create, agent, 1 client | 40.0 | 41.4 | 66.0 | 51.6 | 50 | **Meets in 2 of 4 runs.** Run 3's floor p99 was 5.9 ms. |
| Create, agent, 8 clients (`gaps`) | 80.3 | 52.1 | 76.7 | 93.7 | 50 | **MISS** (best 52.1) |
| Create, admin, 8 clients (`gaps`) | 97.8 | 64.0 | 72.4 | 102.3 | 50 | **MISS** (best 64.0) |
| Create, small tenant agent, 8 clients (`gaps`) | 49.0 | 97.5 | 92.1 | 55.1 | 50 | **MISS** (meets in 1 of 4 runs) |
| Create, agent, 8 clients, `gapless` type | - | 183.6 | 169.1 | 150.7 | none | Reported only: the type chose this cost (D-3-9) |

| Row | p95 | Target | Verdict |
| --- | --- | --- | --- |
| Agent list by group, page 1 / page 20 | 73.5 / 71.2 | 150 | meets |
| **Agent `filter[state]=new`** | **76.2** (was 192.6) | 150 | **meets.** The plan now uses `records_list_group_state` (TICK-656): `count` is an Index Only Scan with no `state` filter, where before it read about 20k rows and dropped them |
| Admin `state in`, `sort=-updated_at` | 68.2 (was 118.6) | 150 | meets, no regression |

### Creates: the counter queue is gone, and the commits are what remain

**Allocation (`nava_numbering_alloc_*`), 8 clients, `gaps` types, runs 2 to 4:**

| | p50 | p95 |
| --- | --- | --- |
| Allocation, acquire included | 2.2 to 3.3 ms | 4.8 to 7.6 ms |
| Acquire wait on the 4-connection `nava_alloc` pool | 0.25 to 0.28 ms | 0.48 to 1.46 ms |

`nava_numbering_alloc_timeouts` stayed at 0. There were no `retry_later` responses and no failed requests in any run.

**Counter wait.** This is the `AllocTx` trip as the handler's tracer sees it.

| Rows | Wait p95 |
| --- | --- |
| 8-client rows, `gaps` | 3.6 to 5.6 ms (it was 133 to 310 ms when the counter sat in the create transaction) |
| `gapless` row | 123.6 to 151.4 ms |

**Where an 8-client create spends its time.** Run 2, agent, statement p50s in ms. They sum to about 35 ms, against a request p50 of 39.6.

| Round trip | p50 (ms) |
| --- | --- |
| Idempotency claim (`BEGIN` + set tenant, `INSERT`, `COMMIT`) | 1.8 + 2.3 + **7.0** |
| Create (`BEGIN` + set tenant, prefetch batch, checks batch) | 1.8 + 4.6 + 2.5 |
| Allocation (`AllocTx`, `synchronous_commit = off`) | 2.4 |
| `INSERT` batch | 4.1 |
| Completion and `COMMIT` batch | **8.7** |

- **The two synchronous commits are the cost, and so is their variance.** One commits the idempotency claim; the other commits the create. Their fsync latency on this host switches between two levels:

  | Commit p50, claim / final | Rows near target | Rows missing by 40 to 50 ms |
  | --- | --- | --- |
  | 7 / 9 ms | 49 to 55 ms | |
  | 17 to 25 / 17 to 26 ms | | 92 to 102 ms |

  That is why the slow row moves between runs: small tenant in runs 2 and 3, agent and admin in run 4.
- **The seed's checkpoint is not the cause.** Run 4 issues `CHECKPOINT` after the seed, and the slow window moved instead of going away.
- **Allocation is about 7% of the request.** No change to the counter or the allocator pool would bring these rows under 50 ms.

**Levers, for the CTO's decision.** QA does not implement either; both are scoped by owners.
1. **Fold the idempotency claim into the create transaction.** This was the first trigger the Technical Lead recorded on 11-09, and it was not opened. It removes one synchronous commit and two round trips: about 11 ms at p50 when commits are fast, and about 25 ms when they are slow. In the fast window this would bring all three rows under 50 ms. In the slow window they would still miss.
2. **The reference host's fsync.** Docker Desktop on WSL2 writes WAL to a virtual disk whose commit latency switches between levels, as shown above. Whether production storage behaves the same is a question for the infra owner. It is not a reason to relax the target.

**The 1-client row is now borderline:** 40.0, 41.4, 51.6 and 66.0 ms, against 28.7 ms on 11-09 with gapless code. A `gaps` create makes 9 round trips instead of 8. Its p50 is 30 to 34 ms, so a single slow commit pushes p95 over 50.

## Final re-run, after TICK-647, TICK-648 and TICK-644

**Runs** (raw reports in `docs/quality/acl-perf-800k/`):

| File | When (UTC) | Tip | Harness | Rows | Floor p50 / p95 | Clean? |
| --- | --- | --- | --- | --- | --- | --- |
| `final-2026-11-09-full.md` | 11-09 18:15 | `45635f89` (has TICK-644) | story tip | all | 1.81 / 3.10 | Floor clean. Another agent's stack came up mid-run: in the rows from agent page 20 to `site=s7`, trivial statements took 10 to 25 ms p50, against 3 to 4 ms elsewhere. Those rows are contended. The rest are clean. |
| `final-2026-11-09-lists-repeat.md` | 11-09 18:41 | `45635f89` | story tip | the contended list rows | 4.02 / 8.11 | No. A meet still holds, because contention only adds time. |
| `final-2026-11-10-create-state-bound.md` | 11-10 01:40 | `caa8ce29` | this PR (2 ms bound check) | creates, `state=new`, bound check | 2.60 / 4.03 | Gate passed, then noisy: 1-client `BEGIN` p50 3.14 ms against 1.20 in the 18:15 run. Used only for the bound check. |

Gated attempts aborted at the floor: 11-09 16:58 (3.42), 17:10 (4.18); 11-10 01:21 (4.25), 01:36 (4.00).
`caa8ce29` adds only TICK-651 (a savepoint memo fix) on top of `45635f89`.

### p95 against target (ms)

| Row | p95 | Target | Verdict | Source |
| --- | --- | --- | --- | --- |
| Agent list by group, page 1 / page 20 | 72.5 / 92.5 | 150 | meets | full / repeat |
| Admin list, page 1 / page 20 | 66.6 / 59.8 | 150 | meets | repeat |
| Agent `sort=-updated_at` / `sort=assignee` | 146.2 / 117.5 | 150 | meets | full |
| Admin `state in`, `sort=-updated_at` | 118.6 | 150 | meets | full |
| **Agent `filter[state]=new`** | **192.6** (885 contended, 390 noisy) | 150 | **MISS, fixed column** | repeat |
| Requester list, page 1 / page 20 | 75.4 / 76.8 | 150 | meets (TICK-644 fixed it) | repeat |
| Small tenant: agent / admin list | 111.5 / 70.4 | 150 | meets | full |
| `data` `priority=1` / `site=s7` | 223.8 / 224.4 | 150 | known miss, TICK-646 | repeat |
| Single GET: agent / admin / small tenant | 25.0 / 21.0 / 16.1 | 50 | meets, no waiver | full |
| Create, agent, 1 client | 28.7 (p50 24.7) | 50 | meets | full |
| Create, 8 clients: agent / admin / small tenant | 162.8 / 166.5 / 82.9 | 50 | miss on gapless code; D-3-9 decided "gaps allowed", change not yet on the branch | full |

### `filter[state]=new` is a fixed-column miss

Under load in the repeat, `count` p95 is 76.8 ms and `page` is 101.2 ms, while trivial statements in the same requests stay at 3 ms p50. Run alone with `EXPLAIN (ANALYZE, BUFFERS)`, `count` takes 44.2 ms and `page` takes 38.9 ms. In both plans, `records_list_group` is matched on `(tenant_id, entity_definition_id, assignment_group_id)`. `state` is only a `Filter`, which drops 19,547 rows for the count (20,873 buffers) and 22,950 rows for the page (24,492 buffers). That is real database work, not host noise. Per the CTO's 2026-11-09 decision, this reopens the shape work.

### Creates (D-3-9)

Counter, from the clean 18:15 run:

| Scenario | Wait p50 / p95 | Hold p50 / p95 | Statements in hold |
| --- | --- | --- | --- |
| create, agent, 1 client | 3.43 / 4.54 | 5.22 / 6.87 | 1 |
| create, agent, 8 clients | 28.55 / 133.48 | 6.81 / 9.01 | 1 |
| create, admin, 8 clients | 28.73 / 135.90 | 6.81 / 9.76 | 1 |
| create, small tenant, 8 clients | 6.35 / 10.94 | 9.82 / 18.87 | 1 |

- **1 client meets** on a clean host (28.7 ms). It takes 8 round trips, and trivial statements run at the floor (`BEGIN` p50 1.20 ms). The noisy 01:40 run gave 83.7 ms, but there 1-client `BEGIN` alone was 3.14 ms p50, so that number is not used.
- **8 clients** were measured on the gapless code, which is still the tip: the counter upsert sits in the final batch of the create transaction. The decision to allow gaps (vknavaladi, 2026-11-09 17:25 UTC) is not implemented yet. These rows can only be scored once that change lands.
- **Risk for the gaps-allowed change.** The small-tenant 8-client row misses (82.9 ms) even though its counter wait p95 is only 10.9 ms. Removing the counter queue alone may not bring 8-client creates under 50 ms. Measure it after the change; it is not a conclusion yet.

### Data-filter timeout (TICK-647)

With the bound check's 2 ms data-filter timeout, all 4,396 of 4,396 requests answered `422 filter_too_expensive`: p50 29.9, p95 69.0, max 201.9 ms. None hung. At the default 2 s timeout, no `data` list hit it. The bound check first used 100 ms, which didn't fire on a single warm request (11-10 01:30 run), so it is now 2 ms.

## Re-run, after TICK-639 and TICK-640

**Run:** 2026-11-09, on `TICK-336/group-acl-masking` at `1aa5357e`. That
head has TICK-639 (list SQL in index order), TICK-640 (list indexes, `jit =
off`) and TICK-637 (`total` capped at 10,000). The later story-branch commit
`01fc85b4` adds only isolation tests. The harness, host and settings are the
same as the first run. Targets are unchanged.

**Raw reports:**
- `docs/quality/acl-perf-800k/rerun-2026-11-09.md`: every scenario, 8
  clients, 20 s each. Transport floor p50 2.65 / p95 4.85 ms.
- `docs/quality/acl-perf-800k/rerun-2026-11-09-repeat.md`: the GET and create
  rows again on a fresh database (`NAVA_PERF_ONLY='^(single GET|create)'`).
  Transport floor p50 1.65 / p95 2.76 ms.

### Verdict

**Not met.** Large-tenant lists, creates and the small-tenant single read
still miss. The page statement of a fixed-column list now meets the target on
its own (2 to 4 ms single-run). What remains has three causes: the capped
count, the `data` filters and the create path. Per the CTO's conditions,
these numbers go on TICK-336 and the task stops there.

### Reads

p95 in ms, per request, 8 clients. `count` is `total`'s
`SELECT count(*) FROM (… LIMIT 10001)`. `page` is the page statement. Both are
p95 under load, then single-run `EXPLAIN ANALYZE` in brackets.

| Scenario | Request p95 | Target | `count` p95 (single) | `page` p95 (single) | Class |
| --- | --- | --- | --- | --- | --- |
| Agent list by group, page 1 | **443.1** | 150 | 344.6 (244.1) | 36.2 (4.4) | count only |
| Agent list by group, page 20 | **367.0** | 150 | 294.3 (121.8) | 31.7 (2.5) | count only |
| Agent list, `sort=-updated_at` | **343.3** | 150 | 271.8 (103.4) | 29.7 (1.9) | count only |
| Agent list, `sort=assignee` | 290.2 | outside target | 237.2 (109.5) | 17.8 (2.1) | foreign-key sort, outside the target (option (b) on TICK-639); count only |
| Agent list, `filter[state]=new` | **473.8** | 150 | 210.9 (40.4) | 186.4 (27.5) | load: both statements |
| Admin list, `state in (new,in_progress)`, `sort=-updated_at` | **481.9** (p50 92.6) | 150 | 141.8 (34.5) | 145.7 (2.5) | load: tail only |
| Admin list, page 1 | 89.9 | 150 | 11.0 (5.5) | 24.3 (0.8) | met |
| Admin list, page 20 | 89.2 | 150 | 13.4 (6.1) | 23.1 (0.6) | met |
| Agent list, `filter[priority]=1` (data, 5%) | **840.5** | 150 | 704.2 (282.0) | 95.3 (26.5) | data filter |
| Agent list, `filter[site]=s7` (data, 0.5%) | **1,286.8** | 150 | 607.3 (274.9) | 692.9 (268.4) | data filter |
| Requester list, page 1 | **712.7** | 150 | 596.6 (634.5) | 92.1 (16.8) | known miss, TICK-644 |
| Requester list, page 20 | **698.9** | 150 | 566.0 (206.3) | 91.8 (7.3) | known miss, TICK-644 |
| Small tenant: agent list | 101.6 | 150 | 16.2 (1.6) | 27.3 (1.3) | met |
| Small tenant: admin list | 68.2 | 150 | 7.8 (0.7) | 19.2 (0.6) | met |

What the plans show:

- **The count is the agent-list miss.** The count walks
  `records_number_key (tenant_id, entity_definition_id, number)`, filters
  with the ACL predicate, and stops at 10,001 matches. An agent sees about
  24,700 rows, so the count reads about 161,000 rows (28,278 buffers): 100 to
  244 ms single-run. The page walks `records_list_updated` and stops after 51
  rows (736 removed): 2 to 4 ms. A 10,000 cap already reads too much at this
  shape. PLAN-003's count-only fallback is the cap itself, so this is the
  case D25 says does not escalate. The request still misses 150 ms, so the
  CTO decides.
- **`data` filters scan the tenant.** `priority=1` walks the whole tenant in
  the count (398,792 rows removed). `site=s7` also misses in the page: 155,922
  rows removed before 51 matches, 268 ms single-run. Neither filter is
  indexable under RLS (P3-01). This is a `data`-filter miss, so per the
  CTO's conditions it reopens D25.
- **`state=new` and the admin `state in` list are load misses.** Single-run,
  both statements take 2.5 to 40 ms. Under 8 clients the agent `state=new`
  page reaches p95 186 ms. The admin list's p50 is 92.6 ms with a 482 ms p95
  tail. Both are fixed-column lists over 150 ms at p95.
- **Foreign-key sorts are outside the target** (option (b) on TICK-639).
  `sort=assignee` is reported but not judged. Its miss is count-only anyway.
- **The count alone misses.** With `total_capped`, the result is bounded at
  10,000, but the work isn't. To find 10,001 matches the count walks about
  161,000 rows, because the ACL predicate rejects about 94% of what it reads.
  On the agent lists this is the only miss: request p95 290 to 443 ms, of
  which the count is p95 237 to 345 ms and the page p95 18 to 36 ms.
- **Requester lists (TICK-644, known miss).** Against 150 ms per request:
  p95 699 to 713 ms. Against TICK-644's 50 ms database time for the page
  statement: p95 92 ms under load (single-run 7 to 17 ms). The count costs
  more than the page: p95 566 to 597 ms. A heavy requester has about 5,600
  rows, below the cap, so the count reads the whole tenant (394,374 rows
  removed). TICK-644 leaves the count unchanged (its scope: "the total count
  is unchanged"), so the `UNION ALL` rewrite fixes the page only. Even with
  TICK-644 the request misses on the count, the same count-only miss as the
  agent lists.

### Single read

| Scenario | p95, re-run | p95, repeat | Target |
| --- | --- | --- | --- |
| GET, agent | **122.4** | 47.6 | 50 |
| GET, admin | **99.7** | 48.8 | 50 |
| GET, small-tenant agent | **60.7** | **55.3** | 50 |

A GET is 13 statements. Each of them takes ≤ 0.3 ms in the database
(`records_pkey` and primary-key lookups). The request is round trips: p50
32 to 37 ms is about 13 times the floor. In the re-run the floor's p99 was
12 ms and every GET statement's p95 was 6 to 15 ms, so the host was noisy.
In the repeat, agent and admin meet and the small-tenant GET misses by
5 ms. The single-read result depends on the host's transport, not on the
query.

### Create (D-3-9 trigger, 50 ms)

| Scenario | Request p95, re-run / repeat | Counter wait p95 | Counter hold p50 / p95 | Hold: transport (est.) / database / handler, p50 |
| --- | --- | --- | --- | --- |
| Agent, 1 client | **83.4** / **98.6** | 2.8 / 3.2 | 21.7 / 36.9; 29.2 / 42.4 | 29.1 / 0.0 / 1.2; 18.1 / 9.5 / 1.2 |
| Agent, 8 clients | **470.6** / **647.8** | 413.4 / 580.8 | 22.7 / 30.2; 30.6 / 41.6 | 29.1 / 0.0 / 1.1; 18.1 / 11.1 / 1.2 |
| Admin, 8 clients | **1,017.4** / **858.4** | 727.6 / 771.7 | 29.1 / 87.5; 35.5 / 70.5 | 29.1 / 0.0 / 1.2; 18.1 / 15.8 / 1.3 |
| Small-tenant agent, 8 clients | **424.6** / **370.9** | 13.0 / 12.8 | 61.5 / 182.5; 78.7 / 154.9 | 29.1 / 30.2 / 1.8; 18.1 / 57.7 / 2.3 |

Where two values share a cell, the first is from the re-run and the second
from the repeat.

- **One create alone misses 50 ms.** It runs 29 statements in two
  transactions, the idempotency claim and then the write. With 1 client and
  no queue (wait p95 about 3 ms), p95 is 83 to 99 ms. The database part of
  the counter hold is 0 to 10 ms. The rest is round trips: 29 × the floor's
  p50 is 48 to 77 ms on this host before any database work.
- **On one tenant, creates queue on the counter.** With 8 clients, the
  `record_counters` upsert waits p95 413 to 772 ms for the row. Each holder
  keeps the row for 11 statements, through INSERT, write key, audit, outbox
  and COMMIT: p50 23 to 36 ms. Seven waiters × that hold is the wait. That is
  D-3-9's design ("the lock window is short") meeting 11 round trips.
- **The small-tenant creates don't queue** (wait p95 13 ms). Their hold has
  more database time, 30 to 58 ms at p50, from COMMIT: the 8 clients write
  to different tenants and share the WAL flush (`fsync = on`). The COMMIT
  p99 was 696 ms in the re-run.

A create miss reopens D-3-9 (the CTO's conditions).

## Verdict (first run)

**Single read: met.** p95 41.0 to 44.6 ms across the three GET scenarios.

**List: missed, and it is not a count-only miss.** Every list on the
400k-record tenant misses: p95 is 660 ms to 5,044 ms against 150 ms. The page
statement alone, without the `total` count, has p95 400 to 3,300 ms under
load, and 430 to 1,740 ms in a single unloaded `EXPLAIN ANALYZE`. Both
small-tenant lists meet the target (p95 86 and 90 ms).

Per PLAN-003, this miss triggers the D25 escalation. It is reported on
TICK-336 for the Technical Lead. This task does not tune indexes or relax the
target.

## Method

- **Path: through the HTTP handlers.** `TestRecordsACLPerf800k`
  (`internal/server/records_perf_integration_test.go`, build tags
  `integration perf`) sends `GET /api/v1/records/ticket` and
  `GET /api/v1/records/ticket/{id}` to the real `server.NewHandler`: strict
  server, authn, entity-type store, ACL policy read, `CompileRead`,
  `planList`, `store.List`, shaping and JSON. Each latency is the
  whole handler round trip.
- **The SQL is captured, never written by hand.** A pgx `QueryTracer` on the
  handler's own pool records every statement the request ran, with its
  arguments and duration. For each scenario, the first request's `SELECT`s
  are re-run as `EXPLAIN (ANALYZE, BUFFERS, SETTINGS)`, with the same text and
  arguments, as `nava_app` in the same tenant transaction.
- **Data: P3-01's shape** (TICK-339, `bench/records-list/02_seed.sql`) on the
  real migrated schema: the same RNG seed, distributions and tenant sizes.
  One tenant has 400,000 records and 399 tenants have 1,002 or 1,003 each, for
  800,000 in total. 15.0% are open, 50 groups per tenant, 3 groups per user,
  about 1 KB of `data`, and 15,815 user shares (2%). Every tenant's entity
  type is created with `POST /api/v1/entity-types`. The ACL is published with
  `acltest.Publish`. Records, principals, groups and shares are
  bulk-inserted. Every tenant's records belong to its one entity type, so
  each list runs over the whole tenant, as P3-01's did.
- **ACL** (the OR-combined shape of P3-01's query (d), in the real rule
  language):
  - `desk`: the agents group reads `assignment_group_id IN my_groups`.
  - `mine`: requester, assignee or shared.
  - `admins`: the admin group reads everything.
- **Actors:**
  - 200 large-tenant agents (P3-01's actors: agents group plus 3 assignment
    groups).
  - The large tenant's 8 heavy requesters (more than 1,000 records each).
  - The large tenant's admin.
  - One agent and the admin of each small tenant.
  - Page 20 is keyset page 20, the 951st row onwards, reached by following
    the handler's own cursors.
- **Load:** 8 concurrent clients, 20 s per scenario, after a warm-up of each
  request. Percentiles are nearest-rank, as in P3-01. All relations are
  `pg_prewarm`ed.

### Reproduce

```
make bench-acl-perf        # or: task bench:acl-perf
```

It needs Docker and Go and takes about 32 minutes on the host below. It is not
in CI. It starts `postgres:<pin>` with P3-01's server settings, runs the test
and writes the raw report to `bin/acl-perf-800k-raw.md`. The test fails when
any scenario misses its target. Env: `NAVA_PERF_SECONDS` (20),
`NAVA_PERF_CLIENTS` (8), `NAVA_PERF_OUT`, `NAVA_PERF_ONLY` (a regular
expression over scenario names, for example `^create`), `BENCH_KEEP=1`,
`NAVA_PERF_CHECKPOINT=1` (a `CHECKPOINT` after the seed).

Raw output for this run, with every statement, argument and plan:
`docs/quality/acl-perf-800k/run-2026-11-09.md`.

### Hardware and server

- Host: Windows 11, 12th Gen Intel Core i7-1260P, 39.7 GB RAM. Docker 29.9.2
  on WSL2 (kernel 5.15.167.4), with a VM of 16 CPUs and 19.4 GiB. This is the
  same host as P3-01.
- Load source: the Go test process on Windows, connected to PostgreSQL through
  Docker Desktop's port forward. There is no PgBouncer, unlike P3-01.
- `postgres:16.10`: `shared_buffers = 2GB`, `effective_cache_size = 6GB`,
  `max_wal_size = 8GB`, `track_io_timing = on`. Everything else is the
  default: `work_mem = 4MB`, `random_page_cost = 4`, `jit = on`,
  `max_parallel_workers_per_gather = 2`.
  - The raw file prints `max_parallel_workers_per_gather = 0` because it read
    the seeding session. That is fixed for later runs.
- `nava_app` runs with `plan_cache_mode = force_custom_plan` (db-roles) and
  the pool in `QueryExecModeExec`, so every plan is a custom plan.
- Noise: the idle `nava` and acceptance stacks were running on the same
  Docker VM. Each statement round trip costs about 3 to 5 ms at p50 under
  load: `begin` and `set_config` alone take that long. A request runs 10 (list)
  or 12 (GET) statements, so roughly 35 to 50 ms of each request is transport,
  not query. That makes the GET numbers pessimistic. It does not explain the
  list misses, which are seconds.

## Results

p50 / **p95** / p99 in ms, per request through the handler, 8 clients.

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Agent list by group, default sort, page 1 | large | 48 | 3,291.8 | **5,013.4** | 5,200.2 | 150 | **miss** |
| Agent list by group, default sort, page 20 | large | 48 | 3,274.5 | **5,044.4** | 6,180.0 | 150 | **miss** |
| Requester list (heavy requesters), page 1 | large | 56 | 2,981.7 | **3,364.1** | 3,577.6 | 150 | **miss** |
| Requester list (heavy requesters), page 20 | large | 57 | 2,985.7 | **3,282.5** | 3,374.1 | 150 | **miss** |
| Admin list, default sort, page 1 | large | 71 | 2,428.8 | **2,645.7** | 2,698.8 | 150 | **miss** |
| Admin list, default sort, page 20 | large | 63 | 2,687.7 | **3,318.0** | 3,806.9 | 150 | **miss** |
| Agent list, `filter[state]=new` | large | 264 | 590.4 | **747.4** | 823.8 | 150 | **miss** |
| Agent list, `filter[priority]=1` (data) | large | 51 | 3,314.6 | **3,787.3** | 4,545.6 | 150 | **miss** |
| Agent list, `sort=-updated_at` | large | 56 | 3,071.4 | **3,592.8** | 3,697.5 | 150 | **miss** |
| Agent list, `sort=assignee` | large | 70 | 2,431.3 | **2,729.5** | 3,008.8 | 150 | **miss** |
| Admin list, `filter[state][in]=new,in_progress`, `sort=-updated_at` | large | 295 | 545.0 | **660.2** | 702.5 | 150 | **miss** |
| Agent list, page 1 | small | 3,073 | 46.9 | **86.5** | 107.9 | 150 | met |
| Admin list, page 1 | small | 2,739 | 53.0 | **90.1** | 123.5 | 150 | met |
| Single GET, agent | large | 5,223 | 29.0 | **41.0** | 55.4 | 50 | met |
| Single GET, admin | large | 4,982 | 30.9 | **43.4** | 56.5 | 50 | met |
| Single GET, agent | small | 4,847 | 31.3 | **44.6** | 59.2 | 50 | met |

Every list returns `total`. The raw report splits each list into its two
heavy statements. Statement 8 is the `count(*)` for `total`, and statement 9
is the page:

| Large-tenant scenario | `count(*)` p95 | Page p95 | Single-run `EXPLAIN ANALYZE` page (of which JIT) |
| --- | --- | --- | --- |
| Agent, default sort, page 1 | 2,602 | 2,376 | 550 ms (377) |
| Agent, default sort, page 20 | 1,786 | 3,297 | 561 ms (381) |
| Requester, page 1 | 1,464 | 2,054 | 530 ms (405) |
| Admin, default sort, page 1 | 581 | 2,017 | 774 ms (23) |
| Admin, default sort, page 20 | 789 | 2,561 | 1,740 ms (30) |
| Agent, `filter[state]=new` | 302 | 400 | 134 ms (55) |

The page statement misses on its own, so a capped count cannot rescue it.
This is a full D25 trigger, not the count-only fallback.

## Why: the plans

The compiled ACL predicate itself is not the cost. The share `EXISTS` runs
once per statement, as a hashed subplan (`SubPlan 2`, `loops=1`, 13
buffers), and the small-tenant lists over the same predicate take 1 to 8 ms
in the database. The cost is in the shape of the list SQL around it.

Agent list, default sort, page 1, page statement (excerpt):

```
Limit (actual time=546.845..546.876 rows=51 loops=1)
  Buffers: shared hit=70049
  ->  Sort  Sort Key: r.created_at DESC NULLS LAST, r.id   Sort Method: top-N heapsort
        ->  Hash Left Join (groups g)
              ->  Nested Loop
                    ->  Index Scan using records_number_key on records r (rows=24666)
                          Index Cond: ((tenant_id = '8b89…') AND (entity_definition_id = 'd028…'))
                          Filter: (assignment_group_id = ANY (…) OR requester_id = … OR assignee_id = … OR (hashed SubPlan 2))
                          Rows Removed by Filter: 375334
JIT: … Total 377.234 ms
Execution Time: 550.007 ms
```

Admin list, default sort, page 20: `Seq Scan on records r … rows=399050`, then
a sort, at 1,740 ms.

1. **No list reads in index order. Each one reads the whole tenant and
   sorts.** The builder orders by `r.<col> <dir> NULLS LAST, r.id ASC`.
   P3-01's walk used `records_tenant_updated (tenant_id, updated_at DESC, id
   DESC)` with `ORDER BY updated_at DESC, id DESC`. A `DESC` index is
   `NULLS FIRST`, and the tiebreaker here runs `ASC`, so the index's order is
   never the requested one, even for `sort=-updated_at`. The planner instead
   walks `records_number_key (tenant_id, entity_definition_id, number)` to
   apply `entity_definition_id = $2`, reads 400k rows (70k buffers) and
   top-N sorts.
2. **The default sort is `created_at`, which no index covers.** P3-01
   measured and indexed `updated_at`. `planList`'s `defaultSortField` is
   `created_at`.
3. **The keyset predicate is not an index condition.** Page 20 renders
   `(col < $v OR col IS NULL OR (col = $v AND r.id > $id))`, not a row
   comparison `(col, id) < ($v, $id)`. Page 20 therefore costs as much as page
   1 or more: 1,740 ms for the admin.
4. **`total` is an exact, uncapped `count(*)`.** It reads every matching row:
   24,666 for an agent and 400,000 for the admin. PLAN-003's fallback for a
   count miss is a capped count. The builder has none.
5. **JIT compiles every large-tenant list statement.** Plans that read the
   whole tenant cost about 1.35M, far above `jit_above_cost` (100k). JIT then
   spends 200 to 430 ms per statement. That is two thirds of a single-run
   agent page. With JIT off, the remaining scan and sort would take 150 to
   190 ms per statement, two statements per request, and would still miss
   150 ms. That figure is **derived** (execution time minus JIT time), not
   measured with `jit = off`. JIT is a multiplier on cause 1, not a cause of
   its own.
6. **Concurrency multiplies the single-run cost.** 8 clients each reading
   400k rows twice per request is CPU-bound: 550 ms single-run becomes
   3 to 5 s at p95.

The JSONB filter (`filter[priority]=1`) cannot use an index under RLS, as
P3-01 found. It reads the whole tenant anyway, like the unfiltered list.

## Re-run (CTO's D25 decision)

The CTO decided D25 on TICK-336: no D25 work now, and the targets stay.
P3-36 runs again on this harness once TICK-639 and TICK-640 have merged. The
re-run covers every large-tenant row and both `data` selectivities, with the
targets unchanged. A miss goes back on TICK-336 with its plans.

For that re-run the harness has scenarios that are not in the 2026-11-09
numbers above:

- **Agent list, `filter[site]=s7`, the rare `data` value.** `site` is one of
  200 values, each 0.5% of rows, so it matches less than 1% of an agent's
  visible rows. It sits next to the existing `priority=1` row (5%). The seed
  derives `site` from `category`'s random draw, so every other column and
  the RNG stream are unchanged. The raw report's Data section prints the
  large tenant's `site = s7` count.
- **Create, end to end, against the D-3-9 trigger (p95 ≤ 50 ms)**, added at
  the Technical Lead's request. TICK-640 (PR #539) adds six indexes and
  measured only the INSERT statement: its p95 went from 11–22 ms to
  20–29 ms. These scenarios measure the whole `POST /api/v1/records/ticket`
  instead: authn, idempotency, validation, the ACL write check, numbering,
  the INSERT, the audit row and COMMIT. They run as an agent (into one of
  its own groups) and as the admin on the large tenant, and as a
  small-tenant agent. The body is P3-01's record shape, about 1 KB of
  `data`. Every request carries a fresh `Idempotency-Key`. The creates run
  after every read scenario, so the rows they add never reach a read. The
  raw report lists each create's statements and per-statement timings; its
  writes are not EXPLAINed, because `EXPLAIN ANALYZE` would run them again.
  8 clients on one tenant all take that tenant's `record_counters` row
  (D-3-9), so the large-tenant create p95 includes that lock wait.
- **Create, agent (1 client)**, next to the 8-client row. With one client
  no other create queues on the counter row, so this row is the create path
  alone. D-3-9's trigger states no concurrency; the two rows together show
  whether a miss comes from the path or from the counter queue.
- **Counter hold**, a table in the raw report for every create scenario. Each
  traced statement carries its start offset in the request. For each create,
  the window runs from the end of the `record_counters` upsert (the row is
  locked) to the end of the `COMMIT` that releases it. The table prints:
  - the upsert itself (the wait for the row plus its execution);
  - the hold, and how many statements run inside it;
  - the hold split into transport (statements in the window times the
    transport floor's p50, an estimate), database (the rest of their
    statement time) and handler time (between statements).

- **Transport floor.** Before the scenarios, 8 clients run `SELECT 1` 200
  times each on the handler's pool, and the report prints its p50/p95/p99.
  A create runs 29 statements and a list 10, so a request's floor is about
  its statement count times that p50. The per-statement tables are read
  against it.

A create-only smoke on 2026-11-09 (`NAVA_PERF_ONLY='^create'`, 5 s per
scenario, story branch before TICK-639/TICK-640, host shared with other
work) checked the scenario, not the target: every request returned 201.
Two things from it are for the re-run to confirm or clear, not results:

- The `record_counters` upsert waited p50 409–565 ms and p95 about 3 s on
  the large tenant. 8 creates on one tenant and type queue for one counter
  row, held from the upsert to COMMIT (D-3-9).
- Every round trip, `SELECT current_setting(...)` included, took about 13 ms
  at p50, against 3.4 ms in the 2026-11-09 list run. At 29 statements that
  is about 400 ms per create before any database work. The transport floor
  above exists to separate this from the database.

Known miss in the re-run: requester lists until **TICK-644** (P3-34c, the
requester list as a `UNION ALL` of index-ordered branches) lands. TICK-640
left them at 38–166 ms of database time. The re-run reports them against
their unchanged target with a link to TICK-644, not as a new finding.

## What this does not cover

- PATCH. Create is measured from the re-run on (above); PATCH is not.
- Through PgBouncer. Handler to PostgreSQL is direct, through Docker's port
  forward. The pooler adds queueing (P3-01) and would only make the lists
  slower.
- Filters on more than one data property, `in` filters over data, and sorts
  on `state` or `requester_id`. They share the plan shape above (whole tenant,
  then sort) and would not change the verdict.
- A CI perf job. The run takes about 32 minutes and holds about 4 GB of
  Docker memory, so it is a documented run (`make bench-acl-perf`), as P3-01
  was.

## Hand-offs

- **Technical Lead:** D25 escalation to the CTO (PLAN-003). List p95 misses
  150 ms on every large-tenant scenario, and the page statement misses on its
  own. The numbers and plans are in this document and in
  `docs/quality/acl-perf-800k/run-2026-11-09.md`.
- **Owner of P3-34 (`records/store.List`, `planList`):** causes 1 to 4 are
  drift between the list SQL and the query shape P3-01 measured and P3-09
  indexed. The CTO's D25 decision should settle the fix before anyone changes
  it. This task does not.
- **Principal Software Engineer:** reviewer of this measurement.
