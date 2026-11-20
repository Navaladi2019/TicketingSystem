# P3-36 raw report (TICK-621)

Generated 2026-11-19T23:18:44Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

## Server

```
PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
effective_cache_size = 786432 8kB
fsync = on
jit = off
max_parallel_workers_per_gather = 2
random_page_cost = 4
shared_buffers = 262144 8kB
synchronous_commit = on
track_io_timing = on
work_mem = 4096 kB
```

nava_app role settings: `plan_cache_mode=force_custom_plan`

## Data

```
records 800000, open 120278, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 400000
record_shares 15815
heap 1042 MB, indexes 754 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.32 / p95 2.34 / p99 3.15 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_logical_emit_message(true, ...); COMMIT` probe, a commit that flushes WAL with no INSERT, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). An attempt is also void if a container other than the run's database, the buildkit builder and the three Multica platform containers runs during it. A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| agent list by group, default sort, page 1 | - | large | 2260 | 63.98 | 108.58 | 140.96 | - | - | - | - | - | 150 | meets |
| agent list by group, default sort, page 20 | - | large | 2420 | 60.01 | 103.18 | 130.38 | - | - | - | - | - | 150 | meets |
| requester list (heavy requesters), page 1 | - | large | 2979 | 48.91 | 83.47 | 101.01 | - | - | - | - | - | 150 | meets |
| requester list (heavy requesters), page 20 | - | large | 3014 | 47.90 | 83.58 | 99.04 | - | - | - | - | - | 150 | meets |
| admin list, default sort, page 1 | - | large | 2927 | 50.22 | 92.80 | 117.02 | - | - | - | - | - | 150 | meets |
| admin list, default sort, page 20 | - | large | 2964 | 49.56 | 89.88 | 111.23 | - | - | - | - | - | 150 | meets |
| agent list, filter[state]=new | - | large | 2547 | 58.45 | 92.35 | 110.64 | - | - | - | - | - | 150 | meets |
| agent list, filter[priority]=1 (data) | - | large | 792 | 199.81 | 246.00 | 271.80 | - | - | - | - | - | 150 | **MISS** |
| agent list, filter[site]=s7 (rare data, < 1% of visible rows) | - | large | 771 | 204.29 | 248.24 | 276.72 | - | - | - | - | - | 150 | **MISS** |
| agent list, filter[site]=s7, short data-filter timeout (bound check) | - | large | 6185 | 24.12 | 35.47 | 45.07 | - | - | - | - | - | 0 | n/a (bound check) |
| agent list, sort=-updated_at | - | large | 2671 | 53.67 | 95.92 | 117.06 | - | - | - | - | - | 150 | meets |
| agent list, sort=assignee | - | large | 2848 | 51.06 | 86.40 | 107.91 | - | - | - | - | - | 150 | meets |
| admin list, filter[state][in]=new,in_progress, sort=-updated_at | - | large | 3238 | 45.02 | 82.94 | 98.97 | - | - | - | - | - | 150 | meets |
| small tenant: agent list, page 1 | - | small | 2391 | 60.81 | 105.46 | 127.86 | - | - | - | - | - | 150 | meets |
| small tenant: admin list, page 1 | - | small | 3247 | 43.23 | 85.89 | 170.56 | - | - | - | - | - | 150 | meets |
| single GET, agent | - | large | 9481 | 14.22 | 31.10 | 41.65 | - | - | - | - | - | 50 | meets |
| single GET, admin | - | large | 9133 | 14.54 | 32.78 | 47.75 | - | - | - | - | - | 50 | meets |
| single GET, small tenant agent | - | small | 9026 | 14.63 | 32.83 | 49.32 | - | - | - | - | - | 50 | meets |
| composed: agent list by group, default sort, page 1 | - | large | 1944 | 71.87 | 149.42 | 179.11 | - | - | - | - | - | 150 | meets |
| composed: agent list by group, default sort, page 20 | - | large | 2055 | 70.05 | 124.81 | 174.98 | - | - | - | - | - | 150 | meets |
| composed: agent list, filter[state]=new | - | large | 2269 | 64.07 | 106.56 | 141.26 | - | - | - | - | - | 150 | meets |
| composed: single GET, agent | - | large | 7828 | 17.74 | 37.57 | 55.70 | - | - | - | - | - | 50 | meets |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.32 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |

## Attempt setup and host-probe samples (creates)

Before each create attempt: wait until no container runs other than the run's database, `buildx_buildkit_nava-ci0` and the three `multica-*` platform containers (docker ps is polled every 5 s during the attempt too; another container voids it), then until no autovacuum or analyze backend works in the bench database, then the host-speed probe and the storage probe. `pg_stat_activity` is read at the probe's midpoint (every non-idle backend and every non-client backend, other than the reader); `docker stats` is one sample taken while the probe runs. The seed was quiesced once with `VACUUM (ANALYZE)` and `CHECKPOINT`.


## Data-filter statement timeout (TICK-647)

Default timeout 2s (store.DataFilterTimeout); the bound check runs with 2ms. A 422 is `filter_too_expensive`. Latency includes the 422s.

| Scenario | Requests | 422 | p50 | p95 | max |
| --- | --- | --- | --- | --- | --- |
| agent list, filter[priority]=1 (data) | 792 | 0 | 199.81 | 246.00 | 335.27 |
| agent list, filter[site]=s7 (rare data, < 1% of visible rows) | 771 | 0 | 204.29 | 248.24 | 291.69 |
| agent list, filter[site]=s7, short data-filter timeout (bound check) | 6185 | 6185 | 24.12 | 35.47 | 72.55 |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### agent list by group, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2260 | 3.66 | 10.72 | 19.67 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2260 | 4.28 | 10.69 | 19.73 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2260 | 3.65 | 10.29 | 18.54 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2260 | 3.95 | 9.52 | 15.73 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2260 | 6.87 | 14.49 | 25.47 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2260 | 19.54 | 45.43 | 71.59 |
| 7 | `commit` | 2260 | 4.04 | 12.83 | 23.74 |

### agent list by group, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2420 | 3.26 | 10.00 | 21.53 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2420 | 4.00 | 9.38 | 18.07 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2420 | 3.35 | 9.57 | 17.50 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2420 | 3.60 | 9.59 | 17.75 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2420 | 6.06 | 11.96 | 20.01 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2420 | 18.18 | 44.19 | 59.80 |
| 7 | `commit` | 2420 | 3.60 | 12.63 | 27.20 |

### requester list (heavy requesters), page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2979 | 2.97 | 8.45 | 18.61 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2979 | 3.78 | 9.12 | 16.10 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2979 | 3.22 | 8.60 | 17.66 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2979 | 3.49 | 8.38 | 15.37 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2979 | 5.18 | 10.86 | 21.25 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2979 | 13.00 | 31.62 | 47.22 |
| 7 | `commit` | 2979 | 3.04 | 9.32 | 19.47 |

### requester list (heavy requesters), page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 3014 | 2.87 | 7.91 | 17.40 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3014 | 3.62 | 8.63 | 15.32 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3014 | 3.10 | 8.57 | 16.89 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3014 | 3.31 | 8.22 | 16.39 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3014 | 5.15 | 10.93 | 20.75 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3014 | 12.98 | 34.82 | 48.82 |
| 7 | `commit` | 3014 | 2.94 | 8.65 | 17.72 |

### admin list, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2927 | 3.22 | 10.29 | 19.61 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2927 | 3.90 | 10.97 | 19.39 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2927 | 3.41 | 10.51 | 21.60 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2927 | 3.60 | 10.35 | 19.71 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 2927 | 3.81 | 10.17 | 18.22 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 2927 | 11.80 | 35.09 | 49.79 |
| 7 | `commit` | 2927 | 3.19 | 11.38 | 21.61 |

### admin list, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2964 | 3.20 | 10.41 | 19.29 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2964 | 3.92 | 10.41 | 19.50 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2964 | 3.35 | 10.14 | 19.26 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2964 | 3.61 | 9.41 | 16.55 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 2964 | 3.90 | 10.60 | 19.65 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 2964 | 11.66 | 33.04 | 50.21 |
| 7 | `commit` | 2964 | 3.12 | 10.54 | 20.59 |

### agent list, filter[state]=new

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2547 | 2.93 | 7.89 | 18.53 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2547 | 3.79 | 9.03 | 17.27 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2547 | 3.10 | 8.06 | 15.98 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2547 | 3.40 | 7.53 | 17.32 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2547 | 6.04 | 11.08 | 18.96 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2547 | 20.30 | 41.40 | 57.98 |
| 7 | `commit` | 2547 | 3.17 | 9.94 | 20.45 |

### agent list, filter[priority]=1 (data)

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 792 | 3.45 | 7.84 | 12.16 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 792 | 4.55 | 7.96 | 11.54 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 792 | 3.70 | 7.16 | 9.21 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 792 | 4.07 | 7.94 | 12.56 |
| 5 | `SELECT current_setting('statement_timeout')` | 792 | 3.21 | 6.67 | 9.38 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 792 | 3.04 | 6.36 | 9.44 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 792 | 68.35 | 86.32 | 97.87 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 792 | 91.90 | 115.27 | 129.33 |
| 9 | `SET LOCAL statement_timeout = '0'` | 792 | 3.57 | 8.60 | 14.40 |
| 10 | `commit` | 792 | 3.11 | 6.72 | 11.10 |

### agent list, filter[site]=s7 (rare data, < 1% of visible rows)

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 771 | 3.38 | 6.63 | 10.61 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 771 | 4.41 | 7.89 | 12.69 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 771 | 3.61 | 6.76 | 12.38 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 771 | 4.04 | 7.55 | 11.30 |
| 5 | `SELECT current_setting('statement_timeout')` | 771 | 3.15 | 6.39 | 9.81 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 771 | 3.02 | 5.71 | 8.87 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 771 | 77.58 | 96.29 | 109.18 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 771 | 88.10 | 113.02 | 128.56 |
| 9 | `SET LOCAL statement_timeout = '0'` | 771 | 3.53 | 7.66 | 14.12 |
| 10 | `commit` | 771 | 3.16 | 6.91 | 10.27 |

### agent list, filter[site]=s7, short data-filter timeout (bound check)

Round trips: 8, statements: 8, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 6185 | 2.15 | 3.77 | 5.49 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 6185 | 3.01 | 4.88 | 6.50 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 6185 | 2.41 | 4.21 | 6.44 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 6185 | 2.77 | 4.47 | 6.37 |
| 5 | `SELECT current_setting('statement_timeout')` | 6185 | 2.17 | 3.85 | 5.49 |
| 6 | `SET LOCAL statement_timeout = '2'` | 6185 | 1.94 | 3.69 | 5.14 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 6185 | 4.45 | 6.13 | 7.75 |
| 8 | `rollback` | 6185 | 1.99 | 3.65 | 5.05 |

### agent list, sort=-updated_at

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2671 | 3.08 | 9.12 | 17.27 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2671 | 3.84 | 9.83 | 16.91 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2671 | 3.28 | 9.59 | 19.27 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2671 | 3.47 | 8.66 | 16.10 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2671 | 5.75 | 11.41 | 18.54 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2671 | 15.67 | 37.07 | 56.02 |
| 7 | `commit` | 2671 | 3.18 | 10.30 | 18.90 |

### agent list, sort=assignee

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2848 | 2.97 | 8.72 | 17.05 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2848 | 3.69 | 8.45 | 13.84 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2848 | 3.09 | 8.10 | 16.70 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2848 | 3.37 | 8.11 | 13.97 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2848 | 5.54 | 10.79 | 20.82 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2848 | 14.93 | 35.80 | 49.54 |
| 7 | `commit` | 2848 | 2.92 | 8.98 | 16.84 |

### admin list, filter[state][in]=new,in_progress, sort=-updated_at

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 3238 | 2.92 | 8.83 | 17.57 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3238 | 3.67 | 9.57 | 17.07 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3238 | 3.05 | 9.52 | 18.59 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3238 | 3.31 | 9.01 | 16.29 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 3238 | 3.79 | 9.33 | 16.09 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 3238 | 10.72 | 31.72 | 46.99 |
| 7 | `commit` | 3238 | 2.88 | 9.56 | 18.94 |

### small tenant: agent list, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d', true)` | 6 | 3.33 | 9.25 | 9.25 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2391 | 4.02 | 9.23 | 17.23 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2391 | 3.31 | 9.05 | 18.49 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2391 | 3.54 | 9.48 | 16.36 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2391 | 11.56 | 19.28 | 30.79 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2391 | 18.08 | 42.02 | 64.08 |
| 7 | `commit` | 2391 | 3.12 | 10.22 | 22.68 |

### small tenant: admin list, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d', true)` | 9 | 2.38 | 18.83 | 18.83 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3247 | 3.51 | 10.70 | 21.09 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3247 | 2.98 | 10.06 | 20.89 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3247 | 3.33 | 10.86 | 22.06 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 3247 | 3.50 | 10.13 | 18.80 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 3247 | 9.50 | 31.65 | 50.56 |
| 7 | `commit` | 3247 | 2.78 | 10.20 | 20.27 |

### single GET, agent

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 9481 | 2.45 | 6.04 | 9.92 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 9481 | 7.48 | 18.07 | 25.02 |
| 3 | `commit` | 9481 | 2.42 | 6.26 | 10.72 |

### single GET, admin

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 9133 | 2.47 | 6.52 | 11.04 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 9133 | 7.31 | 18.69 | 27.35 |
| 3 | `commit` | 9133 | 2.51 | 6.99 | 12.06 |

### single GET, small tenant agent

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d', true)` | 24 | 2.51 | 3.57 | 3.64 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 9026 | 7.64 | 18.74 | 26.94 |
| 3 | `commit` | 9026 | 2.54 | 7.01 | 13.00 |

### composed: agent list by group, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 1944 | 4.07 | 16.09 | 28.30 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1944 | 4.79 | 15.86 | 27.12 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1944 | 4.23 | 16.77 | 29.55 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1944 | 4.38 | 14.68 | 27.20 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1944 | 8.97 | 21.46 | 35.39 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1944 | 19.06 | 51.72 | 83.86 |
| 7 | `commit` | 1944 | 4.39 | 18.83 | 31.26 |

### composed: agent list by group, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2055 | 4.21 | 14.68 | 26.57 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2055 | 4.64 | 14.22 | 25.62 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2055 | 4.17 | 13.63 | 26.28 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2055 | 4.16 | 12.95 | 24.34 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2055 | 8.65 | 18.44 | 32.51 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2055 | 18.64 | 46.57 | 72.08 |
| 7 | `commit` | 2055 | 4.31 | 15.26 | 34.27 |

### composed: agent list, filter[state]=new

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 2269 | 3.39 | 10.46 | 19.62 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2269 | 4.15 | 10.65 | 20.72 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2269 | 3.44 | 10.55 | 20.64 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2269 | 3.68 | 10.23 | 20.08 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2269 | 8.51 | 16.05 | 29.09 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2269 | 20.67 | 43.84 | 65.77 |
| 7 | `commit` | 2269 | 3.62 | 12.61 | 22.09 |

### composed: single GET, agent

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '0bda265b-a07a-45c4-915c-f302d32ad2c7', true)` | 7828 | 2.98 | 8.00 | 13.31 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 7828 | 8.87 | 21.43 | 30.93 |
| 3 | `commit` | 7828 | 3.00 | 8.24 | 14.03 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### agent list by group, default sort, page 1

Request: `GET /api/v1/records/ticket` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.061..0.065 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.040..0.043 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.012..0.013 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.483 ms
Execution Time: 0.125 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.044..0.048 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.033..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.170 ms
Execution Time: 0.084 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.105..0.108 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.103..0.105 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.085..0.091 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.071..0.076 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.047..0.049 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.018..0.020 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.480 ms
Execution Time: 0.180 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=528.50..528.51 rows=1 width=8) (actual time=2.263..2.273 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..515.99 rows=1001 width=4) (actual time=0.087..2.077 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..594.78 rows=1154 width=4) (actual time=0.085..1.866 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..583.24 rows=1154 width=0) (actual time=0.084..1.601 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.33 rows=1001 width=0) (actual time=0.083..1.409 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.32 rows=1001 width=4) (actual time=0.081..1.182 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.51 rows=5733 width=4) (actual time=0.081..0.958 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.51 rows=5733 width=0) (actual time=0.066..0.704 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..397.74 rows=140 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                ->  Result  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.34 rows=140 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.91 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.79 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.833 ms
Execution Time: 2.562 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1394.68 rows=1 width=1301) (actual time=0.540..3.516 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1386.34 rows=1 width=1266) (actual time=0.495..3.121 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1378.03 rows=1 width=1259) (actual time=0.451..2.753 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1376.88 rows=51 width=1259) (actual time=0.447..2.650 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3047.26 rows=115 width=1259) (actual time=0.444..2.633 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..2818.57 rows=51 width=1259) (actual time=0.186..2.337 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..316792.96 rows=5733 width=1259) (actual time=0.186..2.324 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..316792.96 rows=5733 width=1251) (actual time=0.172..2.285 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..144.65 rows=51 width=1259) (actual time=0.038..0.045 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..396.34 rows=140 width=1259) (actual time=0.037..0.044 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.34 rows=140 width=1251) (actual time=0.033..0.037 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..37.79 rows=12 width=1259) (actual time=0.036..0.038 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..37.79 rows=12 width=1259) (actual time=0.036..0.037 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.79 rows=12 width=1251) (actual time=0.033..0.033 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.180..0.186 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.179..0.185 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.121..0.163 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.116..0.156 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.081..0.086 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.060..0.073 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.048..0.051 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.025..0.026 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.021..0.021 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.004..0.004 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.323 ms
Execution Time: 3.957 ms
```

### agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.7wYy83cAvO5RGw8ZnfSfU7aXPjj1m5yhQpbtqmg741o` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.056..0.059 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.039..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.449 ms
Execution Time: 0.107 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.035..0.038 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.027..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.147 ms
Execution Time: 0.067 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.063..0.065 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.061..0.063 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.050..0.055 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.042..0.046 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.027..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.011..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.286 ms
Execution Time: 0.110 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=528.50..528.51 rows=1 width=8) (actual time=1.593..1.599 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..515.99 rows=1001 width=4) (actual time=0.066..1.453 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..594.78 rows=1154 width=4) (actual time=0.064..1.301 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..583.24 rows=1154 width=0) (actual time=0.064..1.099 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.33 rows=1001 width=0) (actual time=0.063..0.959 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.32 rows=1001 width=4) (actual time=0.062..0.794 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.51 rows=5733 width=4) (actual time=0.061..0.662 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.51 rows=5733 width=0) (actual time=0.051..0.468 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..397.74 rows=140 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                ->  Result  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.34 rows=140 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.91 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.79 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.250 ms
Execution Time: 1.817 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.38..1416.48 rows=1 width=1301) (actual time=0.673..3.001 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1408.14 rows=1 width=1266) (actual time=0.638..2.736 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.80..1399.83 rows=1 width=1259) (actual time=0.617..2.540 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.80..1398.68 rows=51 width=1259) (actual time=0.607..2.464 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.80..3096.42 rows=115 width=1259) (actual time=0.605..2.451 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..2867.39 rows=51 width=1259) (actual time=0.062..1.872 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..309464.15 rows=5505 width=1259) (actual time=0.062..1.862 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..309464.15 rows=5505 width=1251) (actual time=0.056..1.833 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..145.31 rows=51 width=1259) (actual time=0.166..0.172 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..381.09 rows=134 width=1259) (actual time=0.165..0.170 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..381.09 rows=134 width=1251) (actual time=0.160..0.164 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..37.46 rows=12 width=1259) (actual time=0.079..0.086 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..37.46 rows=12 width=1259) (actual time=0.077..0.083 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.46 rows=12 width=1251) (actual time=0.062..0.067 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.289..0.296 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.288..0.294 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.225..0.265 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.220..0.257 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.169..0.174 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.142..0.155 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.126..0.127 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.095..0.096 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.028..0.028 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.024..0.024 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.179 ms
Execution Time: 3.336 ms
```

### requester list (heavy requesters), page 1

Request: `GET /api/v1/records/ticket` as `req-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.059..0.063 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.039..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.444 ms
Execution Time: 0.113 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.044..0.047 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.033..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.165 ms
Execution Time: 0.084 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.061..0.063 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.060..0.061 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.050..0.054 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.041..0.044 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.010..0.011 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.280 ms
Execution Time: 0.101 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=182.55..182.56 rows=1 width=8) (actual time=1.374..1.378 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..170.03 rows=1001 width=4) (actual time=0.061..1.255 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..172.24 rows=1014 width=4) (actual time=0.059..1.125 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..162.10 rows=1014 width=0) (actual time=0.058..0.952 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..98.07 rows=1001 width=0) (actual time=0.057..0.834 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..88.06 rows=1001 width=4) (actual time=0.056..0.692 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..123.60 rows=1407 width=4) (actual time=0.056..0.540 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..123.60 rows=1407 width=0) (actual time=0.048..0.374 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..37.85 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.73 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.73 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..37.73 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.800 ms
Execution Time: 1.526 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $7)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $7)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $7)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $7
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],51]`

```
Nested Loop Left Join  (cost=22.56..180.07 rows=1 width=1301) (actual time=0.149..0.726 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..171.74 rows=1 width=1266) (actual time=0.124..0.485 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.98..163.43 rows=1 width=1259) (actual time=0.105..0.335 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.98..162.28 rows=51 width=1259) (actual time=0.102..0.287 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.98..198.04 rows=64 width=1259) (actual time=0.101..0.277 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..138.34 rows=51 width=1259) (actual time=0.048..0.212 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..3805.35 rows=1407 width=1259) (actual time=0.048..0.204 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..3805.35 rows=1407 width=1251) (actual time=0.041..0.181 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..37.73 rows=12 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..37.73 rows=12 width=1259) (actual time=0.029..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..37.73 rows=12 width=1251) (actual time=0.027..0.027 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.022..0.025 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.022..0.024 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.10 rows=1 width=1259) (actual time=0.016..0.018 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.10 rows=1 width=1251) (actual time=0.015..0.016 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.014..0.016 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.013..0.013 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.490 ms
Execution Time: 0.989 ms
```

### requester list (heavy requesters), page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTI5IDAzOjIwOjI1LjYwNTY3NSswMCIsImkiOiJkOWZlM2ZiMi02ZWU5LTdhMTktNjE1Ny0wODUxOWU2M2U3OTcifQ.NG1cX43NFhTrVxxvKOSQwoQSuyDjmnF2rmJunLq43tk` as `req-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.067..0.072 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.046..0.049 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.600 ms
Execution Time: 0.140 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.039..0.042 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.031 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.153 ms
Execution Time: 0.075 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.071..0.074 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.069..0.071 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.056..0.061 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.047..0.051 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.032..0.033 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.010..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.302 ms
Execution Time: 0.124 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=182.55..182.56 rows=1 width=8) (actual time=2.055..2.062 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..170.03 rows=1001 width=4) (actual time=0.105..1.850 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..172.24 rows=1014 width=4) (actual time=0.104..1.664 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..162.10 rows=1014 width=0) (actual time=0.103..1.359 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..98.07 rows=1001 width=0) (actual time=0.102..1.185 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..88.06 rows=1001 width=4) (actual time=0.100..0.968 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..123.60 rows=1407 width=4) (actual time=0.100..0.805 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..123.60 rows=1407 width=0) (actual time=0.088..0.553 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..37.85 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.73 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.73 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..37.73 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.813 ms
Execution Time: 2.237 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) AND ((r.updated_at, r.id) < ($7::timestamptz, $8::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) AND ((r.updated_at, r.id) < ($7::timestamptz, $8::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) AND ((r.updated_at, r.id) < ($7::timestamptz, $8::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],"2026-09-29 03:20:25.605675+00","d9fe3fb2-6ee9-7a19-6157-08519e63e797",51]`

```
Nested Loop Left Join  (cost=22.57..180.19 rows=1 width=1301) (actual time=0.136..0.662 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..171.85 rows=1 width=1266) (actual time=0.113..0.447 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.99..163.54 rows=1 width=1259) (actual time=0.097..0.314 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.99..162.39 rows=51 width=1259) (actual time=0.095..0.271 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.99..192.68 rows=62 width=1259) (actual time=0.094..0.262 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..138.53 rows=51 width=1259) (actual time=0.046..0.203 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..3152.39 rows=1164 width=1259) (actual time=0.045..0.196 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..3152.39 rows=1164 width=1251) (actual time=0.040..0.176 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..32.20 rows=10 width=1259) (actual time=0.026..0.026 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..32.20 rows=10 width=1259) (actual time=0.026..0.026 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..32.20 rows=10 width=1251) (actual time=0.023..0.023 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.021..0.023 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.021..0.023 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.11 rows=1 width=1259) (actual time=0.016..0.018 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.11 rows=1 width=1251) (actual time=0.014..0.016 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.014..0.015 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.460 ms
Execution Time: 0.894 ms
```

### admin list, default sort, page 1

Request: `GET /api/v1/records/ticket` as `admin-large`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.041..0.044 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.028..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.283 ms
Execution Time: 0.077 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.025..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.124 ms
Execution Time: 0.060 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.051..0.053 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.049..0.051 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.042..0.045 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.035..0.038 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.023..0.025 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.236 ms
Execution Time: 0.082 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69",["b52d1796-dbb9-42e5-b00b-9c6d8ea45eab"]]`

```
Aggregate  (cost=78.83..78.84 rows=1 width=8) (actual time=0.511..0.512 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.32 rows=1001 width=4) (actual time=0.046..0.433 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..12922.65 rows=196311 width=4) (actual time=0.044..0.335 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..12922.65 rows=196311 width=0) (actual time=0.034..0.192 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.201 ms
Execution Time: 0.545 ms
```

Statement 6:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $7
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69",["b52d1796-dbb9-42e5-b00b-9c6d8ea45eab"],51]`

```
Limit  (cost=1.00..100.75 rows=51 width=1293) (actual time=0.110..0.489 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..383990.33 rows=196311 width=1293) (actual time=0.109..0.479 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..382518.00 rows=196311 width=1266) (actual time=0.092..0.432 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..318273.59 rows=196311 width=1258) (actual time=0.054..0.217 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..315811.40 rows=196311 width=1251) (actual time=0.036..0.169 rows=51 loops=1)
                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.013..0.014 rows=1 loops=1)
                                Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                Filter: (id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.498 ms
Execution Time: 0.557 ms
```

### admin list, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjU5OjU5LjU2MzExOCswMCIsImkiOiJkNmRhMTEwZi1jOGM3LTEzZmItZjRmNy1jNWUzYjFkMmMzMjQifQ.-mPBiHVDRP4EqE4ixZtHNUgQsgonTWqShDntiuTJnE0` as `admin-large`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.033..0.034 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.022..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.007..0.007 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.216 ms
Execution Time: 0.061 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.044 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.162 ms
Execution Time: 0.078 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.040..0.041 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.039..0.040 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.033..0.035 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.028..0.030 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.019..0.020 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.172 ms
Execution Time: 0.063 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69",["b52d1796-dbb9-42e5-b00b-9c6d8ea45eab"]]`

```
Aggregate  (cost=78.83..78.84 rows=1 width=8) (actual time=0.384..0.386 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.32 rows=1001 width=4) (actual time=0.032..0.326 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..12922.65 rows=196311 width=4) (actual time=0.030..0.252 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..12922.65 rows=196311 width=0) (actual time=0.026..0.144 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.142 ms
Execution Time: 0.408 ms
```

Statement 6:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true AND ((r.updated_at, r.id) < ($7::timestamptz, $8::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69",["b52d1796-dbb9-42e5-b00b-9c6d8ea45eab"],"2026-09-30 23:59:59.563118+00","d6da110f-c8c7-13fb-f4f7-c5e3b1d2c324",51]`

```
Limit  (cost=1.00..100.89 rows=51 width=1293) (actual time=0.066..0.403 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..384453.41 rows=196288 width=1293) (actual time=0.065..0.396 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..382981.25 rows=196288 width=1266) (actual time=0.052..0.360 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..318744.14 rows=196288 width=1258) (actual time=0.040..0.204 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..316282.25 rows=196288 width=1251) (actual time=0.028..0.169 rows=51 loops=1)
                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:59:59.563118+00'::timestamp with time zone, 'd6da110f-c8c7-13fb-f4f7-c5e3b1d2c324'::uuid)))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.009..0.010 rows=1 loops=1)
                                Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                Filter: (id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.361 ms
Execution Time: 0.452 ms
```

### agent list, filter[state]=new

Request: `GET /api/v1/records/ticket?filter%5Bstate%5D=new` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.048..0.051 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.032..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.353 ms
Execution Time: 0.091 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.038..0.041 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.140 ms
Execution Time: 0.076 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.061..0.063 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.060..0.061 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.050..0.054 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.041..0.044 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.293 ms
Execution Time: 0.104 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=531.59..531.60 rows=1 width=8) (actual time=1.493..1.499 rows=1 loops=1)
  Buffers: shared hit=539
  ->  Limit  (cost=0.55..527.96 rows=290 width=4) (actual time=0.063..1.365 rows=1001 loops=1)
        Buffers: shared hit=539
        ->  Result  (cost=0.55..527.96 rows=290 width=4) (actual time=0.062..1.228 rows=1001 loops=1)
              Buffers: shared hit=539
              ->  Append  (cost=0.55..525.06 rows=290 width=0) (actual time=0.061..1.044 rows=1001 loops=1)
                    Buffers: shared hit=539
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..44.72 rows=281 width=0) (actual time=0.060..0.917 rows=1001 loops=1)
                          Buffers: shared hit=539
                          ->  Limit  (cost=0.55..41.91 rows=281 width=4) (actual time=0.059..0.769 rows=1001 loops=1)
                                Buffers: shared hit=539
                                ->  Result  (cost=0.55..41.91 rows=281 width=4) (actual time=0.059..0.651 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=539
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..41.91 rows=281 width=0) (actual time=0.048..0.467 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=539
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..396.59 rows=7 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.52 rows=7 width=4) (never executed)
                                ->  Result  (cost=0.42..396.52 rows=7 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.52 rows=7 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.81 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.80 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..37.80 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.80 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.133 ms
Execution Time: 1.675 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=837.02..1223.56 rows=1 width=1301) (actual time=4.720..5.229 rows=51 loops=1)
  Buffers: shared hit=1601
  ->  Nested Loop  (cost=836.72..1215.23 rows=1 width=1266) (actual time=4.687..5.013 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1448
        ->  Subquery Scan on x  (cost=836.44..1206.92 rows=1 width=1259) (actual time=4.667..4.857 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1295
              ->  Limit  (cost=836.44..1205.77 rows=51 width=1259) (actual time=4.663..4.805 rows=51 loops=1)
                    Buffers: shared hit=1295
                    ->  Merge Append  (cost=836.44..1270.95 rows=60 width=1259) (actual time=4.661..4.796 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1295
                          ->  Limit  (cost=791.06..791.19 rows=51 width=1259) (actual time=4.307..4.330 rows=50 loops=1)
                                Buffers: shared hit=1204
                                ->  Sort  (cost=791.06..791.77 rows=281 width=1259) (actual time=4.305..4.321 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=1204
                                      ->  Result  (cost=0.55..781.69 rows=281 width=1259) (actual time=0.060..3.212 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=1204
                                            ->  Index Scan using records_list_group_state on records r  (cost=0.55..781.69 rows=281 width=1251) (actual time=0.052..2.833 rows=1168 loops=1)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                  Buffers: shared hit=1204
                          ->  Limit  (cost=0.42..396.52 rows=7 width=1259) (actual time=0.115..0.197 rows=2 loops=1)
                                Buffers: shared hit=46
                                ->  Result  (cost=0.42..396.52 rows=7 width=1259) (actual time=0.112..0.194 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=46
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.52 rows=7 width=1251) (actual time=0.099..0.179 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=46
                          ->  Limit  (cost=0.42..37.80 rows=1 width=1259) (actual time=0.070..0.071 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..37.80 rows=1 width=1259) (actual time=0.068..0.069 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.80 rows=1 width=1251) (actual time=0.062..0.062 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.161..0.164 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.161..0.164 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.136..0.150 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.131..0.145 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.069..0.073 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.050..0.058 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.038..0.039 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.020 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.016 ms
Execution Time: 5.561 ms
```

### agent list, filter[priority]=1 (data)

Request: `GET /api/v1/records/ticket?filter%5Bpriority%5D=1` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.045..0.048 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.030..0.033 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.316 ms
Execution Time: 0.084 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.025..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.121 ms
Execution Time: 0.061 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.058..0.060 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.056..0.057 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.046..0.049 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.038..0.040 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.025..0.026 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.304 ms
Execution Time: 0.097 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.021 ms
Execution Time: 0.022 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}"]`

```
Aggregate  (cost=15755.99..15756.00 rows=1 width=8) (actual time=44.008..44.017 rows=1 loops=1)
  Buffers: shared hit=20546
  ->  Limit  (cost=0.55..15755.24 rows=60 width=4) (actual time=0.084..43.827 rows=1001 loops=1)
        Buffers: shared hit=20546
        ->  Result  (cost=0.55..15755.24 rows=60 width=4) (actual time=0.083..43.657 rows=1001 loops=1)
              Buffers: shared hit=20546
              ->  Append  (cost=0.55..15754.64 rows=60 width=0) (actual time=0.083..43.401 rows=1001 loops=1)
                    Buffers: shared hit=20546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..15275.32 rows=57 width=0) (actual time=0.082..43.213 rows=1001 loops=1)
                          Buffers: shared hit=20546
                          ->  Limit  (cost=0.55..15274.75 rows=57 width=4) (actual time=0.081..43.018 rows=1001 loops=1)
                                Buffers: shared hit=20546
                                ->  Result  (cost=0.55..15274.75 rows=57 width=4) (actual time=0.081..42.869 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=20546
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..15274.75 rows=57 width=0) (actual time=0.074..42.603 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"priority": "1"}'::jsonb)
                                            Rows Removed by Filter: 19225
                                            Buffers: shared hit=20546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..396.71 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.70 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..396.70 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.83 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.82 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..37.82 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.011 ms
Execution Time: 44.139 ms
```

Statement 8:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}",51]`

```
Nested Loop Left Join  (cost=15322.37..15750.05 rows=1 width=1301) (actual time=51.688..52.134 rows=51 loops=1)
  Buffers: shared hit=24863
  ->  Nested Loop  (cost=15322.07..15741.71 rows=1 width=1266) (actual time=51.598..51.859 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=24710
        ->  Subquery Scan on x  (cost=15321.79..15733.40 rows=1 width=1259) (actual time=51.580..51.711 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=24557
              ->  Limit  (cost=15321.79..15732.25 rows=51 width=1259) (actual time=51.577..51.662 rows=51 loops=1)
                    Buffers: shared hit=24557
                    ->  Merge Append  (cost=15321.79..15756.40 rows=54 width=1259) (actual time=51.576..51.654 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=24557
                          ->  Limit  (cost=15276.41..15276.54 rows=51 width=1259) (actual time=51.287..51.307 rows=50 loops=1)
                                Buffers: shared hit=24492
                                ->  Sort  (cost=15276.41..15276.55 rows=57 width=1259) (actual time=51.285..51.299 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=24492
                                      ->  Result  (cost=0.55..15274.75 rows=57 width=1259) (actual time=0.097..50.084 rows=1182 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=24492
                                            ->  Index Scan using records_list_group on records r  (cost=0.55..15274.75 rows=57 width=1251) (actual time=0.089..49.655 rows=1182 loops=1)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"priority": "1"}'::jsonb)
                                                  Rows Removed by Filter: 22936
                                                  Buffers: shared hit=24492
                          ->  Limit  (cost=0.42..396.70 rows=1 width=1259) (actual time=0.120..0.121 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.42..396.70 rows=1 width=1259) (actual time=0.117..0.118 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=23
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=1 width=1251) (actual time=0.102..0.103 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 19
                                            Buffers: shared hit=23
                          ->  Limit  (cost=0.42..37.82 rows=1 width=1259) (actual time=0.038..0.070 rows=2 loops=1)
                                Buffers: shared hit=17
                                ->  Result  (cost=0.42..37.82 rows=1 width=1259) (actual time=0.038..0.069 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=17
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=1 width=1251) (actual time=0.033..0.064 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 12
                                            Buffers: shared hit=17
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.124..0.127 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.124..0.126 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.112..0.114 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.110..0.111 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.067..0.069 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.046..0.054 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.035..0.035 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.019 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.743 ms
Execution Time: 52.500 ms
```

### agent list, filter[site]=s7 (rare data, < 1% of visible rows)

Request: `GET /api/v1/records/ticket?filter%5Bsite%5D=s7` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.043..0.046 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.028..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.329 ms
Execution Time: 0.084 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.031..0.034 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.024..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.120 ms
Execution Time: 0.059 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.063..0.065 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.062..0.063 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.051..0.055 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.043..0.046 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.011..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.254 ms
Execution Time: 0.111 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.007..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.017 ms
Execution Time: 0.020 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}"]`

```
Aggregate  (cost=15755.99..15756.00 rows=1 width=8) (actual time=45.069..45.078 rows=1 loops=1)
  Buffers: shared hit=25107
  ->  Limit  (cost=0.55..15755.24 rows=60 width=4) (actual time=0.257..45.030 rows=121 loops=1)
        Buffers: shared hit=25107
        ->  Result  (cost=0.55..15755.24 rows=60 width=4) (actual time=0.255..44.997 rows=121 loops=1)
              Buffers: shared hit=25107
              ->  Append  (cost=0.55..15754.64 rows=60 width=0) (actual time=0.254..44.955 rows=121 loops=1)
                    Buffers: shared hit=25107
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..15275.32 rows=57 width=0) (actual time=0.253..43.716 rows=120 loops=1)
                          Buffers: shared hit=24492
                          ->  Limit  (cost=0.55..15274.75 rows=57 width=4) (actual time=0.253..43.676 rows=120 loops=1)
                                Buffers: shared hit=24492
                                ->  Result  (cost=0.55..15274.75 rows=57 width=4) (actual time=0.252..43.645 rows=120 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=24492
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..15274.75 rows=57 width=0) (actual time=0.245..43.595 rows=120 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"site": "s7"}'::jsonb)
                                            Rows Removed by Filter: 23998
                                            Buffers: shared hit=24492
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..396.71 rows=1 width=0) (actual time=0.373..0.929 rows=1 loops=1)
                          Buffers: shared hit=523
                          ->  Limit  (cost=0.42..396.70 rows=1 width=4) (actual time=0.372..0.927 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..396.70 rows=1 width=4) (actual time=0.370..0.924 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=1 width=0) (actual time=0.359..0.912 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.83 rows=1 width=0) (actual time=0.168..0.169 rows=0 loops=1)
                          Buffers: shared hit=67
                          ->  Limit  (cost=0.42..37.82 rows=1 width=4) (actual time=0.168..0.168 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..37.82 rows=1 width=4) (actual time=0.166..0.167 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=1 width=0) (actual time=0.160..0.160 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (actual time=0.099..0.102 rows=0 loops=1)
                          Buffers: shared hit=25
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (actual time=0.098..0.101 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (actual time=0.098..0.100 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=25
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (actual time=0.095..0.097 rows=0 loops=1)
                                            Buffers: shared hit=25
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.054..0.057 rows=3 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=13
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.046 rows=3 loops=1)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Heap Blocks: exact=3
                                                        Buffers: shared hit=13
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                                                              Buffers: shared hit=10
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=8
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.256 ms
Execution Time: 45.225 ms
```

Statement 8:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}",51]`

```
Nested Loop Left Join  (cost=15322.37..15750.05 rows=1 width=1301) (actual time=42.187..43.147 rows=51 loops=1)
  Buffers: shared hit=25413
  ->  Nested Loop  (cost=15322.07..15741.71 rows=1 width=1266) (actual time=42.163..42.962 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=25260
        ->  Subquery Scan on x  (cost=15321.79..15733.40 rows=1 width=1259) (actual time=42.148..42.803 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=25107
              ->  Limit  (cost=15321.79..15732.25 rows=51 width=1259) (actual time=42.145..42.757 rows=51 loops=1)
                    Buffers: shared hit=25107
                    ->  Merge Append  (cost=15321.79..15756.40 rows=54 width=1259) (actual time=42.144..42.749 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=25107
                          ->  Limit  (cost=15276.41..15276.54 rows=51 width=1259) (actual time=41.540..41.563 rows=50 loops=1)
                                Buffers: shared hit=24492
                                ->  Sort  (cost=15276.41..15276.55 rows=57 width=1259) (actual time=41.538..41.555 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 187kB
                                      Buffers: shared hit=24492
                                      ->  Result  (cost=0.55..15274.75 rows=57 width=1259) (actual time=0.194..41.115 rows=120 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=24492
                                            ->  Index Scan using records_list_group on records r  (cost=0.55..15274.75 rows=57 width=1251) (actual time=0.189..41.046 rows=120 loops=1)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"site": "s7"}'::jsonb)
                                                  Rows Removed by Filter: 23998
                                                  Buffers: shared hit=24492
                          ->  Limit  (cost=0.42..396.70 rows=1 width=1259) (actual time=0.352..0.911 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..396.70 rows=1 width=1259) (actual time=0.350..0.908 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=1 width=1251) (actual time=0.338..0.895 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                          ->  Limit  (cost=0.42..37.82 rows=1 width=1259) (actual time=0.136..0.137 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..37.82 rows=1 width=1259) (actual time=0.135..0.136 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=1 width=1251) (actual time=0.132..0.132 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.110..0.114 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.109..0.113 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.100..0.103 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.098..0.100 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.061..0.064 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.044..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.018..0.018 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.598 ms
Execution Time: 43.423 ms
```

### agent list, filter[site]=s7, short data-filter timeout (bound check)

Request: `GET /api/v1/records/ticket?filter%5Bsite%5D=s7` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.039..0.042 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.025..0.027 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.305 ms
Execution Time: 0.078 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.032 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.022..0.024 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.121 ms
Execution Time: 0.058 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.051..0.053 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.050..0.052 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.040..0.044 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.034..0.037 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.022..0.023 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.239 ms
Execution Time: 0.085 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.007 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.017 ms
Execution Time: 0.017 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}"]`

```
Aggregate  (cost=15755.99..15756.00 rows=1 width=8) (actual time=45.001..45.009 rows=1 loops=1)
  Buffers: shared hit=25107
  ->  Limit  (cost=0.55..15755.24 rows=60 width=4) (actual time=0.262..44.960 rows=121 loops=1)
        Buffers: shared hit=25107
        ->  Result  (cost=0.55..15755.24 rows=60 width=4) (actual time=0.261..44.927 rows=121 loops=1)
              Buffers: shared hit=25107
              ->  Append  (cost=0.55..15754.64 rows=60 width=0) (actual time=0.260..44.883 rows=121 loops=1)
                    Buffers: shared hit=25107
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..15275.32 rows=57 width=0) (actual time=0.259..43.583 rows=120 loops=1)
                          Buffers: shared hit=24492
                          ->  Limit  (cost=0.55..15274.75 rows=57 width=4) (actual time=0.259..43.539 rows=120 loops=1)
                                Buffers: shared hit=24492
                                ->  Result  (cost=0.55..15274.75 rows=57 width=4) (actual time=0.258..43.508 rows=120 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=24492
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..15274.75 rows=57 width=0) (actual time=0.252..43.457 rows=120 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"site": "s7"}'::jsonb)
                                            Rows Removed by Filter: 23998
                                            Buffers: shared hit=24492
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..396.71 rows=1 width=0) (actual time=0.364..1.020 rows=1 loops=1)
                          Buffers: shared hit=523
                          ->  Limit  (cost=0.42..396.70 rows=1 width=4) (actual time=0.363..1.018 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..396.70 rows=1 width=4) (actual time=0.361..1.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=1 width=0) (actual time=0.348..1.001 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.83 rows=1 width=0) (actual time=0.138..0.139 rows=0 loops=1)
                          Buffers: shared hit=67
                          ->  Limit  (cost=0.42..37.82 rows=1 width=4) (actual time=0.138..0.138 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..37.82 rows=1 width=4) (actual time=0.136..0.137 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=1 width=0) (actual time=0.129..0.129 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (actual time=0.099..0.101 rows=0 loops=1)
                          Buffers: shared hit=25
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (actual time=0.098..0.100 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (actual time=0.097..0.099 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=25
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (actual time=0.095..0.097 rows=0 loops=1)
                                            Buffers: shared hit=25
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.058 rows=3 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=13
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.047 rows=3 loops=1)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Heap Blocks: exact=3
                                                        Buffers: shared hit=13
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.029 rows=0 loops=1)
                                                              Buffers: shared hit=10
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=8
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.006 ms
Execution Time: 45.121 ms
```

### agent list, sort=-updated_at

Request: `GET /api/v1/records/ticket?sort=-updated_at` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.046..0.049 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.031..0.033 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.351 ms
Execution Time: 0.090 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.035..0.038 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.026..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.145 ms
Execution Time: 0.068 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.062..0.064 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.061..0.062 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.050..0.054 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.041..0.045 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.010..0.011 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.282 ms
Execution Time: 0.103 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=528.50..528.51 rows=1 width=8) (actual time=1.622..1.630 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..515.99 rows=1001 width=4) (actual time=0.058..1.487 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..594.78 rows=1154 width=4) (actual time=0.056..1.339 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..583.24 rows=1154 width=0) (actual time=0.056..1.121 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.33 rows=1001 width=0) (actual time=0.055..0.979 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.32 rows=1001 width=4) (actual time=0.054..0.819 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.51 rows=5733 width=4) (actual time=0.053..0.688 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.51 rows=5733 width=0) (actual time=0.042..0.497 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..397.74 rows=140 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                ->  Result  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.34 rows=140 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.91 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.79 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.108 ms
Execution Time: 1.759 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1394.68 rows=1 width=1301) (actual time=0.373..3.257 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1386.34 rows=1 width=1266) (actual time=0.350..2.946 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1378.03 rows=1 width=1259) (actual time=0.335..2.699 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1376.88 rows=51 width=1259) (actual time=0.332..2.600 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3047.26 rows=115 width=1259) (actual time=0.330..2.585 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..2818.57 rows=51 width=1259) (actual time=0.146..2.360 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..316792.96 rows=5733 width=1259) (actual time=0.145..2.347 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..316792.96 rows=5733 width=1251) (actual time=0.140..2.314 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..144.65 rows=51 width=1259) (actual time=0.024..0.031 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..396.34 rows=140 width=1259) (actual time=0.024..0.030 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.34 rows=140 width=1251) (actual time=0.020..0.025 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..37.79 rows=12 width=1259) (actual time=0.023..0.025 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..37.79 rows=12 width=1259) (actual time=0.022..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.79 rows=12 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.135..0.142 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.134..0.141 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.088..0.127 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.086..0.122 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.059..0.065 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.041..0.052 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.031..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.017..0.017 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.817 ms
Execution Time: 3.509 ms
```

### agent list, sort=assignee

Request: `GET /api/v1/records/ticket?sort=assignee` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.060..0.064 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.037..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.013..0.013 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.436 ms
Execution Time: 0.118 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.039..0.042 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.030..0.032 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.158 ms
Execution Time: 0.077 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.051..0.052 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.050..0.051 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.041..0.044 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.035..0.038 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.024..0.025 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.228 ms
Execution Time: 0.088 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=528.50..528.51 rows=1 width=8) (actual time=1.291..1.296 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..515.99 rows=1001 width=4) (actual time=0.053..1.178 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..594.78 rows=1154 width=4) (actual time=0.051..1.053 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..583.24 rows=1154 width=0) (actual time=0.051..0.883 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.33 rows=1001 width=0) (actual time=0.050..0.765 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.32 rows=1001 width=4) (actual time=0.049..0.630 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.51 rows=5733 width=4) (actual time=0.049..0.521 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.51 rows=5733 width=0) (actual time=0.039..0.358 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..397.74 rows=140 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                ->  Result  (cost=0.42..396.34 rows=140 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.34 rows=140 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.91 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.79 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.79 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.170 ms
Execution Time: 1.469 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
ORDER BY sort_key ASC NULLS LAST, updated_at ASC, id ASC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key ASC NULLS LAST, x.updated_at ASC, x.id ASC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=484.55..1761.14 rows=1 width=1309) (actual time=2.211..3.872 rows=51 loops=1)
  Buffers: shared hit=1718
  ->  Nested Loop  (cost=484.25..1752.81 rows=1 width=1274) (actual time=2.193..3.689 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1565
        ->  Subquery Scan on x  (cost=483.97..1744.49 rows=1 width=1267) (actual time=2.177..3.553 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1412
              ->  Limit  (cost=483.97..1743.35 rows=51 width=1267) (actual time=2.174..3.506 rows=51 loops=1)
                    Buffers: shared hit=1412
                    ->  Merge Append  (cost=483.97..3323.75 rows=115 width=1267) (actual time=2.173..3.497 rows=51 loops=1)
                          Sort Key: r.assignee_id, r.updated_at, r.id
                          Buffers: shared hit=1412
                          ->  Limit  (cost=0.42..2838.32 rows=51 width=1267) (actual time=0.062..1.364 rows=50 loops=1)
                                Buffers: shared hit=808
                                ->  Result  (cost=0.42..319012.96 rows=5733 width=1267) (actual time=0.062..1.356 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=808
                                      ->  Index Scan Backward using records_list_assignee on records r  (cost=0.42..319012.96 rows=5733 width=1251) (actual time=0.057..1.336 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 727
                                            Buffers: shared hit=808
                          ->  Limit  (cost=401.01..401.14 rows=51 width=1267) (actual time=1.737..1.739 rows=2 loops=1)
                                Buffers: shared hit=523
                                ->  Sort  (cost=401.01..401.36 rows=140 width=1267) (actual time=1.735..1.737 rows=2 loops=1)
                                      Sort Key: r_1.assignee_id, r_1.updated_at, r_1.id
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=523
                                      ->  Result  (cost=0.42..396.34 rows=140 width=1267) (actual time=0.024..1.252 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=523
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.34 rows=140 width=1251) (actual time=0.022..1.125 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                          ->  Sort  (cost=38.00..38.03 rows=12 width=1267) (actual time=0.264..0.266 rows=1 loops=1)
                                Sort Key: r_2.assignee_id, r_2.updated_at, r_2.id
                                Sort Method: quicksort  Memory: 127kB
                                Buffers: shared hit=56
                                ->  Limit  (cost=0.42..37.79 rows=12 width=1267) (actual time=0.029..0.157 rows=51 loops=1)
                                      Buffers: shared hit=56
                                      ->  Result  (cost=0.42..37.79 rows=12 width=1267) (actual time=0.028..0.149 rows=51 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=56
                                            ->  Index Scan Backward using records_list_assignee on records r_2  (cost=0.42..37.79 rows=12 width=1251) (actual time=0.021..0.129 rows=51 loops=1)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=56
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1267) (actual time=0.106..0.109 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1267) (actual time=0.105..0.108 rows=1 loops=1)
                                      Sort Key: r_3.assignee_id, r_3.updated_at, r_3.id
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1267) (actual time=0.078..0.101 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.075..0.097 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.056 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.047 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.354 ms
Execution Time: 4.039 ms
```

### admin list, filter[state][in]=new,in_progress, sort=-updated_at

Request: `GET /api/v1/records/ticket?filter%5Bstate%5D%5Bin%5D=new,in_progress&sort=-updated_at` as `admin-large`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.056..0.060 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.038..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.470 ms
Execution Time: 0.112 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.045 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.031..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.160 ms
Execution Time: 0.081 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.049..0.051 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.048..0.049 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.034..0.037 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.029..0.032 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.019..0.020 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.357 ms
Execution Time: 0.094 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND r.state = ANY($7::text[]) LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69",["b52d1796-dbb9-42e5-b00b-9c6d8ea45eab"],["new","in_progress"]]`

```
Aggregate  (cost=90.88..90.89 rows=1 width=8) (actual time=0.551..0.552 rows=1 loops=1)
  Buffers: shared hit=532
  ->  Limit  (cost=0.42..78.37 rows=1001 width=4) (actual time=0.041..0.486 rows=1001 loops=1)
        Buffers: shared hit=532
        ->  Result  (cost=0.42..1491.80 rows=19153 width=4) (actual time=0.041..0.404 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=532
              ->  Index Only Scan using records_list_state on records r  (cost=0.42..1491.80 rows=19153 width=0) (actual time=0.036..0.287 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (state = ANY ('{new,in_progress}'::text[])))
                    Heap Fetches: 0
                    Buffers: shared hit=532
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.164 ms
Execution Time: 0.577 ms
```

Statement 6:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND r.state = ANY($7::text[]) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69","872ab354-0140-4267-b4b9-6d16809b7b69",["b52d1796-dbb9-42e5-b00b-9c6d8ea45eab"],["new","in_progress"],51]`

```
Limit  (cost=1.00..865.57 rows=51 width=1293) (actual time=0.083..1.526 rows=51 loops=1)
  Buffers: shared hit=635
  ->  Result  (cost=1.00..324690.50 rows=19153 width=1293) (actual time=0.082..1.513 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Buffers: shared hit=635
        ->  Nested Loop Left Join  (cost=1.00..324546.85 rows=19153 width=1266) (actual time=0.071..1.454 rows=51 loops=1)
              Buffers: shared hit=635
              ->  Nested Loop  (cost=0.70..316549.89 rows=19153 width=1258) (actual time=0.058..1.187 rows=51 loops=1)
                    Buffers: shared hit=482
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..316302.18 rows=19153 width=1251) (actual time=0.044..1.125 rows=51 loops=1)
                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                          Filter: (state = ANY ('{new,in_progress}'::text[]))
                          Rows Removed by Filter: 419
                          Buffers: shared hit=479
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.009..0.010 rows=1 loops=1)
                                Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                Filter: (id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.41 rows=1 width=40) (actual time=0.004..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.41 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.387 ms
Execution Time: 1.588 ms
```

### small tenant: agent list, page 1

Request: `GET /api/v1/records/ticket` as `agent-s1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.098..0.102 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.074..0.077 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.013..0.013 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.450 ms
Execution Time: 0.155 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.030..0.033 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.024..0.025 rows=1 loops=1)
        Index Cond: (tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.116 ms
Execution Time: 0.056 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.074..0.076 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.072..0.074 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.060..0.064 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.049..0.053 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.033..0.034 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.012..0.014 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.327 ms
Execution Time: 0.123 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","041eeb2e-2606-4cb2-8c7b-b6d6fd26d741",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],"4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"]]`

```
Aggregate  (cost=34.43..34.44 rows=1 width=8) (actual time=3.968..3.980 rows=1 loops=1)
  Buffers: shared hit=4704
  ->  Limit  (cost=0.55..34.38 rows=4 width=4) (actual time=0.123..3.947 rows=179 loops=1)
        Buffers: shared hit=4704
        ->  Result  (cost=0.55..34.38 rows=4 width=4) (actual time=0.087..3.885 rows=179 loops=1)
              Buffers: shared hit=4704
              ->  Append  (cost=0.55..34.34 rows=4 width=0) (actual time=0.086..3.848 rows=179 loops=1)
                    Buffers: shared hit=4704
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.59 rows=1 width=0) (actual time=0.086..0.325 rows=50 loops=1)
                          Buffers: shared hit=21
                          ->  Limit  (cost=0.55..4.58 rows=1 width=4) (actual time=0.084..0.316 rows=50 loops=1)
                                Buffers: shared hit=21
                                ->  Result  (cost=0.55..4.58 rows=1 width=4) (actual time=0.084..0.309 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=21
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.58 rows=1 width=0) (actual time=0.073..0.290 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid) AND (assignment_group_id IS NOT NULL))
                                            Filter: (assignment_group_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))
                                            Rows Removed by Filter: 953
                                            Heap Fetches: 0
                                            Buffers: shared hit=21
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..8.46 rows=1 width=0) (actual time=0.026..0.818 rows=97 loops=1)
                          Buffers: shared hit=1010
                          ->  Limit  (cost=0.42..8.45 rows=1 width=4) (actual time=0.026..0.802 rows=97 loops=1)
                                Buffers: shared hit=1010
                                ->  Result  (cost=0.42..8.45 rows=1 width=4) (actual time=0.025..0.789 rows=97 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=1010
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..8.45 rows=1 width=0) (actual time=0.023..0.770 rows=97 loops=1)
                                            Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                                            Filter: ((requester_id IS NOT NULL) AND (requester_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 906
                                            Buffers: shared hit=1010
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..8.46 rows=1 width=0) (actual time=0.027..0.600 rows=32 loops=1)
                          Buffers: shared hit=1014
                          ->  Limit  (cost=0.42..8.46 rows=1 width=4) (actual time=0.027..0.593 rows=32 loops=1)
                                Buffers: shared hit=1014
                                ->  Result  (cost=0.42..8.46 rows=1 width=4) (actual time=0.026..0.588 rows=32 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=1014
                                      ->  Index Scan using records_list_requester on records r_2  (cost=0.42..8.46 rows=1 width=0) (actual time=0.023..0.579 rows=32 loops=1)
                                            Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                                            Filter: ((assignee_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND (assignee_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 971
                                            Buffers: shared hit=1014
                    ->  Subquery Scan on "*SELECT* 4"  (cost=0.72..12.80 rows=1 width=0) (actual time=2.072..2.073 rows=0 loops=1)
                          Buffers: shared hit=2659
                          ->  Limit  (cost=0.72..12.79 rows=1 width=4) (actual time=2.070..2.072 rows=0 loops=1)
                                Buffers: shared hit=2659
                                ->  Result  (cost=0.72..12.79 rows=1 width=4) (actual time=2.069..2.070 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=2659
                                      ->  Nested Loop Semi Join  (cost=0.72..12.79 rows=1 width=0) (actual time=2.066..2.067 rows=0 loops=1)
                                            Buffers: shared hit=2659
                                            ->  Index Scan using records_list_assignee on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.018..0.772 rows=824 loops=1)
                                                  Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 179
                                                  Buffers: shared hit=1010
                                            ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.32 rows=1 width=32) (actual time=0.001..0.001 rows=0 loops=824)
                                                  Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (record_id = r_3.id))
                                                  Filter: (((principal_kind = 'user'::text) AND (principal_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 0
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1649
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.033 ms
Execution Time: 4.083 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","041eeb2e-2606-4cb2-8c7b-b6d6fd26d741",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],"4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],51]`

```
Nested Loop Left Join  (cost=2.62..54.99 rows=1 width=1301) (actual time=2.588..3.617 rows=51 loops=1)
  Buffers: shared hit=3849
  ->  Nested Loop  (cost=2.32..46.66 rows=1 width=1266) (actual time=2.565..3.368 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3696
        ->  Subquery Scan on x  (cost=2.04..38.34 rows=1 width=1259) (actual time=2.548..3.207 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=3543
              ->  Limit  (cost=2.04..38.25 rows=4 width=1259) (actual time=2.540..3.148 rows=51 loops=1)
                    Buffers: shared hit=3543
                    ->  Merge Append  (cost=2.04..38.25 rows=4 width=1259) (actual time=2.538..3.137 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=3543
                          ->  Limit  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.093..0.277 rows=14 loops=1)
                                Buffers: shared hit=295
                                ->  Result  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.092..0.273 rows=14 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=295
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.087..0.262 rows=14 loops=1)
                                            Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[])))
                                            Rows Removed by Filter: 278
                                            Buffers: shared hit=295
                          ->  Limit  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.020..0.258 rows=30 loops=1)
                                Buffers: shared hit=292
                                ->  Result  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.020..0.253 rows=30 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=292
                                      ->  Index Scan using records_list_updated on records r_1  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.016..0.238 rows=30 loops=1)
                                            Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                                            Filter: ((requester_id IS NOT NULL) AND (requester_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 259
                                            Buffers: shared hit=292
                          ->  Limit  (cost=0.42..8.46 rows=1 width=1259) (actual time=0.036..0.196 rows=9 loops=1)
                                Buffers: shared hit=297
                                ->  Result  (cost=0.42..8.46 rows=1 width=1259) (actual time=0.035..0.193 rows=9 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=297
                                      ->  Index Scan using records_list_updated on records r_2  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.034..0.188 rows=9 loops=1)
                                            Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                                            Filter: ((assignee_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND (assignee_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 285
                                            Buffers: shared hit=297
                          ->  Limit  (cost=0.72..12.79 rows=1 width=1259) (actual time=2.387..2.388 rows=0 loops=1)
                                Buffers: shared hit=2659
                                ->  Result  (cost=0.72..12.79 rows=1 width=1259) (actual time=2.386..2.387 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                                      Buffers: shared hit=2659
                                      ->  Nested Loop Semi Join  (cost=0.72..12.79 rows=1 width=1251) (actual time=2.383..2.384 rows=0 loops=1)
                                            Buffers: shared hit=2659
                                            ->  Index Scan using records_list_updated on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.008..1.062 rows=824 loops=1)
                                                  Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 179
                                                  Buffers: shared hit=1010
                                            ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.32 rows=1 width=32) (actual time=0.001..0.001 rows=0 loops=824)
                                                  Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (record_id = r_3.id))
                                                  Filter: (((principal_kind = 'user'::text) AND (principal_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 0
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1649
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.502 ms
Execution Time: 3.791 ms
```

### small tenant: admin list, page 1

Request: `GET /api/v1/records/ticket` as `admin-s1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.062..0.066 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.040..0.043 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.500 ms
Execution Time: 0.119 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.044 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.031..0.033 rows=1 loops=1)
        Index Cond: (tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.158 ms
Execution Time: 0.080 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.072..0.074 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.070..0.072 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.057..0.062 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.047..0.051 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.032..0.033 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.011..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.394 ms
Execution Time: 0.121 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","041eeb2e-2606-4cb2-8c7b-b6d6fd26d741","ffd85c5b-42e7-4153-b3f4-a5c1679534e7","ffd85c5b-42e7-4153-b3f4-a5c1679534e7","ffd85c5b-42e7-4153-b3f4-a5c1679534e7",["9f1436e3-4a7e-478a-afeb-f01a0f186a75"]]`

```
Aggregate  (cost=4.46..4.47 rows=1 width=8) (actual time=0.958..0.960 rows=1 loops=1)
  Buffers: shared hit=18
  ->  Limit  (cost=0.42..4.44 rows=1 width=4) (actual time=0.061..0.777 rows=1001 loops=1)
        Buffers: shared hit=18
        ->  Result  (cost=0.42..4.44 rows=1 width=4) (actual time=0.059..0.579 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
              Buffers: shared hit=18
              ->  Index Only Scan using records_list_assignee on records r  (cost=0.42..4.44 rows=1 width=0) (actual time=0.049..0.333 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=18
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.339 ms
Execution Time: 1.008 ms
```

Statement 6:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $7
```

Arguments: `["2ca3b969-a46c-4aaa-bf47-8829c4f80b1d","041eeb2e-2606-4cb2-8c7b-b6d6fd26d741","ffd85c5b-42e7-4153-b3f4-a5c1679534e7","ffd85c5b-42e7-4153-b3f4-a5c1679534e7","ffd85c5b-42e7-4153-b3f4-a5c1679534e7",["9f1436e3-4a7e-478a-afeb-f01a0f186a75"],51]`

```
Limit  (cost=1.00..25.09 rows=1 width=1293) (actual time=0.065..0.403 rows=51 loops=1)
  Buffers: shared hit=360
  ->  Result  (cost=1.00..25.09 rows=1 width=1293) (actual time=0.064..0.396 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
        Buffers: shared hit=360
        ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.051..0.357 rows=51 loops=1)
              Buffers: shared hit=360
              ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.040..0.189 rows=51 loops=1)
                    Buffers: shared hit=207
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.028..0.099 rows=51 loops=1)
                          Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (entity_definition_id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid))
                          Buffers: shared hit=54
                    ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Index Cond: (tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                          Filter: (id = '041eeb2e-2606-4cb2-8c7b-b6d6fd26d741'::uuid)
                          Buffers: shared hit=153
              ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = '2ca3b969-a46c-4aaa-bf47-8829c4f80b1d'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.694 ms
Execution Time: 0.465 ms
```

### single GET, agent

Request: `GET /api/v1/records/ticket/3f94a2ac-38ee-af8f-f434-143a689a293d` as `agent-1`

### single GET, admin

Request: `GET /api/v1/records/ticket/7af47606-284d-7369-c4d9-ac496c0f82c8` as `admin-large`

### single GET, small tenant agent

Request: `GET /api/v1/records/ticket/c7df223c-cbb4-f467-776f-6fea3db8bf06` as `agent-s1`

### composed: agent list by group, default sort, page 1

Request: `GET /api/v1/records/ticket` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.041..0.043 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.027..0.029 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.293 ms
Execution Time: 0.076 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.031 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.022..0.024 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.110 ms
Execution Time: 0.068 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.054..0.056 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.053..0.054 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.040..0.046 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.034..0.040 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.023..0.024 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.012 rows=2 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.220 ms
Execution Time: 0.086 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=2768.53..2768.54 rows=1 width=8) (actual time=3.910..3.920 rows=1 loops=1)
  Buffers: shared hit=1023
  ->  Limit  (cost=0.55..2756.02 rows=1001 width=4) (actual time=0.062..3.722 rows=1001 loops=1)
        Buffers: shared hit=1023
        ->  Result  (cost=0.55..3177.18 rows=1154 width=4) (actual time=0.060..3.527 rows=1001 loops=1)
              Buffers: shared hit=1023
              ->  Append  (cost=0.55..3165.64 rows=1154 width=0) (actual time=0.060..3.265 rows=1001 loops=1)
                    Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..2679.35 rows=1001 width=0) (actual time=0.058..3.054 rows=1001 loops=1)
                          Buffers: shared hit=1023
                          ->  Limit  (cost=0.55..2669.34 rows=1001 width=4) (actual time=0.057..2.863 rows=1001 loops=1)
                                Buffers: shared hit=1023
                                ->  Result  (cost=0.55..15274.75 rows=5729 width=4) (actual time=0.056..2.706 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=1023
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..15274.75 rows=5729 width=0) (actual time=0.045..2.471 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 1
                                            Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..398.10 rows=140 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.70 rows=140 width=4) (never executed)
                                ->  Result  (cost=0.42..396.70 rows=140 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=140 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.94 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.82 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.82 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.585 ms
Execution Time: 4.075 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1397.56 rows=1 width=1301) (actual time=0.433..3.154 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1389.23 rows=1 width=1266) (actual time=0.406..2.854 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1380.92 rows=1 width=1259) (actual time=0.390..2.587 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1379.77 rows=51 width=1259) (actual time=0.387..2.509 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3053.76 rows=115 width=1259) (actual time=0.385..2.493 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..2824.91 rows=51 width=1259) (actual time=0.180..2.253 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..317283.74 rows=5729 width=1259) (actual time=0.179..2.240 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..317283.74 rows=5729 width=1251) (actual time=0.171..2.205 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..144.78 rows=51 width=1259) (actual time=0.027..0.033 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..396.70 rows=140 width=1259) (actual time=0.026..0.031 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=140 width=1251) (actual time=0.023..0.026 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..37.82 rows=12 width=1259) (actual time=0.026..0.028 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..37.82 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=12 width=1251) (actual time=0.023..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.148..0.155 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.147..0.154 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.098..0.137 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.096..0.132 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.070..0.075 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.050..0.064 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.040..0.041 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.016..0.017 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.293 ms
Execution Time: 3.472 ms
```

### composed: agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.7wYy83cAvO5RGw8ZnfSfU7aXPjj1m5yhQpbtqmg741o` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.045..0.047 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.029..0.031 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.009..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.328 ms
Execution Time: 0.086 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.024..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.110 ms
Execution Time: 0.062 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.062..0.063 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.061..0.062 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.046..0.053 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.038..0.044 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.013 rows=2 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.253 ms
Execution Time: 0.098 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=2768.53..2768.54 rows=1 width=8) (actual time=3.273..3.282 rows=1 loops=1)
  Buffers: shared hit=1023
  ->  Limit  (cost=0.55..2756.02 rows=1001 width=4) (actual time=0.054..3.151 rows=1001 loops=1)
        Buffers: shared hit=1023
        ->  Result  (cost=0.55..3177.18 rows=1154 width=4) (actual time=0.053..2.893 rows=1001 loops=1)
              Buffers: shared hit=1023
              ->  Append  (cost=0.55..3165.64 rows=1154 width=0) (actual time=0.052..2.698 rows=1001 loops=1)
                    Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..2679.35 rows=1001 width=0) (actual time=0.052..2.542 rows=1001 loops=1)
                          Buffers: shared hit=1023
                          ->  Limit  (cost=0.55..2669.34 rows=1001 width=4) (actual time=0.051..2.400 rows=1001 loops=1)
                                Buffers: shared hit=1023
                                ->  Result  (cost=0.55..15274.75 rows=5729 width=4) (actual time=0.050..2.285 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=1023
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..15274.75 rows=5729 width=0) (actual time=0.040..2.112 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 1
                                            Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..398.10 rows=140 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.70 rows=140 width=4) (never executed)
                                ->  Result  (cost=0.42..396.70 rows=140 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.70 rows=140 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.94 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.82 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..37.82 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.82 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.044 ms
Execution Time: 3.453 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) AND ((r.updated_at, r.id) < ($9::timestamptz, $10::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) AND ((r.updated_at, r.id) < ($9::timestamptz, $10::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) AND ((r.updated_at, r.id) < ($9::timestamptz, $10::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) AND ((r.updated_at, r.id) < ($9::timestamptz, $10::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $11
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.39..1419.42 rows=1 width=1301) (actual time=0.312..2.502 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1411.08 rows=1 width=1266) (actual time=0.285..2.282 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.81..1402.77 rows=1 width=1259) (actual time=0.227..2.083 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.81..1401.62 rows=51 width=1259) (actual time=0.225..2.027 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.81..3103.04 rows=115 width=1259) (actual time=0.223..2.016 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..2873.85 rows=51 width=1259) (actual time=0.057..1.817 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..309935.43 rows=5501 width=1259) (actual time=0.057..1.810 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..309935.43 rows=5501 width=1251) (actual time=0.053..1.786 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..145.44 rows=51 width=1259) (actual time=0.021..0.026 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..381.44 rows=134 width=1259) (actual time=0.021..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..381.44 rows=134 width=1251) (actual time=0.019..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..37.49 rows=12 width=1259) (actual time=0.022..0.029 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..37.49 rows=12 width=1259) (actual time=0.022..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.49 rows=12 width=1251) (actual time=0.020..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.50 rows=1 width=1259) (actual time=0.120..0.126 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.50 rows=1 width=1259) (actual time=0.120..0.126 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.080..0.115 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.078..0.112 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.058..0.062 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.042..0.052 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.759 ms
Execution Time: 2.709 ms
```

### composed: agent list, filter[state]=new

Request: `GET /api/v1/records/ticket?filter%5Bstate%5D=new` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.060..0.063 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.040..0.042 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.448 ms
Execution Time: 0.112 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.040..0.042 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.030..0.032 rows=1 loops=1)
        Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.145 ms
Execution Time: 0.076 ms
```

Statement 4:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.062..0.064 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.061..0.062 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.047..0.054 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.039..0.045 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.028 rows=1 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.010..0.013 rows=2 loops=1)
                          Index Cond: (tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.262 ms
Execution Time: 0.098 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=1272.46..1272.47 rows=1 width=8) (actual time=3.280..3.290 rows=1 loops=1)
  Buffers: shared hit=1040
  ->  Limit  (cost=0.55..1268.83 rows=290 width=4) (actual time=0.063..3.155 rows=1001 loops=1)
        Buffers: shared hit=1040
        ->  Result  (cost=0.55..1268.83 rows=290 width=4) (actual time=0.062..3.002 rows=1001 loops=1)
              Buffers: shared hit=1040
              ->  Append  (cost=0.55..1265.93 rows=290 width=0) (actual time=0.061..2.800 rows=1001 loops=1)
                    Buffers: shared hit=1040
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..785.20 rows=281 width=0) (actual time=0.060..2.634 rows=1001 loops=1)
                          Buffers: shared hit=1040
                          ->  Limit  (cost=0.55..782.39 rows=281 width=4) (actual time=0.059..2.485 rows=1001 loops=1)
                                Buffers: shared hit=1040
                                ->  Result  (cost=0.55..782.39 rows=281 width=4) (actual time=0.058..2.364 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=1040
                                      ->  Index Scan using records_list_group_state on records r  (cost=0.55..782.39 rows=281 width=0) (actual time=0.049..2.184 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 6
                                            Buffers: shared hit=1040
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..396.95 rows=7 width=0) (never executed)
                          ->  Limit  (cost=0.42..396.88 rows=7 width=4) (never executed)
                                ->  Result  (cost=0.42..396.88 rows=7 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.88 rows=7 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (state = ANY ('{new}'::text[]))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..37.84 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..37.83 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..37.83 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.83 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.231 ms
Execution Time: 3.414 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["0bda265b-a07a-45c4-915c-f302d32ad2c7","e8cf73ce-6fa2-46f0-af98-995c9ec22409",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=837.72..1224.60 rows=1 width=1301) (actual time=4.799..5.357 rows=51 loops=1)
  Buffers: shared hit=1585
  ->  Nested Loop  (cost=837.43..1216.27 rows=1 width=1266) (actual time=4.767..5.096 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1432
        ->  Subquery Scan on x  (cost=837.14..1207.95 rows=1 width=1259) (actual time=4.728..4.902 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1279
              ->  Limit  (cost=837.14..1206.81 rows=51 width=1259) (actual time=4.724..4.845 rows=51 loops=1)
                    Buffers: shared hit=1279
                    ->  Merge Append  (cost=837.14..1272.04 rows=60 width=1259) (actual time=4.722..4.835 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1279
                          ->  Limit  (cost=791.77..791.89 rows=51 width=1259) (actual time=4.468..4.491 rows=50 loops=1)
                                Buffers: shared hit=1204
                                ->  Sort  (cost=791.77..792.47 rows=281 width=1259) (actual time=4.467..4.482 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=1204
                                      ->  Result  (cost=0.55..782.39 rows=281 width=1259) (actual time=0.064..3.442 rows=1162 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=1204
                                            ->  Index Scan using records_list_group_state on records r  (cost=0.55..782.39 rows=281 width=1251) (actual time=0.059..3.075 rows=1162 loops=1)
                                                  Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                  Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Rows Removed by Filter: 6
                                                  Buffers: shared hit=1204
                          ->  Limit  (cost=0.42..396.88 rows=7 width=1259) (actual time=0.047..0.105 rows=2 loops=1)
                                Buffers: shared hit=30
                                ->  Result  (cost=0.42..396.88 rows=7 width=1259) (actual time=0.044..0.101 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=30
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..396.88 rows=7 width=1251) (actual time=0.031..0.087 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (state = ANY ('{new}'::text[]))
                                            Rows Removed by Filter: 24
                                            Buffers: shared hit=30
                          ->  Limit  (cost=0.42..37.83 rows=1 width=1259) (actual time=0.054..0.055 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..37.83 rows=1 width=1259) (actual time=0.053..0.054 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..37.83 rows=1 width=1251) (actual time=0.049..0.049 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.146..0.151 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.146..0.149 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.121..0.137 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.117..0.133 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.058..0.062 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.042..0.050 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.021..0.021 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '0bda265b-a07a-45c4-915c-f302d32ad2c7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'e8cf73ce-6fa2-46f0-af98-995c9ec22409'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.698 ms
Execution Time: 5.563 ms
```

### composed: single GET, agent

Request: `GET /api/v1/records/ticket/3f94a2ac-38ee-af8f-f434-143a689a293d` as `agent-1`


## Host

```
### 2026-11-19T23:05:41Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 26260877312 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: buildx_buildkit_nava-ci0 multica-backend-1 multica-frontend-1 multica-postgres-1 
```
