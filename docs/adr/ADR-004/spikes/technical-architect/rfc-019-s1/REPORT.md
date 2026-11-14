> Source: Technical Architect, TICK-864 comment `01a1db42-ff0a`, 2026-11-14. Archived verbatim for ADR-004 E9. Spike S1 of RFC-019.

## Spike S1 report (RFC-019 F1): the R2b audit row on the allocator pool

**Verdict: FAIL.** P2 met the criterion in 6 of 7 reps and broke it in 1. Under the rule we agreed ("fail if any B timeout or p95 above 1.5× baseline"), **F1 is now blocking.** Tenant B never timed out. In the failing rep, B's p95 rose to **114 ms, 32× its baseline**, and A had 12 timeouts. The cause is Known from the code, and it applies in the passing reps too. The R2b row is a **synchronously committed** audit transaction, placed on a 4-connection pool whose transactions are tuned to commit **asynchronously** and take about 2 ms. One measured alternative held in 5 of 5 reps; it is under Implications.

### Hypothesis and criterion (as approved)
- **Hypothesis:** tenant A retries refused `gaps` creates at its full rate (50 rps, burst 100, one replica). Each refusal writes the R2b row on the pipeline's `TxRunner`, which is the allocator pool. Tenant B's allocations still meet ADR-005 A5.
- **Pass:** B's allocation p95 ≤ 200 ms, 0 B timeouts, and B's p95 ≤ 1.5× its baseline with A idle. **Fail:** anything else.

### Setup
- **Code:** PR #679 head `111a3527` (`TICK-674/allocator-formats`), Go 1.26.11, pgx v5.11.0. The real `PoolRunner.Allocate` → `tenancy.AllocTx` → `store.QueueAllocate`, and the real `audit.Append`.
- **Database:** `postgres:16.10`, the pinned image, migrated with the real migrations, `nava_app` under RLS. Durable defaults: `fsync=on`, `synchronous_commit=on`, unlike CI's `fsync=off`. Container limited to 4 CPUs and 2 GB. Docker Desktop on Windows 11, i7-1260P, 40 GB RAM.
- **Pools:**
  - Allocator pool `MaxConns=4` (`allocatorMaxConns`), built with `AllocPoolConfig`, `AllocTimeout` = production 200 ms.
  - API pool `MaxConns=20`, the same as `default_pool_size`.
  - No PgBouncer in the path, so acquire times are if anything optimistic.
- **Data:** two tenants with one `gaps` type each, default format. 16k to 40k R2b-shaped rows per run.

### Method
- Load is open-loop: every arrival is its own goroutine, the way concurrent HTTP requests are admitted.
- **B** runs at 20 rps for 30 s per phase. Each arrival opens a create transaction on the API pool and times `PoolRunner.Allocate` inside it. That is what `nava_numbering_alloc_seconds` measures, acquire wait included.
- **A** sends a burst of 100, then 50 rps. Each arrival opens its create transaction on the API pool, runs `Allocate`, and then:
  - **P1:** nothing more. This is PR #679 today.
  - **P2:** writes the R2b row on the allocator `PoolRunner` while its create transaction is still open, as RFC-019 specifies.
  - **P2-alt:** rolls back, then writes the row on the API pool.
  
  In every case A's create transaction rolls back.
- **Row stub:** the row is R2b's shape (domain `record`, subject `entity_definition`/D, `/number_space` diff with the four keys). Its `action` is `cascade_too_deep`, because neither the Go nor the SQL action list has `number_space_exhausted` yet. That is the same table, columns, indexes, Append path and diff size.
- **P3 and P4** push A to 5× and 20× the rate limit, to see the headroom.

### Raw results (B = tenant B's allocation latency)

**Run 1**

| Phase | B p50 | B p95 | B p99 | B max | B timeouts | A rows | A timeouts | A audit-tx p95 |
|---|---|---|---|---|---|---|---|---|
| P0 baseline, A idle | 2.35 | 3.91 | 9.10 | 27.93 | 0 | 0 | 0 | — |
| P1 A at limit, no R2b row | 2.24 | 3.30 | 5.43 | 11.30 | 0 | 0 | 0 | — |
| **P2 A at limit, R2b row (RFC-019)** | 1.85 | **3.10** | 3.89 | 18.77 | 0 | 1599 | 0 | 19.61 |
| P3 A at 5×, R2b row | 2.12 | 24.03 | 31.28 | 59.68 | 0 | 7996 | 0 | 32.87 |
| P4 A at 20×, R2b row | 20.57 | 30.59 | 108.57 | 152.05 | 0 | 30270 | 51 | 48.81 |
| P0' baseline again | 2.48 | 3.19 | 3.81 | 4.78 | 0 | 0 | 0 | — |

**Run 2**

| Phase | B p50 | B p95 | B p99 | B max | B timeouts | A rows | A timeouts | A audit-tx p95 |
|---|---|---|---|---|---|---|---|---|
| P0 baseline | 2.28 | 3.52 | 4.07 | 8.44 | 0 | 0 | 0 | — |
| P1 no R2b row | 2.23 | 3.61 | 4.51 | 21.08 | 0 | 0 | 0 | — |
| **P2 R2b row (RFC-019)** | 2.58 | **114.37** | 163.93 | 205.96 | 0 | 1581 | **12** | **169.64** |
| P3 5× | 2.27 | 25.09 | 28.32 | 32.36 | 0 | 7992 | 0 | 34.51 |
| P4 20× | 19.63 | 26.81 | 83.30 | 136.61 | 0 | 30457 | 0 | 37.12 |
| P0' baseline | 2.41 | 3.26 | 3.94 | 4.69 | 0 | 0 | 0 | — |

**Run 3: 5 reps, RFC-019 placement against the alternative** (P0 baseline p95 3.15). There were no checkpoints in any phase except rep 5 P2 (1). Each P2 phase made about 1,600 `wal_sync` calls, one per R2b row; the P0 baseline made 177.

| Rep | P2, row on **allocator** pool: B p95 / p99 / max | A audit p95 | P2-alt, row on **API pool after rollback**: B p95 / p99 / max | A audit p95 |
|---|---|---|---|---|
| 1 | 3.71 / 11.18 / 30.58 | 21.42 | 3.71 / 5.11 / 6.84 | 54.83 |
| 2 | 3.14 / 4.61 / 20.30 | 20.79 | 3.20 / 3.64 / 4.28 | 31.95 |
| 3 | 3.03 / 3.57 / 20.47 | 19.27 | 3.00 / 3.34 / 4.03 | 37.24 |
| 4 | 3.09 / 3.61 / 16.77 | 19.84 | 2.99 / 3.48 / 4.11 | 32.87 |
| 5 | 3.19 / 4.11 / 18.77 | 17.93 | 3.01 / 3.30 / 3.73 | 32.91 |

All times are in ms. B had 0 timeouts and 0 errors in every phase of every run.

### Implications

**Known (code, plus these numbers):**
- **Why the row is expensive on that pool.** An allocation runs `SET LOCAL synchronous_commit = off` (`internal/tenancy/alloc.go:31`) and holds its allocator connection for about 2 ms. The R2b row is written by `tenancy.InTenantTx`, which commits synchronously. It holds an allocator connection through a WAL flush, about 20 ms p95 here, so roughly **10× as long** as the allocation it follows.
- **Where it hurts B.** With A at the limit, 1,600 such transactions per 30 s take about 0.8 of the 4 connections on average. Any stall in the WAL flush (run 2: audit p95 170 ms) now holds allocator connections, so other tenants' allocations queue behind it.
- **The control has no such exposure.** In P1 the allocator pool never waits on a flush, and B's p95 matched its baseline in both runs.
- **Even when P2 passes,** B's worst case is 4 to 7× worse than with P2-alt: max 16.8 to 30.6 ms against 3.7 to 6.8 ms.

**Inferred:**
- The run-2 stall was a slow host fsync. It is not explained by a checkpoint, but run 2 has no checkpoint counters. Docker Desktop disks are slow, but production disks also stall: under failover, backup I/O or a noisy volume. The mechanism does not depend on how often that happens. It decides **which pool pays** when it happens.
- Multiple replicas do not change the result. Each replica has its own 4-connection pool, and the per-tenant limit is per replica, so the load per pool stays the same.

**What F1 now requires (blocking for rev 1):** the R2b row must not be written on the allocator pool. Two options I would put to the Solution Architect:

1. **Measured: write the row after the create transaction has ended, on the API pool.** This is P2-alt, which passed 5 of 5 reps. B's p99 was ≤ 5.11 ms, the same as baseline.
   - It cannot reproduce the TICK-657 starvation. The caller has released its connection before it asks for one, so there is no hold-and-wait.
   - **Cost:** the pipeline cannot do this from inside `Write`, because the caller's transaction is still open there. The refusal is returned to the caller with the entry to append, and the caller (`internal/server/records.go`, or any later create path) appends it after `InTenantTx` returns.
   - **Proposed R2b text:** "The entry is written in a transaction of its own on the caller's pool, **after** the refused create's transaction has ended (never on the allocator pool, never while a create transaction is open). `Write` returns the refusal carrying the entry; the caller appends it and returns the refusal joined with any audit error (CI-A18 f)."
   - **Add CI-A18 (h):** a refused create's audit row is written by no allocator-pool connection. Check it with a `PoolRunner` whose pool is closed after allocation: the row must still be written.
2. **Not measured: a separate tiny audit pool for the pipeline's own-transaction rows.** This keeps the R2b shape inside `Write`, but it adds a pool and a PgBouncer entry, and it would need its own spike. I lean toward option 1: it adds no new component.

**Same exposure, much rarer:** `rejectTooDeep` (`pipeline.go:549`) has the same placement today. Cascade depth bounds how often it fires, not client retries, so I am not reopening E1. I note it for whoever settles option 1 vs 2, because one rule for both rows would be simpler.

**Budget note for `docs/budgets.md` (proposed, for the Postgres & Data Modeling Specialist):** "Allocator-pool transactions are allocations only (`synchronous_commit = off`, about 2 ms). No synchronous-commit transaction runs on the allocator pool. Measured: S1, TICK-864, 2026-11-14, PG 16.10, PR #679 `111a3527`." Without that rule, F1 can come back through any later own-transaction write.

### How to rerun
1. `git worktree add ../s1wt 111a3527`, then copy the file below to `internal/records/pipeline/zz_spike_s1_integration_test.go`.
2. `docker run -d --name s1-pg -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=<local-only> -p 127.0.0.1:55864:5432 --cpus=4 --memory=2g postgres:16.10 -c max_connections=200`
3. `NAVA_TEST_PG_URL=postgres://pgadmin:<local-only>@127.0.0.1:55864/postgres S1_SECONDS=30 go test -tags integration -count=1 -timeout 15m -run '^TestSpikeS1RefusalFloodOnAllocatorPool$' -v ./internal/records/pipeline/`. Set `S1_PHASES=reps S1_REPS=5` for run 3.

The run takes about 4 to 6 minutes. The spike code is throwaway: it was not committed, and the container and worktree were removed.

The spike code is archived beside this report as `zz_spike_s1_integration_test.go.txt`. It has a `.txt` suffix so that `go test -tags integration ./...` does not compile it. Copy it into `internal/records/pipeline/` without the suffix to rerun it.
