# Performance budgets

Each row is a cost the platform keeps in bounds: what is measured, how, the budget, and the baseline it was set from. A change that breaks a budget needs a new measurement and the owner's sign-off.

| Budget | Measured as | Review threshold (report-only in CI, ADR-004-E4) | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Commit-check trigger (`audit_commit_check`, ADR-004 decision 2), single subject | ADR-004-E3 method: `pg_stat_user_functions` self_time / calls, `track_functions = pl`. One connection (pooled-backend state), N single-subject transactions as `nava_worker`, at N = 200, 1,000 and 5,000. Young partition analysed at 60 rows, autovacuum off on it. 38 partitions. Same-run reference: a copy of the function without the `plan_cache_mode` clause, same fixture, same job. Measured by `TestCommitCheckBudget` in the CI job `commit-check-budget` (`postgres:16.10`, `ubuntu-latest`, 4 vCPU, nothing else on the runner; the host CPU varies, so each baseline names it). A baseline (re-)take is the median of at least 3 runs of the job on that CPU model (ADR-004-E6) | 300 µs per call at any of the three N (ADR-004-E3). **Fails the CI job** (ADR-004-E4 verdict rule), whatever the absolute numbers: `fail_no_rows` (a `COMMIT-CHECK-BUDGET row=` line missing, or a measured value ≤ 0, NaN or missing), `fail_ratio` (same-run clause / reference per call at N = 200 > 5, or import > 15; NaN counts as out of range), `fail_flat` (clause per call max/min ≥ 1.5 across the three N). Over the threshold with in-range ratios and flat is `warn`: a `::warning::` annotation, the job passes. `fail_ratio` with a row over its threshold in the same run is the E3-R2 S3 trigger | CI baselines, 2026-11-03/04, clause per call at N = 200 / 1,000 / 5,000 (max/min), reference at N = 200, ratio: `AMD EPYC 9V45 96-Core Processor`: 126.1 / 123.8 / 125.1 (1.02), ref 93.4, ratio 1.35, [run 41708883885](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41708883885/job/126171856161). `AMD EPYC 7763 64-Core Processor`: 235.7 / 227.7 / 230.5 (1.03), ref 161.9, ratio 1.46, [run 41679998550](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41679998550/job/126071064410). CPU not logged: 170.5 / 163.6 / 164.8 (1.04), ref 133.5, ratio 1.28, [run 41673342312](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41673342312/job/126047559767). Provenance only: TA S2 laptop runs (TICK-502), 248–251 µs at 38 partitions | ADR-004-E3 (method, threshold), ADR-004-E4 (verdict rule); Technical Architect S2 and re-check (TICK-502) | Postgres & Data Modeling Specialist |
| Commit-check trigger, import | ADR-004-E3 method: COMMIT time / 10,000 for one 10,000-subject import, fresh `nava_worker` session, tenant absent from the statistics, median of 3, 38 partitions. Same-run reference and CI job as the row above | 175 µs per probe (ADR-004-E3). Same fail conditions as the row above | CI baselines, 2026-11-03/04, clause per probe (runs), reference, ratio: `AMD EPYC 9V45 96-Core Processor`: 80.3 (80.2 / 81.5 / 80.3), ref 9.9, ratio 8.13, [run 41708883885](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41708883885/job/126171856161). `AMD EPYC 7763 64-Core Processor`: 194.9 (195.3 / 194.9 / 191.2), ref 22.7, ratio 8.58, over the threshold (`warn` under ADR-004-E4). Slow host, not a regression: the ratio is in line with the other models, and the TICK-526 spike read 189.0–193.5 over 7 runs (ADR-004-E6), [run 41679998550](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41679998550/job/126071064410). CPU not logged: 127.7 (127.4 / 127.7 / 129.3), ref 13.1, ratio 9.75, [run 41673342312](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41673342312/job/126047559767). Provenance only: TA laptop, 120–142 µs at 2 partitions | ADR-004-E3 (method, threshold), ADR-004-E4 (verdict rule); Technical Architect S2 and re-check (TICK-502) | Postgres & Data Modeling Specialist |

## Migration lock holds

Locks a migration holds on tables that request paths need. `lock_timeout` bounds only the wait to take a lock, not how long the migration then holds it; these rows bound the hold.

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Migration A/B tenant-lock hold (S8 backfill, `audit_backfill_s8()`): ACCESS EXCLUSIVE on `tenants`, `entity_definition_versions` and `audit_log` until COMMIT | Wall time of `SELECT audit_backfill_s8()` as `nava_migrator`, `scratch/audit-backfill-s8/run.sh` | ≤ 18 s at 100k S8 versions | 0.18 ms per backfilled version, linear. A re-run with every entry present takes 2.32 s at 300k. Measured 2026-11-03 on postgres:16.10 with 16 CPUs | PR #414 `scratch/audit-backfill-s8/RESULT.md` (P5-15, TICK-433); Technical Architect's review §3 | Postgres & Data Modeling Specialist |
| Migration B lock hold (P5-32): ACCESS EXCLUSIVE on `tenants`, `entity_definitions`, `entity_definition_versions`, `audit_log`, `audit_chain_head` and `outbox` until COMMIT (R27) | Wall time of migration B alone (`migrate.Up` of `20261107053707_audit_migration_b.sql`), `docs/data/audit-migration-b-bench/run.sh` | ≤ 5 s at 10,000 tenants and 300,000 S8 versions | 4.29 s at 10,000 tenants / 300,972 versions (backfill re-run 2.86 s, seeding 1.28 s); 0.48 s at 1,000 / 30,093. About 0.13 ms per tenant plus 9.5 µs per version plus 0.1 s, linear. Measured 2026-11-07 on postgres:16.10 with 16 CPUs | `docs/data/audit-migration-b-bench/RESULT.md` (P5-32, TICK-442) | Postgres & Data Modeling Specialist |

## Record history

A history page replays the record's whole audit history (`read.Reader.RecordEntries`), so the history is capped rather than paged in the database.

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Record history length (`GET /api/v1/records/{entity}/{id}/history`) | Audit entries of one record (create, update, state_change), counted before any is read. `TestPatchHistoryIsCapped` proves one page reads at most the cap's rows, with the `Reader.OnRecordRows` hook | 500 entries (`read.DefaultMaxRecordEntries`). Over it the endpoint answers `422 history_too_long`, logs `record history longer than the cap` and counts `nava.records.history.too_long`. Nothing is read or replayed | Page cost is O(entries) per page, so a walk is O(N²/limit); the cap bounds it. Trigger to lift the cap: TICK-548 (stored inverse diffs or snapshots, a schema change) | P5-17-fu2 (TICK-542), Technical Lead decision on the issue | Principal Software Engineer |

## Record numbering

ADR-005 §10 (`docs/adr/ADR-005-record-numbering.md` on `main`). A `gaps` type (the default) takes its number with L2s on the allocator pool; a `gapless` type takes it last inside the create transaction (L1).

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Counter wait, `gaps` type (L2s) | p95 of the time an allocation waits on its `record_counters` row, inside PostgreSQL. 8 clients creating one type in one scope, spike S1 method (SQL level, no HTTP) | **≤ 2 ms** p95 | Spike S1: 0.84 / 0.99 ms (p50 / p95) on a quiet host, 0.97 / 1.20 ms on a contended host. For context only, not this measure: the TICK-621 HTTP harness reads "counter wait" as the whole `AllocTx` trip seen by the handler's tracer, round trips included. Its final re-run (`25aedcf0`) read 5.4 to 11.7 ms p95 on 8-client `gaps` attempts that the storage-window gate ruled host-invalid, with `nava_numbering_alloc` acquire p95 ≤ 1.66 ms and 0 timeouts | ADR-005 §1, §2.2, §10; spike S1 (`docs/adr/ADR-005/spikes/technical-architect/s1/`); `docs/quality/acl-perf-800k.md` (PR #544, `1ede2364`) | Postgres & Data Modeling Specialist |
| Create, 1 client, over HTTP | Request p95 of `POST /api/v1/records/{entity}` on a quiet host, TICK-621 harness (`NAVA_PERF_ONLY='^create'`), frozen gates | **≤ 50 ms** p95 | **32.6 ms**, a valid attempt that meets: final quiesced re-run at `25aedcf0` (harness `469687ff`), 2026-11-10. This closes ADR-005 A3 for 1 client | `docs/quality/acl-perf-800k.md` "Final quiesced create re-run after TICK-692" (PR #544, `1ede2364`); ADR-005 A3 | Quality, Testing & CI Advisor |
| Create, 8 clients, `gaps` type | Request p95, 8 clients creating one type in one tenant, same harness | **≤ 50 ms** p95. Applies to `gaps` types only. **Pending decision on TICK-621** (options A/B are with vknavaladi) | Final re-run: all 9 attempts host-invalid on the storage-window gate (commit p50 11.4 to 16.6 ms). With the gate set aside, 8 of 9 attempts read over 50 ms (agent 66.4 / 90.0 / 71.0, admin 81.1 / 66.0 / 52.6, small tenant 50.0 / 63.1 / 73.1 ms) | `docs/quality/acl-perf-800k.md` (PR #544, `1ede2364`); ADR-005 §2.1 | Quality, Testing & CI Advisor |
| **Waiver: create, 8 clients, `gapless` type (L1)** | Request p95, 8 clients creating one `gapless` type in one tenant, same harness | **≤ 213.7 ms** p95. The 50 ms 8-client target is waived for `gapless` types only: their creates queue on the type's counter row by design (ADR-005 §2.1) | **213.7 ms**, final re-run at `25aedcf0`, one attempt, reported only; counter wait p50 / p95 42.0 / 188.6 ms. Earlier runs the same day: 189.0, 383.0 and 228.7 ms | `docs/quality/acl-perf-800k.md` (PR #544, `1ede2364`); ADR-005 §2.1 (L1 waiver) | Postgres & Data Modeling Specialist |

## Connection sizing

Every server connection the compose stack can open to its one PostgreSQL, counted at its configured ceiling, fits in `max_connections` minus `superuser_reserved_connections`. `TestConnectionBudgetFits` (`deploy/compose/connbudget_test.go`) recomputes the sum from `compose.yaml`, `deploy/compose/pgbouncer/pgbouncer.ini`, the Temporal config and `cmd/nava-core/allocator.go`, and fails when it no longer fits. `TestAllocatorPoolSizing` checks that the `nava_alloc` pool is `allocatorMaxConns × nava-core replicas`.

Through PgBouncer, each (database entry, role) pair is one server pool, however many replicas share it. Its size is the role's `[users] pool_size`, else the entry's `pool_size`, else `default_pool_size`. No pool has a reserve.

With R nava-core replicas behind one PgBouncer:

```
  (main 20 + allocator 4) × R            nava/nava_app (default_pool_size) + nava_alloc/nava_app (ADR-005 A5)
+ 5                                      nava/nava_audit_reader, per PgBouncer instance (ADR-004-A1 A1-8)
+ 53                                     workers: nava_worker 20, nava_relay 20, nava_onboard 8, nava_purge 3, nava_groupsync 2
+ 2                                      pgbouncer_auth, the auth_query lookup, 1 per database entry
+ 20                                     Keycloak (KC_DB_POOL_MAX_SIZE)
+ 120                                    Temporal: 4 services in one process × (20 default store + 10 visibility)
+ 2                                      migrator (goose) and temporal-schema, one-shot
≤ max_connections 250 − superuser_reserved_connections 3 = 247
```

At R = 1 (compose) the total is **226**, 21 under the limit. Each further replica adds 24, so a second replica needs `max_connections` raised too: R replicas need `max_connections ≥ 226 + 24 × (R − 1) + 3`. Raise `default_pool_size` (or give the `nava` entry a `pool_size`) to `20 × R` and `nava_alloc`'s `pool_size` to `4 × R` with it.

- The main pool's server side is shared: today one PgBouncer gives `nava_app` 20 server connections for every nava-core replica and nava-onboard's `nava_app` pools together.
- The allocator has its own entry, `nava_alloc`, so open creates holding every `nava` server connection cannot starve it (TICK-657).
- The worker and relay roles keep `default_pool_size` because their client pools use pgxpool's default width, `max(4, CPUs)`. The purge, onboard and groupsync caps match their clients' `MaxConns`, so the caps never make a client wait.
- Temporal is counted at its configured ceiling, not at what it was seen to use; it is 120 of the 226.
- Trigger to revisit (AR-TICK-653-04): more than about 10 API replicas, or connection pressure on PostgreSQL. At that point the allocator alone is 40 server connections.
- CI's integration pooler (`scripts/ci/pgbouncer/pooled.ini`) mirrors this with `nava_alloc` `pool_size=4` against a 5-connection `nava` pool. `isolation.ini` and `noprep.ini` have no allocator entry because neither suite allocates numbers.
