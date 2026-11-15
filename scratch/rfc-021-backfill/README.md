# RFC-021 critique round 1: Postgres spike (TICK-897)

This spike measures the RFC-021 rev 0 backfill (D3, D7) and filter (D12) mechanics on synthetic data shaped like `main`. It is not a migration. The Solution Architect archives it under `docs/adr/ADR-009/spikes/postgres/r1/` in step 4.

## Setup

- **Hardware:** Intel i7-1260P laptop (12 cores / 16 threads), 40 GB RAM, Docker Desktop on WSL2 (16 vCPU, 24 GB visible).
- **PostgreSQL:** 16.10 (`postgres:16.10`, the compose version). `shared_buffers=1GB`, `max_wal_size=8GB`, `jit=off`, `log_lock_waits=on`, `deadlock_timeout=50ms`. Parallel query is off, because the container's 64 MB `/dev/shm` refused parallel workers.
- **Data:** 10 tenants × 100k = 1.0M records (868 MB heap). The type under test is tenant 1's single type, 100k records. `main`'s three list indexes, the `audit_log` envelope with its commit key, and ADR-006 D3 `rfv_num` with a D4-style trigger on one field (`priority`). Forced RLS everywhere. Plans run as `nava_app`, the backfill as `nava_worker`, with custom plans.
- **Scenario A:** 60% of the type (60k) lacks `priority`. The other 40% hold {1,2,4,5}. The new default 3 is stored nowhere.
- **Scenario B:** only 34 of the 100k lack `priority`. This is the `failed_records` tail, the end of the verification loop, or a default added to a field most records already fill.

## Files

| File | What it does |
|---|---|
| `01_schema.sql`, `02_seed.sql`, `02b_reshape.sql` | Schema and seed for scenario A |
| `03_plans.sql` → `plans-A-60pct-absent.txt` | J0/J1 (JSONB `@>`), S0/S1 (side table), closed and open window, page and capped count |
| `04_scenario_b.sql`, `05_plans_b.sql` → `plans-B-34-absent.txt` | Scenario B. Also S3 (UNION ALL, absent branch driven by an id table) and S4 (`(tenant, def, schema_version)` index) |
| `06_pending_dense.sql` → `plans-P-60k-pending.txt` | Absent branch driven by an id table with 60k ids (the dense case) |
| `07_reset_a.sql`, `08_backfill_proc.sql` | `CALL backfill(batch, lockmode)`. Per record it runs the statements `pipeline.update` sends: `store.Get`, `PayloadJSON` before, versioned `UPDATE`, `PayloadJSON` after, audit `INSERT`, with the D4 trigger live. It is server side, so it has **no network round trips** and is a lower bound on lock hold time |
| `run_concurrent.sh`, `w_update.sql`, `w_share.sql` → `concurrent-results.txt` | The backfill with live user updates (100/s) and `record_shares` FK inserts (25/s), measured by pgbench |

## Results

### Filters (D12), ms, `EXPLAIN (ANALYZE, BUFFERS)`

| Query | A: 60k absent | B: 34 absent |
|---|---|---|
| Side table, window closed, page | 1.44 | 0.13 |
| **Side table, D12 `OR NOT (data ? f)`, page** | **0.59** | **98.6** (69,409 rows removed by filter) |
| Side table, D12, capped count (1,001) | 2.65 | 52.6 (99,966 removed) |
| JSONB, D12, page | 0.30 | 99.4 |
| JSONB `@>` closed (today's unindexed scan) | 442 | n/a |
| UNION ALL, absent branch from an id table (PK), page | 258 (60k ids) | **0.45** |
| UNION ALL, absent branch from a `schema_version < 3` index | n/a | 43.1 (still 99,966 removed) |

- The `schema_version` index does not narrow anything. D3's no-op skip leaves the 40k records that already held the key at their old `schema_version` (`old_sv = 40000` after the run).
- The 60k-row id table takes 441 ms to fill and is 11 MB.

### Backfill (D3, D7)

| Run | Rewritten | Wall time | WAL | WAL per record |
|---|---|---|---|---|
| b100, alone | 60,000 | 22.1 s; 88.1 s on a later run (host drift) | 229–233 MB | 3,998–4,065 B |
| b500, alone | 60,000 | 13.3 s | 242 MB | 4,221 B |
| b50 / b100 / b500 with live load | 60,000 | 2 min 1 s to 2 min 25 s | 228–237 MB | 3,987–4,145 B |

- WAL per record does not depend on batch size: 4.0–4.2 KB in all seven runs, against 1.24 KB per row for the ADR-006 D10 backfill.
- Batches: 1,000 at size 100, for 60k rewrites. The 40k records that need no change still take batch slots, and pauses.
- With an unthrottled backfill, user p95 on this host went from 18–24 ms to 3–10 s. Only **3 lock waits over 50 ms** were logged across those runs, so the cause is I/O saturation, not row locks. Wall times drifted up to 4× between identical runs, so this host's timings support only orders of magnitude. The WAL bytes are deterministic.
- After the run: `rfv_num` holds 100,000 rows (60,000 at the default), and 0 records lack the key. The D4 trigger converges with no change.
