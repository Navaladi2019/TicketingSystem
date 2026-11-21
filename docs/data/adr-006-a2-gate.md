# ADR-006-A2 A2-8 (2), (2a), (3a): budget check and S-A2-cancel (TICK-1048)

Scope: PLAN-007-A2 T-A2-4. The run used the T-A2-2 code (TICK-1046, PR #827,
head `bc81fe34`) with `CountBudget` = 200 ms in force. Harness:
`internal/server/records_a2_gate_perf_integration_test.go` (`TestA2Gate`, tags
`integration perf`, not in CI). Runner: `scratch/tick-1048-a2-gate/run.sh`.
Raw output: `docs/data/adr-006-a2-gate/run-N/`.

## Verdict: PASS (runs 2 and 4, 2026-11-21 22:33–22:49Z, the Technical Lead's bench window)

| Gate | Rule | Result | Status |
| --- | --- | --- | --- |
| (2a) host rule | floor p50 ≤ 3 ms and no-filter control p95 ≤ 80.3 ms, before and after | floor 0.54 / 1.11 ms; control p95 **47.4 / 44.4 ms** (n = 3,282 / 3,176) | **Pass**, attempt 1 |
| (2) non-count p95, band (ii) | ≤ 94 ms | **50.7 ms** (n = 2,787, p50 42.1, p99 57.8); agent-1 alone 49.2 ms | **PASS, final** (passes by 46%). A2-2's formula is not rerun; `CountBudget` stays 200 ms |
| (3a) S-A2-cancel | count phase ≤ 215 ms at p99; every answer 200 + `unknown`; a further statement; no temp growth | p99 **205.5 ms (+5.5)** single-client, **211.0 ms (+11.0)** at 8 clients; 300 of 300 answered 200 + `unknown` + `total: null` + 50 items; every request ran `commit` after `ROLLBACK TO`; temp_files 22 / temp_bytes 239,756,336 before, after and 60 s after | **PASS** |

The host was idle for the window. `docker ps` showed only the bench database,
`buildx_buildkit_nava-ci0` and `multica-*`; other containers used a median
0.65% and at most 14.6% of one core (`run-2/contention.txt`). Run 2 measured
(2a) and (2) and then stopped: on the idle host no fixture count exceeded the
budget (slowest 70.2 ms, `run-2/raw.md`), so (3a) had no slow plan. Run 3
added the anti-correlated field `pw` (115.8 ms, still under). Run 4 made `pw`
multi-valued (four values per invisible record) and scored (3a)
(`NAVA_PERF_GATE_ATTEMPTS=0`, floor 1.08 ms before the spike).

### Run 2: band (ii), count skipped, 8 clients, 15 s

| | p50 | p95 | p99 | n |
| --- | --- | --- | --- | --- |
| control before (agent list, no filter) | 38.0 | 47.4 | | 3,282 |
| **band (ii) `filter[kp]=3`, count skipped (agents)** | 42.1 | **50.7** | 57.8 | 2,787 |
| band (ii), count skipped (agent-1 alone) | | 49.2 | | 2,875 |
| band (ii) with the budgeted count (reported only) | 64.8 | 75.9 | | 1,824 |
| control after | 36.7 | 44.4 | | 3,176 |

P3-36's control read 73 ms when it was scored; the same handler reads 44–47 ms
here (QA's TICK-946 baseline on the same idle host: 59.12 ms on its own
scenario). The non-count request runs 7 statements; its page statement took
11.4 ms in the captured request, the other six 1.7–1.9 ms each
(`run-2/raw.md`, plans in `run-2/plans.md`). Raw samples:
`run-2/band-samples.txt`.

### Run 4: S-A2-cancel

Cell: **a50, `filter[pw]=1`**. `pw` is 1 on four side rows (`ord` 0–3) of
each of the 176k records that neither a6 nor a50 sees (704k rows), and on 100
records a50 sees and 100 a6 sees. So the first page is full (50 of 100) and the
count must find all 100, past the 1001 cap's early stop. Unbudgeted plain
count: 442.8 / 452.6 / 462.1 ms (EXPLAIN median 472.4). Visibility-map state
`none` for `pw`'s pages. Plans: `run-4/plans.md`.

| Run | n | 200 + `unknown` + `total: null` + 50 items | `commit` after `ROLLBACK TO` | count phase p50 / p95 / p99 / max ms | p99 overshoot |
| --- | --- | --- | --- | --- | --- |
| 1 client | 100 | 100 | 100 | 203.8 / 205.0 / 205.5 / 205.6 | **+5.5 ms** |
| 8 clients | 200 | 200 | 200 | 207.7 / 209.2 / 211.0 / 212.6 | **+11.0 ms** |

| Run | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |
| --- | --- | --- | --- |
| 1 client | 201.4 / 202.6 / 202.6 | +2.6 ms | 2.4 / 3.6 |
| 8 clients | 202.7 / 205.0 / 207.0 | +5.0 ms | 4.9 / 7.7 |

The cancel itself costs 2.6–5.0 ms at p99. Most of the remainder is the two
round trips A2-2 already budgets (6 ms). At 8 clients the round trips reach
7.7 ms at p99, so the phase lands at +11.0, inside the 15 ms margin. **No extra
A2-2 term is needed.** The cell's client latency is 615 ms p50 single-client
and 1,301 ms at 8 clients: that is the anti-correlated page walk (A2-9, the
R-a track), not the count, which is capped at the budget.

## Run 1 (2026-11-21 12:45Z, contended host): inconclusive, kept as history

| Gate | Rule | Result | Status |
| --- | --- | --- | --- |
| (2a) host rule | floor p50 ≤ 3 ms and no-filter control p95 ≤ 80.3 ms, before and after | **Failed on both attempts**: control p95 172.9 / 408.0 ms (attempt 1) and 242.5 / 168.7 ms (attempt 2); floor 1.90 / 14.13 and 5.39 / 4.47 ms | **Inconclusive.** Stopped after two attempts (Technical Lead's rule). |
| (2) non-count p95, band (ii) | ≤ 94 ms | 198.6 ms (attempt 1), 139.5 ms (attempt 2), on a host that failed its gates | **Inconclusive**: not used in A2-2's formula. Rerun on an idle host. |
| (3a) S-A2-cancel | count phase ≤ 215 ms at p99; every answer 200 + `unknown`; a further statement; no temp growth | p99 299.3 ms single-client (+99.3), 655.3 ms at 8 clients (+455.3); 67 of 100 single-client answers `unknown`, 33 `capped`; 200 of 200 at 8 clients `unknown`; every request ran `commit` after `ROLLBACK TO`; temp_files 17 / temp_bytes 200,074,288 before, after and 60 s after | **Measured FAIL on a contended host.** The cell is also too close to the budget (see below). Rerun on an idle host. |

These numbers are not a ruling on the 94 ms or on the 15 ms overshoot term.
The host was busy for the whole run. The owner's `nava` stack (12
containers) and another agent's integration databases (`nava-it-pg-*`,
`nava-it-pgb-*`) were up. Their `docker stats` CPU summed to a median of 64% of
one core, a p95 of 225% and a maximum of 584% (`run-1/contention.txt`, 72
samples). The transport floor was 8.11 ms just before the spike. P3-36's
control read 72.5–73.5 ms on an idle host and 90.7–215.9 ms p50 here.

## Setup

- Host: Windows 11, 12th Gen Intel Core i7-1260P, Docker 29.9.2 on WSL2
  (kernel 5.15.167.4), VM with 16 CPUs and 24.5 GiB. A fresh `postgres:16.10`
  container with P3-01's settings (`shared_buffers = 2GB`,
  `effective_cache_size = 6GB`, `fsync = on`, `jit = off`). `nava_app` runs
  with `plan_cache_mode = force_custom_plan`. No PgBouncer: the handler's own
  pool, as in P3-36. The server runs in-process, so `WithSkipCount` works
  (D-7A2-7). Latency is client-side, at the harness.
- Data: P3-36's seed (800,000 records, one tenant with 400,000 tickets and 399
  tenants with about 1,000 each). It is used so that P3-36's control reads on
  P3-36's data. The S21 kit's shape sits on top: a num field `kp`, flagged
  `ready`, at the kit's priority distribution. It came out as 3,976 / 4,081 /
  191,830 / 200,113 records for values 1 / 2 / 3 / 4 (1 / 1 / 48 / 50%).
- ACL: the agent is P3-36's agent (the agents group plus 3 of 50 assignment
  groups, about 6% of the type; agent-1 sees 11,551 records with `kp = 3`). The
  kit's Alice sees about 5% through one group. That is the one deviation from
  the kit's shape, and it is a little stricter.
- (2) cell: `GET /api/v1/records/ticket?filter[kp]=3`, 8 clients for 15 s,
  round-robin over P3-36's 200 agents (also run as agent-1 alone), with the
  count skipped by `WithSkipCount`. Each attempt first takes the floor (`SELECT
  1`, 8 × 200) and the control (agent list with no filter, 8 clients, 15 s),
  then takes both again after the cell.

```mermaid
erDiagram
    records ||--o{ rfv_num : "kp (kit priority), pk (PR #812)"
    records ||--o{ rfv_text : "pt"
    records ||--o{ rfv_bool : "pb"
    record_index_fields ||--o{ rfv_num : "kp, pk: ready"
    records {
        uuid tenant_id PK
        uuid id PK
        uuid assignment_group_id "ACL: desk rule, 3 of 50 groups (6%)"
        jsonb data "kp 1/1/48/50%"
    }
    record_index_fields {
        uuid tenant_id PK
        uuid id PK
        text field_key "kp, pk, pt, pb"
        text state "ready"
    }
    rfv_num {
        uuid tenant_id PK
        uuid field_id PK
        uuid record_id PK
        float8 value "kp 400k rows (vacuumed), pk 400k rows (not vacuumed)"
    }
```

### Run 1: (2) band (ii), count skipped: 8 clients, 15 s, ms

| Attempt | Host before (floor p50 / control p95) | Host after | Non-count p50 / p95 / p99 (agents) | p95, agent-1 alone | With the budgeted count, p50 / p95 (reported only) |
| --- | --- | --- | --- | --- | --- |
| 1 | 1.90 / **172.9** | **14.13** / **408.0** | 107.1 / 198.6 / 265.2 (n = 1,014) | 210.6 | 225.9 / 665.5 |
| 2 | **5.39** / **242.5** | **4.47** / **168.7** | 80.3 / 139.5 / 167.0 (n = 1,370) | 144.1 | 199.5 / 325.3 |

Bold marks a failed gate. Raw samples: `run-1/band-samples.txt`. With the
count skipped, a request runs 7 statements and no count: BEGIN + tenant, type,
definition, ACL, registry, page, commit (`run-1/raw.md`). The page statement
took 17.2 ms server-side in the captured request. Its plan is in
`run-1/plans.md`.

### Run 1: (3a) S-A2-cancel

Fixture: PR #812's (A2-11). a6 sees 24,055 records (6%) and a50 sees 200,379
(50%). The side rows `pk`, `pt`, `pb` were not vacuumed and autovacuum was
off. `pk` uses `rfv_num`, whose `kp` half was already vacuumed, so the table
reads 50% all-visible. `pk`'s own pages were never vacuumed. Each count was
timed unbudgeted first (EXPLAIN (ANALYZE, BUFFERS), 3 runs, median):

| Candidate | Unbudgeted count, median ms |
| --- | --- |
| a6, pk = 3 (broad, 6% ACL) | **591.5** (spike cell) |
| a6, pk = 3 + pt = x + pb = true | 501.1 |
| a6, pk = 20 (partly correlated) | 86.7 |
| a50, pk = 3 | 34.4 |
| a6, pk = 10 (anti-correlated, 0 visible) | no count: the first page is empty (A2-4 shortcut) |

The pk = 10 cell named in the task never reaches a count on the T-A2-2 code.
The shortcut answers it exactly with no count statement. The spike therefore
used the slowest count that does run. Its plan is a Parallel Hash Semi Join
that builds a hash of `rfv_num` (78k rows per worker) and stops at the
`LIMIT 1001` cap (`run-1/plans.md`).

| Run | n | 200 + `unknown` + `total: null` + 50 items | `commit` after `ROLLBACK TO` | count phase p50 / p95 / p99 / max ms | p99 overshoot | client p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1 client | 100 | 67 (33 answered `capped`: their count finished inside the budget, p50 165.9 ms, max 200.9 ms) | 100 | 234.0 / 291.9 / 299.3 / 465.1 | **+99.3** | 289.5 / 470.9 |
| 8 clients | 200 | 200 | 200 | 295.7 / 469.0 / 655.3 / 1,192.6 | **+455.3** | 502.7 / 835.2 |

The count phase splits into the count statement and the two round trips
around it. These figures come from `run-1/spike-samples.csv`, and cover only
the `unknown` answers:

| Run | count statement p50 / p99 / max ms | two round trips p50 / p99 ms |
| --- | --- | --- |
| 1 client | 236.9 / 412.8 / 412.8 | 8.9 / 52.3 |
| 8 clients | 260.2 / 511.0 / 1,158.0 | 31.8 / 196.4 |

What this run shows, and what it doesn't:

- **The failure path held on every request:** 300 of 300 answered 200 with
  50 items. No response was a 5xx. Each budget overrun gave `total: null` /
  `unknown`. The transaction committed after `ROLLBACK TO`. Temp files did not
  grow (0 bytes across the run and the 60 s after).
- **The overshoot is not measured on a valid host.** The two round trips alone
  took 8.9 ms at the p50 single-client. A2-2 assumes 3 ms each. The count
  statement ran to 237 ms at the p50 against a 200 ms timeout. That 37 ms
  mixes the cancel latency with the scheduling delay of a busy VM. These
  figures can't be used as the extra A2-2 term.
- **The cell is too close to the budget.** Without EXPLAIN's per-row timing,
  this count takes about 166–201 ms. One single-client request in three
  finished inside the budget. On an idle host most would, and "every answer
  `unknown`" would fail for the wrong reason. The harness now times each
  candidate plainly, adds a heavier candidate (`pk < 5`, 78% of the side
  rows) and picks on the plain median. It also reports the split shown above.

## Rerun

On an idle host (no `nava-*`, `nava-it-*` or bench containers, no other agent runs):

```sh
sh scratch/tick-1048-a2-gate/run.sh                               # (2), (2a) and (3a), about 15 min idle
NAVA_PERF_GATE_ATTEMPTS=0 sh scratch/tick-1048-a2-gate/run.sh     # (3a) only, about 6 min
```

The spike picks the candidate with the slowest plain count. Since run 4 that
is a50 `pw = 1`, the multi-valued anti-correlated fixture. `NAVA_PERF_OUT`
sets the report path; samples, plans, host and contention files are written
beside it.
