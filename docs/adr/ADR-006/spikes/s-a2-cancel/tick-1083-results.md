<!-- Archived for ADR-006-E2 from docs/data/adr-006-a2-gate.md (PR #851, branch head 037e4438), TICK-1083 section. Raw output: run-5/ and run-6/ beside this file. -->

## TICK-1083 (T-A2-4b): spill-then-cancel and parallel-plan cells

Scope: the Technical Architect's two non-blocking findings on TICK-1082.
S-A2-cancel's "no extra A2-2 term" claim was checked for two more cases: a
cancel that has already spilled to temp files, and a cancel that waits for
parallel workers to shut down. Same harness and seed as run 4, on a fresh
`postgres:16.10` with `NAVA_PERF_GATE_ATTEMPTS=0`. Raw output: `run-5/` and
`run-6/`.

### Run 6 (2026-11-22 10:09–10:23Z): spill FAIL confirmed; parallel FAIL by a wide margin, but a host gate flagged it

Fresh `postgres:16.10` (commit `7de2a4a8`), i7-1260P, WSL2 Docker VM with 16
vCPUs and 24.5 GiB, P3-36's 800k-record seed (seeded in 2m4s), `CountBudget`
200 ms. `NAVA_PERF_GATE_ATTEMPTS=0 NAVA_PERF_SPIKE_CELLS=spill,parallel`. Only
`multica-*` containers ran beside the bench database (`run-6/host.txt`). The
VM load average peaked at 6.5 of 16 CPUs, from the spike's own workers
(`run-6/contention.txt`).

| Cell | Pick (first `UNION ALL` branch) | count phase p99, 1 client (n = 100) / 8 clients (n = 200) | 200 + `unknown` + `commit` after `ROLLBACK TO` | `pgsql_tmp` after the run / 60 s later | Host gates before / after | Status |
| --- | --- | --- | --- | --- | --- | --- |
| `spill` | a50 `pw = 1 and pk < 5`, `work_mem = 64kB`, no parallel, nested loops and merge joins off; plain median 1,318.1 ms; hash join, 512 batches | **219.0 (+19.0) / 227.8 (+27.8) ms** | 300 / 300 | empty / empty (`pg_ls_tmpdir()` 0 files, `base/pgsql_tmp` 0 entries, 0 parallel workers). It spilled: 78,537 temp files, 2.59 GB during the spike | pass (floor 0.63, control p95 72.1) / pass (1.65, 79.1) | **FAIL**, confirms run 5 |
| `parallel` | a50 `pw = 1`, nested loops off; plain median 578.1 ms; `Gather` with 2 workers launched, running 325–407 ms under EXPLAIN, over a Parallel Hash Semi Join whose shared hash spills (8 batches) | **239.5 (+39.5) / 271.2 (+71.2) ms** | 300 / 300 | empty / empty, 0 parallel workers. 7,352 temp files, 7.95 GB during the spike | **fail** (floor 2.21, control p95 80.6 against 80.3) / **fail** (1.26, 84.3) | **FAIL, gate-flagged**: the harness scores it INCONCLUSIVE |

| Cell | Run | count phase p50 / p95 / p99 / max ms | count statement p50 / p99 / max ms | two round trips p50 / p99 ms | harness `SET LOCAL` p50 / p99 ms (left out) |
| --- | --- | --- | --- | --- | --- |
| spill | 1 client | 210.5 / 214.9 / 219.0 / 222.9 | 207.6 / 214.1 / 214.3 | 2.8 / 8.1 | 2.27 / 3.74 (both runs) |
| spill | 8 clients | 220.3 / 226.2 / 227.8 / 229.1 | 214.6 / 222.6 / 224.0 | 5.4 / 9.0 | |
| parallel | 1 client | 232.6 / 238.7 / 239.5 / 261.6 | 229.1 / 236.6 / 258.1 | 3.3 / 5.1 | 2.36 / 9.41 (both runs) |
| parallel | 8 clients | 234.6 / 252.5 / 271.2 / 274.2 | 226.9 / 260.7 / 267.5 | 6.9 / 24.7 | |

What run 6 shows:

- **Spill, two runs, both FAIL.** Run 5 gave +16.1 / +45.9 ms, run 6 gives
  +19.0 / +27.8 ms. Run 6's host gates passed on both sides of the cell. The
  extra cost is systematic, as in run 5: the single-client count statement's
  p50 is 207.6 ms (run 5: 208.2 ms; `base`, run 5: 201.6 ms). The 8-client
  tail varies between runs (+27.8 against +45.9 ms).
- **Parallel: the claim fails here too, and by more.** The single-client
  count statement's **p50** is 229.1 ms, 29 ms over the budget on every
  cancel, not only in the tail. The pick's `Gather` runs while its workers
  are still building a shared Parallel Hash that spills, so the cancel
  waits for two workers to stop and drop the shared temp files. That is
  parallel shutdown and a spill together. The host gates failed narrowly:
  control p95 80.6 ms before (0.4% over 80.3) and 84.3 ms after (5% over).
  A 0.3–4 ms drift in the no-filter control can't produce a 29 ms shift at
  the median, so I read the FAIL as real. The 8-client figure (+71.2 ms) is
  the less certain of the two.
- **The cancel path holds in both cells.** Every request answered 200 +
  `unknown` + `total: null` + 50 items and ran `commit` after `ROLLBACK TO`.
  No `pgsql_tmp` file or parallel worker was left after the run or 60 s
  later.

**For the Solution Architect (the ticket's fail rule): the extra A2-2 term.**
The term is the count phase's p99 overshoot at 8 clients, less the 6 ms of
round trips A2-2 already budgets. A single count either spills or runs in
parallel (or both, as the parallel pick does), so the terms don't add up:
the larger one applies.

| Basis | 8-client p99 overshoot | Extra term | A2-2: 350 − 94 − 6 − 35 − term | `CountBudget` (rounded down to 10 ms) |
| --- | --- | --- | --- | --- |
| spill, run 6 | +27.8 ms | +22 ms | 193 | 190 ms |
| spill, worst of runs 5 and 6 | +45.9 ms | +40 ms | 175 | 170 ms |
| parallel, run 6 (gate-flagged) | +71.2 ms | +66 ms | 149 | 140 ms |

The spill shape needs settings forced on this seed: `work_mem = 64kB`, no
parallel, and nested loops and merge joins off. At 64 kB its hash has 512
batches, far more than a 4 MB spill of the same hash, so the spill rows are
an upper bound for that shape. The parallel shape needs no forcing. Under
the server's settings, run 6's `a6, three broad fields` count (plain median
208.6 ms) has a launched `Gather` in branch 1 that is still running at
1.25 × the budget (`run-6/raw.md`, cell parallel). The measured pick forced
only `enable_nestloop = off`, and its shared Parallel Hash spilled at the
server's `work_mem` (4 MB). So the parallel row is the one that applies, and
the release trigger's "production count with a `Gather`" holds on the bench
seed. Before the 140 ms row is adopted, a parallel-only rerun with clean
gates (`NAVA_PERF_SPIKE_CELLS=parallel`, about 10 minutes) settles the
8-client figure.

### Run 5 verdict: spill FAIL (one run, measured); parallel INCONCLUSIVE (not validly measured)

| Cell | Rule | Result | Status |
| --- | --- | --- | --- |
| `base` (run 4's cell, rerun as the control) | count phase p99 ≤ 215 ms | p99 **206.4 ms (+6.4)** single-client, **213.6 ms (+13.6)** at 8 clients; 300 of 300 answered 200 + `unknown` + `total: null` + 50 items, then `commit` | **PASS**: it reproduces run 4 (+5.5 / +11.0) |
| `spill`: hash join, `work_mem = 64kB`, spilling when the budget expires | count phase p99 ≤ 215 ms; 200 + `unknown`; a further statement; no `pgsql_tmp` files after the run or 60 s later | p99 **216.1 ms (+16.1)** single-client, **245.9 ms (+45.9)** at 8 clients. 300 of 300 answered 200 + `unknown` + `total: null` + 50 items, then `commit`. `pg_ls_tmpdir()` and `base/pgsql_tmp` were empty after the run and 60 s later. No parallel worker was left. The spill happened: 75,207 temp files and 2.08 GB were written during the spike | **FAIL** on the overshoot. The cancel path and the leak check pass |
| `parallel`: a `Gather` running when the budget expires | as (3a) | p99 +26.8 ms single-client, +195.6 ms at 8 clients, but **the cell did not test the claim** (see below) | **INCONCLUSIVE**: rerun |

**For the Solution Architect (the ticket's fail rule):** the spill cell's
overshoot is an extra A2-2 term. The count phase's p99 overshoot at 8 clients
is 45.9 ms. A2-2 already budgets the two round trips in it (6 ms), so the extra
term is **40 ms** (39.9 rounded up). A2-2 rerun with it: 350 − 94 − 6 − 35 −
40 = 175, rounded down to 10 ms, gives **`CountBudget` = 170 ms**. Two caveats
belong with that number:

- **One run.** The cell's host was clean by the VM load average. P3-36's gate
  before the spike passed (floor 0.60 ms, control p95 60.3 ms), and so did the
  floor just before the cell (1.15 ms). The gate after the spike failed, but
  that was a `buildx` build that started at 04:38:55, five minutes after the
  spill cell had finished. A confirmation run (run 6) was started at 05:17 and
  stopped at 05:27: the S21 fresh stack came back up and was busy, and its
  acceptance measurement must not share the host with a bench.
- **The spill needs forced plans on this seed.** Under the server's settings
  no count both spills and runs past 200 ms here. The spill cell needs
  `work_mem = 64kB`, no parallel, and nested loops and merge joins off before
  the planner picks a hash join over the side table. A production count
  spills only if its hash exceeds `work_mem` (4 MB). The release trigger
  (any production count plan with a hash spill in the R-a track) decides
  whether 170 ms is needed.

**Why a spilled cancel is slower** (an explanation, not measured phase by
phase): the first branch hashes 702k `rfv_num` rows into 512 batches. In
`run-5/plans.md` the build runs from 67 to 290 ms under EXPLAIN, so the
cancel lands mid-build with the batch files open. The abort then closes and
unlinks them: on average 250 files and 6.9 MB per request. The cost shows at
the median, not only in the tail. The single-client count statement takes
208.2 ms at p50, against 201.6 ms in `base`. That is a systematic +6.6 ms on
every cancel. At 8 clients, eight such spills write through WSL2's virtual
disk at once, and the p99 of the statement alone reaches 232.1 ms.

| Cell | Run | n | count phase p50 / p95 / p99 / max ms | p99 overshoot | count statement p50 / p99 / max ms | two round trips p50 / p99 ms |
| --- | --- | --- | --- | --- | --- | --- |
| base | 1 client | 100 | 204.5 / 205.7 / 206.4 / 206.9 | +6.4 | 201.6 / 203.0 / 204.7 | 2.7 / 4.7 |
| base | 8 clients | 200 | 208.6 / 211.3 / 213.6 / 214.4 | +13.6 | 203.1 / 206.4 / 207.7 | 5.3 / 8.5 |
| spill | 1 client | 100 | 211.0 / 214.6 / 216.1 / 217.3 | **+16.1** | 208.2 / 213.4 / 215.6 | 2.6 / 4.2 |
| spill | 8 clients | 200 | 222.4 / 235.1 / 245.9 / 251.9 | **+45.9** | 215.0 / 232.1 / 242.8 | 6.9 / 23.0 |

The spill cell's `SET LOCAL` (the harness's, see below) took 2.56 ms at p50
and 9.36 ms at p99. It is left out of the count phase above.

**Why the parallel cell is inconclusive.** Its pick was a50 `pw = 1` and
`pk < 5` with `enable_nestloop = off`, plain median 2,562.8 ms. The harness then
asked only for "a launched `Gather` somewhere in the plan". In that plan the
`Gather` sits in a later `UNION ALL` branch, and the first branch, the one
running at 200 ms, is a serial Merge Semi Join. So the cancel never waited
for workers. Separately, the 8-client half overlapped the `buildx` build
(six `go build`s, VM load average 288). The harness now requires a launched
`Gather` **in the first branch** with an EXPLAIN total ≥ 1.25 × the budget.
On run 5's plans that criterion selects a50 `pw = 1` with `enable_nestloop =
off`: a Parallel Hash Semi Join whose `Gather` runs 479–582 ms. That is the
candidate a rerun should pick.

### What the harness does now

- **Three cells** (`NAVA_PERF_SPIKE_CELLS`, default `base,spill,parallel`).
  Each picks from its own candidate list the slowest plain (no EXPLAIN)
  median among the candidates whose plan has the cell's shape in the first
  `UNION ALL` branch. `spill` needs a hash join with temp writes there;
  `parallel` needs a launched `Gather` there, still running at 1.25 × the
  budget.
- **The cell's settings cover the count statement alone.** The harness sends
  `SET LOCAL …` on the same connection right after the handler's `SAVEPOINT
  list_count`, through a wrapping pgx tracer (`gateCountTracer`, test file
  only, no production change). `ROLLBACK TO` ends the settings, so the page
  statement runs on the server's settings. The extra round trip is timed and
  left out of the count phase. A first try set the settings as pool startup
  parameters instead. The page walk then spilled too (94 GB of temp writes in
  one spike, on a host Defender was loading), so that try was discarded and
  is not reported.
- **Leak check:** `pg_ls_tmpdir()` (files) and `pg_ls_dir('base/pgsql_tmp')`
  (which also lists a parallel hash's shared fileset directories, skipped by
  `pg_ls_tmpdir()`), right after the spike and 60 s later, plus the count of
  `parallel worker` backends. `pg_stat_database.temp_files` is reported only
  to show that a spill happened.
- **Host gates around each cell** (P3-36's floor and control, before and
  after). The sampler in `run.sh` also records the Docker VM's load average:
  a `buildx` build runs in BuildKit's own containers, which `docker stats`
  does not list. That is how run 5's late contention was found. Run 5's
  `raw.md` came from the version before these last two changes (one gate pair
  around the whole spike, and "a `Gather` anywhere").

Rerun the two new cells (about 20 minutes on an idle host, no `nava-*`
stack up):

```sh
NAVA_PERF_GATE_ATTEMPTS=0 NAVA_PERF_SPIKE_CELLS=spill,parallel sh scratch/tick-1048-a2-gate/run.sh
```

