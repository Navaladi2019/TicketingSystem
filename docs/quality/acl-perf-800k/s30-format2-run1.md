# P3-36 raw report (TICK-621)

Generated 2026-11-19T23:05:09Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
heap 1042 MB, indexes 755 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 2.71 / p95 5.99 / p99 14.12 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_logical_emit_message(true, ...); COMMIT` probe, a commit that flushes WAL with no INSERT, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). An attempt is also void if a container other than the run's database, the buildkit builder and the three Multica platform containers runs during it. A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| agent list by group, default sort, page 1 | - | large | 2378 | 59.92 | 108.70 | 128.33 | - | - | - | - | - | 150 | meets |
| agent list by group, default sort, page 20 | - | large | 2020 | 68.98 | 138.72 | 231.40 | - | - | - | - | - | 150 | meets |
| requester list (heavy requesters), page 1 | - | large | 1075 | 140.56 | 291.85 | 397.81 | - | - | - | - | - | 150 | **MISS** |
| requester list (heavy requesters), page 20 | - | large | 929 | 155.83 | 305.46 | 386.38 | - | - | - | - | - | 150 | **MISS** |
| admin list, default sort, page 1 | - | large | 1280 | 116.59 | 203.78 | 268.73 | - | - | - | - | - | 150 | **MISS** |
| admin list, default sort, page 20 | - | large | 1905 | 71.56 | 176.33 | 226.50 | - | - | - | - | - | 150 | **MISS** |
| agent list, filter[state]=new | - | large | 2188 | 68.49 | 101.15 | 124.44 | - | - | - | - | - | 150 | meets |
| agent list, filter[priority]=1 (data) | - | large | 782 | 199.72 | 251.16 | 292.60 | - | - | - | - | - | 150 | **MISS** |
| agent list, filter[site]=s7 (rare data, < 1% of visible rows) | - | large | 611 | 255.57 | 349.77 | 402.88 | - | - | - | - | - | 150 | **MISS** |
| agent list, filter[site]=s7, short data-filter timeout (bound check) | - | large | 6153 | 23.45 | 40.88 | 54.41 | - | - | - | - | - | 0 | n/a (bound check) |
| agent list, sort=-updated_at | - | large | 2640 | 54.38 | 98.92 | 127.74 | - | - | - | - | - | 150 | meets |
| agent list, sort=assignee | - | large | 2582 | 55.95 | 99.39 | 126.74 | - | - | - | - | - | 150 | meets |
| admin list, filter[state][in]=new,in_progress, sort=-updated_at | - | large | 3191 | 46.41 | 82.95 | 100.04 | - | - | - | - | - | 150 | meets |
| small tenant: agent list, page 1 | - | small | 2492 | 59.11 | 96.14 | 115.04 | - | - | - | - | - | 150 | meets |
| small tenant: admin list, page 1 | - | small | 2902 | 51.62 | 94.12 | 118.80 | - | - | - | - | - | 150 | meets |
| single GET, agent | - | large | 8002 | 18.10 | 33.90 | 43.06 | - | - | - | - | - | 50 | meets |
| single GET, admin | - | large | 8175 | 17.83 | 34.14 | 43.98 | - | - | - | - | - | 50 | meets |
| single GET, small tenant agent | - | small | 7566 | 17.69 | 42.51 | 74.74 | - | - | - | - | - | 50 | meets |
| composed: agent list by group, default sort, page 1 | - | large | 3123 | 46.23 | 81.39 | 97.39 | - | - | - | - | - | 150 | meets |
| composed: agent list by group, default sort, page 20 | - | large | 3076 | 46.80 | 83.91 | 105.31 | - | - | - | - | - | 150 | meets |
| composed: agent list, filter[state]=new | - | large | 2587 | 55.20 | 99.21 | 123.44 | - | - | - | - | - | 150 | meets |
| composed: single GET, agent | - | large | 9759 | 13.86 | 30.83 | 39.91 | - | - | - | - | - | 50 | meets |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (2.71 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

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
| agent list, filter[priority]=1 (data) | 782 | 0 | 199.72 | 251.16 | 343.39 |
| agent list, filter[site]=s7 (rare data, < 1% of visible rows) | 611 | 0 | 255.57 | 349.77 | 458.35 |
| agent list, filter[site]=s7, short data-filter timeout (bound check) | 6153 | 6153 | 23.45 | 40.88 | 85.49 |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### agent list by group, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 2378 | 3.46 | 10.77 | 23.83 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2378 | 4.04 | 11.36 | 21.04 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2378 | 3.52 | 11.44 | 22.79 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2378 | 3.63 | 10.42 | 19.70 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2378 | 5.72 | 13.47 | 24.07 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2378 | 16.65 | 44.07 | 63.14 |
| 7 | `commit` | 2378 | 3.74 | 12.04 | 25.80 |

### agent list by group, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 2020 | 4.05 | 15.01 | 30.93 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2020 | 4.45 | 14.16 | 30.82 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2020 | 3.92 | 14.76 | 35.28 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2020 | 3.92 | 13.65 | 28.64 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2020 | 6.04 | 16.11 | 32.52 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2020 | 18.74 | 54.71 | 85.76 |
| 7 | `commit` | 2020 | 4.38 | 17.01 | 34.48 |

### requester list (heavy requesters), page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 1075 | 7.00 | 37.44 | 84.07 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1075 | 7.34 | 35.14 | 72.02 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1075 | 6.97 | 38.84 | 82.51 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1075 | 6.91 | 38.18 | 66.84 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1075 | 8.70 | 41.54 | 83.70 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1075 | 30.85 | 116.67 | 166.46 |
| 7 | `commit` | 1075 | 7.62 | 43.12 | 79.93 |

### requester list (heavy requesters), page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 929 | 9.46 | 36.57 | 66.90 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 929 | 9.14 | 35.15 | 62.82 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 929 | 9.27 | 42.45 | 67.75 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 929 | 9.32 | 36.47 | 62.33 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 929 | 11.59 | 43.81 | 68.14 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 929 | 40.74 | 124.63 | 162.11 |
| 7 | `commit` | 929 | 10.40 | 42.84 | 82.74 |

### admin list, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 1280 | 7.18 | 29.18 | 49.30 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1280 | 7.62 | 28.00 | 43.65 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1280 | 7.13 | 29.70 | 41.15 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1280 | 7.80 | 27.11 | 45.59 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 1280 | 8.05 | 30.99 | 53.34 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 1280 | 30.23 | 79.81 | 113.25 |
| 7 | `commit` | 1280 | 8.04 | 32.41 | 50.84 |

### admin list, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 1905 | 4.65 | 20.06 | 35.96 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1905 | 5.38 | 20.58 | 41.39 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1905 | 4.78 | 19.55 | 43.24 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1905 | 4.83 | 18.20 | 35.23 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 1905 | 5.17 | 18.51 | 37.80 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 1905 | 17.19 | 58.23 | 91.28 |
| 7 | `commit` | 1905 | 4.72 | 19.80 | 42.95 |

### agent list, filter[state]=new

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 2188 | 3.43 | 10.22 | 19.79 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2188 | 4.48 | 9.75 | 18.51 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2188 | 3.49 | 9.22 | 19.78 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2188 | 3.92 | 8.86 | 16.43 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2188 | 7.32 | 13.65 | 23.29 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2188 | 27.73 | 47.20 | 61.14 |
| 7 | `commit` | 2188 | 4.02 | 11.42 | 22.06 |

### agent list, filter[priority]=1 (data)

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 782 | 2.94 | 6.26 | 14.51 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 782 | 3.94 | 6.94 | 13.27 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 782 | 3.21 | 6.00 | 12.57 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 782 | 3.49 | 6.10 | 9.69 |
| 5 | `SELECT current_setting('statement_timeout')` | 782 | 2.78 | 5.18 | 8.10 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 782 | 2.42 | 4.79 | 7.40 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 782 | 71.74 | 88.24 | 110.15 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 782 | 93.39 | 116.52 | 131.31 |
| 9 | `SET LOCAL statement_timeout = '0'` | 782 | 3.48 | 8.43 | 18.97 |
| 10 | `commit` | 782 | 2.84 | 7.20 | 17.52 |

### agent list, filter[site]=s7 (rare data, < 1% of visible rows)

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 611 | 3.92 | 11.40 | 21.40 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 611 | 4.93 | 11.69 | 22.17 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 611 | 4.01 | 11.47 | 23.95 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 611 | 4.20 | 9.28 | 19.43 |
| 5 | `SELECT current_setting('statement_timeout')` | 611 | 3.43 | 8.50 | 13.37 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 611 | 3.19 | 8.09 | 11.62 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 611 | 92.00 | 137.48 | 184.40 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 611 | 106.47 | 153.78 | 190.83 |
| 9 | `SET LOCAL statement_timeout = '0'` | 611 | 4.41 | 16.71 | 40.91 |
| 10 | `commit` | 611 | 3.64 | 12.90 | 33.40 |

### agent list, filter[site]=s7, short data-filter timeout (bound check)

Round trips: 8, statements: 8, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 6153 | 2.00 | 4.20 | 7.08 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 6153 | 2.86 | 5.23 | 8.13 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 6153 | 2.31 | 4.90 | 9.03 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 6153 | 2.57 | 4.78 | 7.91 |
| 5 | `SELECT current_setting('statement_timeout')` | 6153 | 1.99 | 4.42 | 7.46 |
| 6 | `SET LOCAL statement_timeout = '2'` | 6153 | 1.80 | 4.12 | 7.35 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 6153 | 4.27 | 6.50 | 9.22 |
| 8 | `rollback` | 6153 | 1.79 | 3.90 | 7.12 |

### agent list, sort=-updated_at

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 2640 | 3.16 | 9.74 | 21.15 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2640 | 3.69 | 9.63 | 17.73 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2640 | 3.22 | 9.27 | 19.05 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2640 | 3.27 | 8.76 | 18.48 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2640 | 5.11 | 11.70 | 23.41 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2640 | 14.60 | 37.78 | 56.10 |
| 7 | `commit` | 2640 | 3.12 | 10.78 | 20.24 |

### agent list, sort=assignee

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 2582 | 3.26 | 9.82 | 19.18 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2582 | 3.83 | 9.95 | 18.21 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2582 | 3.34 | 10.30 | 16.96 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2582 | 3.46 | 9.79 | 18.20 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2582 | 5.23 | 13.01 | 25.24 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2582 | 15.96 | 39.88 | 55.10 |
| 7 | `commit` | 2582 | 3.20 | 10.82 | 19.69 |

### admin list, filter[state][in]=new,in_progress, sort=-updated_at

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 3191 | 2.86 | 8.68 | 16.57 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3191 | 3.42 | 9.77 | 19.56 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3191 | 2.96 | 9.47 | 18.20 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3191 | 3.13 | 8.45 | 17.19 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 3191 | 3.65 | 9.78 | 17.31 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 3191 | 10.72 | 30.86 | 42.67 |
| 7 | `commit` | 3191 | 2.81 | 9.65 | 17.99 |

### small tenant: agent list, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39', true)` | 7 | 4.02 | 6.20 | 6.20 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2492 | 3.83 | 9.00 | 18.82 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2492 | 3.21 | 8.64 | 17.59 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2492 | 3.42 | 8.40 | 16.99 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2492 | 10.99 | 17.32 | 26.35 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2492 | 17.23 | 38.28 | 53.18 |
| 7 | `commit` | 2492 | 2.90 | 9.03 | 18.99 |

### small tenant: admin list, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39', true)` | 8 | 2.79 | 9.11 | 9.11 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2902 | 3.97 | 10.59 | 19.20 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2902 | 3.42 | 10.84 | 22.49 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2902 | 3.64 | 10.04 | 19.62 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 2902 | 3.99 | 10.79 | 19.06 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 2902 | 11.99 | 35.03 | 50.60 |
| 7 | `commit` | 2902 | 3.17 | 10.37 | 21.01 |

### single GET, agent

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 8002 | 2.95 | 6.97 | 10.46 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 8002 | 9.47 | 20.20 | 25.98 |
| 3 | `commit` | 8002 | 2.96 | 7.15 | 10.61 |

### single GET, admin

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 8175 | 2.86 | 7.02 | 10.42 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 8175 | 9.30 | 20.03 | 25.92 |
| 3 | `commit` | 8175 | 2.91 | 7.09 | 11.65 |

### single GET, small tenant agent

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39', true)` | 20 | 3.35 | 6.36 | 20.81 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 7566 | 9.17 | 22.83 | 35.76 |
| 3 | `commit` | 7566 | 2.92 | 9.41 | 19.07 |

### composed: agent list by group, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 3123 | 2.54 | 7.65 | 14.85 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3123 | 3.24 | 8.44 | 15.56 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3123 | 2.67 | 7.77 | 15.10 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3123 | 2.86 | 7.38 | 13.79 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3123 | 6.55 | 12.13 | 19.76 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3123 | 11.87 | 31.78 | 44.97 |
| 7 | `commit` | 3123 | 2.45 | 8.37 | 15.24 |

### composed: agent list by group, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 3076 | 2.48 | 8.35 | 18.66 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3076 | 3.15 | 8.43 | 15.91 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3076 | 2.63 | 7.77 | 14.96 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3076 | 2.83 | 7.47 | 14.83 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3076 | 6.44 | 12.27 | 21.24 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3076 | 12.08 | 32.69 | 48.76 |
| 7 | `commit` | 3076 | 2.45 | 8.50 | 15.98 |

### composed: agent list, filter[state]=new

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 2587 | 2.86 | 9.81 | 19.87 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2587 | 3.47 | 9.79 | 17.43 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2587 | 2.92 | 9.13 | 19.26 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2587 | 3.17 | 8.90 | 15.56 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2587 | 7.15 | 14.32 | 23.61 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2587 | 17.25 | 41.28 | 59.31 |
| 7 | `commit` | 2587 | 2.86 | 10.17 | 23.59 |

### composed: single GET, agent

Round trips: 3, statements: 6, statements per batch: [4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e5a182fe-5737-4278-83a6-e7106315eceb', true)` | 9759 | 2.25 | 5.96 | 9.16 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.versi…` | 9759 | 6.85 | 18.67 | 24.32 |
| 3 | `commit` | 9759 | 2.28 | 5.93 | 9.68 |

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.042..0.045 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.028..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.303 ms
Execution Time: 0.081 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.047..0.049 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.040..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.109 ms
Execution Time: 0.072 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.055..0.056 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.054..0.055 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.045..0.048 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.038..0.040 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.028 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.231 ms
Execution Time: 0.087 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=532.48..532.49 rows=1 width=8) (actual time=1.284..1.288 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..519.97 rows=1001 width=4) (actual time=0.058..1.177 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..577.57 rows=1112 width=4) (actual time=0.057..1.022 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..566.45 rows=1112 width=0) (actual time=0.057..0.862 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..98.41 rows=1001 width=0) (actual time=0.056..0.754 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..88.40 rows=1001 width=4) (actual time=0.055..0.627 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..517.41 rows=5889 width=4) (actual time=0.055..0.526 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.41 rows=5889 width=0) (actual time=0.046..0.375 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.95 rows=98 width=0) (never executed)
                          ->  Limit  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                ->  Result  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..365.97 rows=98 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.04 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.92 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.052 ms
Execution Time: 1.450 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1780.89 rows=1 width=1301) (actual time=0.421..2.314 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1772.55 rows=1 width=1266) (actual time=0.395..2.088 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1764.24 rows=1 width=1259) (actual time=0.374..1.914 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1763.09 rows=51 width=1259) (actual time=0.370..1.853 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3918.13 rows=115 width=1259) (actual time=0.368..1.842 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3630.30 rows=51 width=1259) (actual time=0.174..1.624 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..419144.43 rows=5889 width=1259) (actual time=0.174..1.615 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..419144.43 rows=5889 width=1251) (actual time=0.166..1.588 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..190.66 rows=51 width=1259) (actual time=0.032..0.037 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..365.97 rows=98 width=1259) (actual time=0.032..0.036 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..365.97 rows=98 width=1251) (actual time=0.027..0.030 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..50.92 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..50.92 rows=12 width=1259) (actual time=0.025..0.026 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.92 rows=12 width=1251) (actual time=0.023..0.023 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.132..0.136 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.131..0.135 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.087..0.116 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.085..0.112 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.057..0.061 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.043..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.035..0.036 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.018..0.019 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.627 ms
Execution Time: 2.610 ms
```

### agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.BJkdT4WmdSnIZySjSLIq--qvP2ZFz-fjLPw-PdVDlCw` as `agent-1`

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.059..0.062 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.038..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.453 ms
Execution Time: 0.117 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.047..0.050 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.035..0.037 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.168 ms
Execution Time: 0.088 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.074..0.077 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.073..0.074 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.060..0.064 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.049..0.053 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.034..0.036 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.011..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.368 ms
Execution Time: 0.127 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=532.48..532.49 rows=1 width=8) (actual time=2.148..2.159 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..519.97 rows=1001 width=4) (actual time=0.089..1.957 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..577.57 rows=1112 width=4) (actual time=0.087..1.753 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..566.45 rows=1112 width=0) (actual time=0.087..1.463 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..98.41 rows=1001 width=0) (actual time=0.085..1.277 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..88.40 rows=1001 width=4) (actual time=0.084..1.060 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..517.41 rows=5889 width=4) (actual time=0.083..0.834 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.41 rows=5889 width=0) (actual time=0.069..0.577 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.95 rows=98 width=0) (never executed)
                          ->  Limit  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                ->  Result  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..365.97 rows=98 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.04 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.92 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.641 ms
Execution Time: 2.474 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.38..1808.85 rows=1 width=1301) (actual time=0.412..3.104 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1800.51 rows=1 width=1266) (actual time=0.379..2.726 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.80..1792.20 rows=1 width=1259) (actual time=0.350..2.485 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.80..1791.05 rows=51 width=1259) (actual time=0.347..2.403 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.80..3981.16 rows=115 width=1259) (actual time=0.344..2.386 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..3696.21 rows=51 width=1259) (actual time=0.082..2.076 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..409508.44 rows=5651 width=1259) (actual time=0.081..2.064 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..409508.44 rows=5651 width=1251) (actual time=0.072..2.028 rows=49 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..191.26 rows=51 width=1259) (actual time=0.044..0.054 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..352.15 rows=94 width=1259) (actual time=0.044..0.053 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..352.15 rows=94 width=1251) (actual time=0.040..0.047 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..47.43 rows=12 width=1259) (actual time=0.034..0.043 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..47.43 rows=12 width=1259) (actual time=0.034..0.042 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..47.43 rows=12 width=1251) (actual time=0.031..0.037 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.179..0.186 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.178..0.185 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.122..0.164 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.119..0.159 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.087..0.093 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.066..0.079 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.053..0.055 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.028..0.028 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.019..0.019 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.859 ms
Execution Time: 3.570 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.061..0.065 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.041..0.044 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.495 ms
Execution Time: 0.122 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.045..0.048 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.167 ms
Execution Time: 0.088 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.097..0.100 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.094..0.096 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.076..0.082 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.063..0.068 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.043..0.044 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.014..0.016 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.377 ms
Execution Time: 0.163 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=196.24..196.25 rows=1 width=8) (actual time=1.681..1.685 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..183.73 rows=1001 width=4) (actual time=0.064..1.540 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..186.29 rows=1015 width=4) (actual time=0.063..1.320 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..176.14 rows=1015 width=0) (actual time=0.062..1.112 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..98.97 rows=1001 width=0) (actual time=0.061..0.970 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..88.96 rows=1001 width=4) (actual time=0.059..0.772 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..128.68 rows=1450 width=4) (actual time=0.059..0.639 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..128.68 rows=1450 width=0) (actual time=0.047..0.440 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..50.98 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.85 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..50.85 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..50.85 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.326 ms
Execution Time: 1.822 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],51]`

```
Nested Loop Left Join  (cost=22.56..223.18 rows=1 width=1301) (actual time=0.168..1.035 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..214.85 rows=1 width=1266) (actual time=0.140..0.699 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.98..206.53 rows=1 width=1259) (actual time=0.122..0.485 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.98..205.39 rows=51 width=1259) (actual time=0.119..0.411 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.98..255.73 rows=65 width=1259) (actual time=0.117..0.396 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..182.90 rows=51 width=1259) (actual time=0.055..0.315 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..5188.41 rows=1450 width=1259) (actual time=0.054..0.304 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..5188.41 rows=1450 width=1251) (actual time=0.047..0.272 rows=51 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..50.85 rows=13 width=1259) (actual time=0.031..0.032 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..50.85 rows=13 width=1259) (actual time=0.031..0.031 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..50.85 rows=13 width=1251) (actual time=0.027..0.027 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.029..0.032 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.028..0.031 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.10 rows=1 width=1259) (actual time=0.023..0.025 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.10 rows=1 width=1251) (actual time=0.020..0.022 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.020..0.021 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.017..0.018 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.930 ms
Execution Time: 1.270 ms
```

### requester list (heavy requesters), page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTI5IDAzOjIwOjI1LjYwNTY3NSswMCIsImkiOiJkOWZlM2ZiMi02ZWU5LTdhMTktNjE1Ny0wODUxOWU2M2U3OTcifQ.o6zt7ckESnG5PYAdJ29MXTfcgcPsMNpk4icB50S9_F4` as `req-1`

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
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.039..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.516 ms
Execution Time: 0.121 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.048..0.051 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.035..0.037 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.168 ms
Execution Time: 0.091 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.073..0.075 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.071..0.073 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.059..0.064 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.048..0.052 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.032..0.033 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.012..0.014 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.349 ms
Execution Time: 0.125 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=196.24..196.25 rows=1 width=8) (actual time=1.989..1.996 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..183.73 rows=1001 width=4) (actual time=0.064..1.816 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..186.29 rows=1015 width=4) (actual time=0.062..1.618 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..176.14 rows=1015 width=0) (actual time=0.061..1.358 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..98.97 rows=1001 width=0) (actual time=0.060..1.180 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..88.96 rows=1001 width=4) (actual time=0.059..0.968 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..128.68 rows=1450 width=4) (actual time=0.058..0.801 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..128.68 rows=1450 width=0) (actual time=0.047..0.552 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..50.98 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.85 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..50.85 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..50.85 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.210 ms
Execution Time: 2.140 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],"2026-09-29 03:20:25.605675+00","d9fe3fb2-6ee9-7a19-6157-08519e63e797",51]`

```
Nested Loop Left Join  (cost=22.57..223.42 rows=1 width=1301) (actual time=0.206..1.173 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..215.09 rows=1 width=1266) (actual time=0.178..0.750 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.99..206.77 rows=1 width=1259) (actual time=0.157..0.521 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.99..205.63 rows=51 width=1259) (actual time=0.154..0.441 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.99..248.84 rows=63 width=1259) (actual time=0.152..0.425 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..183.15 rows=51 width=1259) (actual time=0.055..0.309 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..4289.05 rows=1197 width=1259) (actual time=0.055..0.298 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..4289.05 rows=1197 width=1251) (actual time=0.047..0.263 rows=51 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..43.73 rows=11 width=1259) (actual time=0.033..0.034 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..43.73 rows=11 width=1259) (actual time=0.032..0.033 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..43.73 rows=11 width=1251) (actual time=0.029..0.029 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.062..0.065 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.061..0.064 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.11 rows=1 width=1259) (actual time=0.025..0.028 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.11 rows=1 width=1251) (actual time=0.021..0.024 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.021..0.022 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.018..0.019 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.187 ms
Execution Time: 1.429 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.057..0.061 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.039..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.010..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.446 ms
Execution Time: 0.121 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.042..0.045 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.031..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.163 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.046..0.047 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.045..0.045 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.036..0.038 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.031..0.033 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.021..0.022 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.208 ms
Execution Time: 0.078 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21",["2fd59270-9dd1-4d31-8a87-448af03e03ef"]]`

```
Aggregate  (cost=78.80..78.81 rows=1 width=8) (actual time=0.959..0.960 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.29 rows=1001 width=4) (actual time=0.087..0.754 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..13252.76 rows=201417 width=4) (actual time=0.084..0.584 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..13252.76 rows=201417 width=0) (actual time=0.071..0.336 rows=1001 loops=1)
                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.357 ms
Execution Time: 1.010 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21",["2fd59270-9dd1-4d31-8a87-448af03e03ef"],51]`

```
Limit  (cost=1.00..124.58 rows=51 width=1293) (actual time=0.084..0.464 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..488087.51 rows=201417 width=1293) (actual time=0.082..0.453 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..486576.88 rows=201417 width=1266) (actual time=0.066..0.406 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..420663.35 rows=201417 width=1258) (actual time=0.053..0.213 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..418137.34 rows=201417 width=1251) (actual time=0.036..0.165 rows=51 loops=1)
                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.011..0.012 rows=1 loops=1)
                                Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                Filter: (id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.511 ms
Execution Time: 0.529 ms
```

### admin list, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjU5OjU5LjU2MzExOCswMCIsImkiOiJkNmRhMTEwZi1jOGM3LTEzZmItZjRmNy1jNWUzYjFkMmMzMjQifQ.XhhMNcSDPT916MI-d4Y1yW44ww4Dl0IS9cud80-DfEc` as `admin-large`

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
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.041..0.044 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.412 ms
Execution Time: 0.117 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.045 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.161 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.110..0.112 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.108..0.110 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.094..0.099 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.050..0.054 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.033..0.035 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.011..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.354 ms
Execution Time: 0.161 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21",["2fd59270-9dd1-4d31-8a87-448af03e03ef"]]`

```
Aggregate  (cost=78.80..78.81 rows=1 width=8) (actual time=0.861..0.863 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.29 rows=1001 width=4) (actual time=0.057..0.726 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..13252.76 rows=201417 width=4) (actual time=0.055..0.556 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..13252.76 rows=201417 width=0) (actual time=0.044..0.310 rows=1001 loops=1)
                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.286 ms
Execution Time: 0.908 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21",["2fd59270-9dd1-4d31-8a87-448af03e03ef"],"2026-09-30 23:59:59.563118+00","d6da110f-c8c7-13fb-f4f7-c5e3b1d2c324",51]`

```
Limit  (cost=1.00..124.72 rows=51 width=1293) (actual time=0.179..0.803 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..488550.06 rows=201393 width=1293) (actual time=0.177..0.787 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..487039.61 rows=201393 width=1266) (actual time=0.155..0.713 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..421133.66 rows=201393 width=1258) (actual time=0.129..0.367 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..418607.95 rows=201393 width=1251) (actual time=0.047..0.230 rows=51 loops=1)
                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:59:59.563118+00'::timestamp with time zone, 'd6da110f-c8c7-13fb-f4f7-c5e3b1d2c324'::uuid)))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.016..0.018 rows=1 loops=1)
                                Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                Filter: (id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.747 ms
Execution Time: 0.902 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.046..0.049 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.031..0.032 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.327 ms
Execution Time: 0.089 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.026..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.120 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.057..0.058 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.055..0.057 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.045..0.049 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.038..0.041 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.009..0.011 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.252 ms
Execution Time: 0.091 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=514.74..514.75 rows=1 width=8) (actual time=1.677..1.685 rows=1 loops=1)
  Buffers: shared hit=539
  ->  Limit  (cost=0.55..511.04 rows=296 width=4) (actual time=0.075..1.543 rows=1001 loops=1)
        Buffers: shared hit=539
        ->  Result  (cost=0.55..511.04 rows=296 width=4) (actual time=0.073..1.394 rows=1001 loops=1)
              Buffers: shared hit=539
              ->  Append  (cost=0.55..508.08 rows=296 width=0) (actual time=0.073..1.201 rows=1001 loops=1)
                    Buffers: shared hit=539
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..45.02 rows=289 width=0) (actual time=0.072..1.062 rows=1001 loops=1)
                          Buffers: shared hit=539
                          ->  Limit  (cost=0.55..42.13 rows=289 width=4) (actual time=0.071..0.903 rows=1001 loops=1)
                                Buffers: shared hit=539
                                ->  Result  (cost=0.55..42.13 rows=289 width=4) (actual time=0.070..0.773 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=539
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..42.13 rows=289 width=0) (actual time=0.061..0.584 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=539
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.15 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.42..366.10 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.42..366.10 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.10 rows=5 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..50.95 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.94 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..50.94 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.94 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.201 ms
Execution Time: 1.921 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=1111.46..1495.50 rows=1 width=1301) (actual time=4.933..5.429 rows=51 loops=1)
  Buffers: shared hit=1601
  ->  Nested Loop  (cost=1111.16..1487.17 rows=1 width=1266) (actual time=4.907..5.235 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1448
        ->  Subquery Scan on x  (cost=1110.88..1478.86 rows=1 width=1259) (actual time=4.890..5.088 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1295
              ->  Limit  (cost=1110.88..1477.71 rows=51 width=1259) (actual time=4.887..5.010 rows=51 loops=1)
                    Buffers: shared hit=1295
                    ->  Merge Append  (cost=1110.88..1528.06 rows=58 width=1259) (actual time=4.884..5.000 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1295
                          ->  Limit  (cost=1065.50..1065.63 rows=51 width=1259) (actual time=4.622..4.642 rows=50 loops=1)
                                Buffers: shared hit=1204
                                ->  Sort  (cost=1065.50..1066.22 rows=289 width=1259) (actual time=4.621..4.635 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=1204
                                      ->  Result  (cost=0.55..1055.86 rows=289 width=1259) (actual time=0.060..3.465 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=1204
                                            ->  Index Scan using records_list_group_state on records r  (cost=0.55..1055.86 rows=289 width=1251) (actual time=0.055..3.093 rows=1168 loops=1)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                  Buffers: shared hit=1204
                          ->  Limit  (cost=0.42..366.10 rows=5 width=1259) (actual time=0.079..0.150 rows=2 loops=1)
                                Buffers: shared hit=46
                                ->  Result  (cost=0.42..366.10 rows=5 width=1259) (actual time=0.076..0.147 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=46
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.10 rows=5 width=1251) (actual time=0.064..0.133 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=46
                          ->  Limit  (cost=0.42..50.94 rows=1 width=1259) (actual time=0.054..0.055 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..50.94 rows=1 width=1259) (actual time=0.053..0.054 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.94 rows=1 width=1251) (actual time=0.049..0.049 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.124..0.126 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.123..0.126 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.110..0.113 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.107..0.110 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.060..0.062 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.044..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.034..0.035 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.882 ms
Execution Time: 5.729 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.038..0.040 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.025..0.027 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.007..0.007 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.274 ms
Execution Time: 0.075 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.033 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.024..0.025 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.124 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.048..0.050 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.047..0.048 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.038..0.041 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.032..0.034 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.022..0.023 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.227 ms
Execution Time: 0.080 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.015 ms
Execution Time: 0.016 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}"]`

```
Aggregate  (cost=20509.58..20509.59 rows=1 width=8) (actual time=39.363..39.377 rows=1 loops=1)
  Buffers: shared hit=17769
  ->  Limit  (cost=458.55..20508.81 rows=62 width=4) (actual time=13.167..39.226 rows=1001 loops=1)
        Buffers: shared hit=17769
        ->  Result  (cost=458.55..20508.81 rows=62 width=4) (actual time=13.165..39.083 rows=1001 loops=1)
              Buffers: shared hit=17769
              ->  Append  (cost=458.55..20508.19 rows=62 width=0) (actual time=13.164..38.896 rows=1001 loops=1)
                    Buffers: shared hit=17769
                    ->  Subquery Scan on "*SELECT* 1"  (cost=458.55..20046.20 rows=59 width=0) (actual time=13.162..38.771 rows=1001 loops=1)
                          Buffers: shared hit=17769
                          ->  Limit  (cost=458.55..20045.61 rows=59 width=4) (actual time=13.161..38.599 rows=1001 loops=1)
                                Buffers: shared hit=17769
                                ->  Result  (cost=458.55..20045.61 rows=59 width=4) (actual time=13.160..38.492 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=17769
                                      ->  Bitmap Heap Scan on records r  (cost=458.55..20045.61 rows=59 width=0) (actual time=13.148..38.322 rows=1001 loops=1)
                                            Recheck Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"priority": "1"}'::jsonb)
                                            Rows Removed by Filter: 19221
                                            Heap Blocks: exact=17394
                                            Buffers: shared hit=17769
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..458.52 rows=5889 width=0) (actual time=8.147..8.147 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.23 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..366.22 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..366.22 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..50.96 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.95 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..50.95 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.764 ms
Execution Time: 39.748 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}",51]`

```
Nested Loop Left Join  (cost=20093.30..20504.60 rows=1 width=1301) (actual time=40.282..40.673 rows=51 loops=1)
  Buffers: shared hit=21490
  ->  Nested Loop  (cost=20093.00..20496.26 rows=1 width=1266) (actual time=40.265..40.525 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21337
        ->  Subquery Scan on x  (cost=20092.72..20487.95 rows=1 width=1259) (actual time=40.255..40.421 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21184
              ->  Limit  (cost=20092.72..20486.80 rows=51 width=1259) (actual time=40.252..40.386 rows=51 loops=1)
                    Buffers: shared hit=21184
                    ->  Merge Append  (cost=20092.72..20509.99 rows=54 width=1259) (actual time=40.251..40.380 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21184
                          ->  Limit  (cost=20047.34..20047.47 rows=51 width=1259) (actual time=40.040..40.069 rows=50 loops=1)
                                Buffers: shared hit=21119
                                ->  Sort  (cost=20047.34..20047.49 rows=59 width=1259) (actual time=40.039..40.063 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=21119
                                      ->  Result  (cost=458.55..20045.61 rows=59 width=1259) (actual time=9.471..39.115 rows=1182 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=21119
                                            ->  Bitmap Heap Scan on records r  (cost=458.55..20045.61 rows=59 width=1251) (actual time=9.466..38.809 rows=1182 loops=1)
                                                  Recheck Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"priority": "1"}'::jsonb)
                                                  Rows Removed by Filter: 22936
                                                  Heap Blocks: exact=20744
                                                  Buffers: shared hit=21119
                                                  ->  Bitmap Index Scan on records_list_group  (cost=0.00..458.52 rows=5889 width=0) (actual time=5.223..5.223 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=375
                          ->  Limit  (cost=0.42..366.22 rows=1 width=1259) (actual time=0.080..0.081 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.42..366.22 rows=1 width=1259) (actual time=0.078..0.079 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=23
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=1 width=1251) (actual time=0.064..0.064 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 19
                                            Buffers: shared hit=23
                          ->  Limit  (cost=0.42..50.95 rows=1 width=1259) (actual time=0.041..0.100 rows=2 loops=1)
                                Buffers: shared hit=17
                                ->  Result  (cost=0.42..50.95 rows=1 width=1259) (actual time=0.040..0.099 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=17
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=1 width=1251) (actual time=0.036..0.095 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 12
                                            Buffers: shared hit=17
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.086..0.089 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.085..0.088 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.078..0.081 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.076..0.078 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.047 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.037 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.474 ms
Execution Time: 40.911 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.045..0.048 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.031..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.296 ms
Execution Time: 0.085 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.044 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.031..0.033 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.226 ms
Execution Time: 0.073 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.057..0.059 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.056..0.057 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.047..0.051 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.040..0.043 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.011..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.262 ms
Execution Time: 0.094 ms
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
Execution Time: 0.021 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}"]`

```
Aggregate  (cost=20509.58..20509.59 rows=1 width=8) (actual time=42.730..42.739 rows=1 loops=1)
  Buffers: shared hit=21734
  ->  Limit  (cost=458.55..20508.81 rows=62 width=4) (actual time=10.890..42.693 rows=121 loops=1)
        Buffers: shared hit=21734
        ->  Result  (cost=458.55..20508.81 rows=62 width=4) (actual time=10.888..42.659 rows=121 loops=1)
              Buffers: shared hit=21734
              ->  Append  (cost=458.55..20508.19 rows=62 width=0) (actual time=10.887..42.623 rows=121 loops=1)
                    Buffers: shared hit=21734
                    ->  Subquery Scan on "*SELECT* 1"  (cost=458.55..20046.20 rows=59 width=0) (actual time=10.886..41.261 rows=120 loops=1)
                          Buffers: shared hit=21119
                          ->  Limit  (cost=458.55..20045.61 rows=59 width=4) (actual time=10.885..41.229 rows=120 loops=1)
                                Buffers: shared hit=21119
                                ->  Result  (cost=458.55..20045.61 rows=59 width=4) (actual time=10.883..41.207 rows=120 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=21119
                                      ->  Bitmap Heap Scan on records r  (cost=458.55..20045.61 rows=59 width=0) (actual time=10.875..41.164 rows=120 loops=1)
                                            Recheck Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"site": "s7"}'::jsonb)
                                            Rows Removed by Filter: 23998
                                            Heap Blocks: exact=20744
                                            Buffers: shared hit=21119
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..458.52 rows=5889 width=0) (actual time=6.093..6.093 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.23 rows=1 width=0) (actual time=0.463..1.075 rows=1 loops=1)
                          Buffers: shared hit=523
                          ->  Limit  (cost=0.42..366.22 rows=1 width=4) (actual time=0.462..1.072 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..366.22 rows=1 width=4) (actual time=0.459..1.069 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=1 width=0) (actual time=0.443..1.052 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..50.96 rows=1 width=0) (actual time=0.162..0.163 rows=0 loops=1)
                          Buffers: shared hit=67
                          ->  Limit  (cost=0.42..50.95 rows=1 width=4) (actual time=0.161..0.162 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..50.95 rows=1 width=4) (actual time=0.159..0.159 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=1 width=0) (actual time=0.150..0.150 rows=0 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (actual time=0.090..0.092 rows=0 loops=1)
                          Buffers: shared hit=25
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (actual time=0.089..0.091 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (actual time=0.088..0.090 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=25
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (actual time=0.085..0.086 rows=0 loops=1)
                                            Buffers: shared hit=25
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.050..0.052 rows=3 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=13
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.041 rows=3 loops=1)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Heap Blocks: exact=3
                                                        Buffers: shared hit=13
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                              Buffers: shared hit=10
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=8
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.009..0.010 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.937 ms
Execution Time: 42.848 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}",51]`

```
Nested Loop Left Join  (cost=20093.30..20504.60 rows=1 width=1301) (actual time=45.305..46.284 rows=51 loops=1)
  Buffers: shared hit=22040
  ->  Nested Loop  (cost=20093.00..20496.26 rows=1 width=1266) (actual time=45.283..46.111 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21887
        ->  Subquery Scan on x  (cost=20092.72..20487.95 rows=1 width=1259) (actual time=45.271..45.982 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21734
              ->  Limit  (cost=20092.72..20486.80 rows=51 width=1259) (actual time=45.268..45.935 rows=51 loops=1)
                    Buffers: shared hit=21734
                    ->  Merge Append  (cost=20092.72..20509.99 rows=54 width=1259) (actual time=45.266..45.927 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21734
                          ->  Limit  (cost=20047.34..20047.47 rows=51 width=1259) (actual time=44.619..44.640 rows=50 loops=1)
                                Buffers: shared hit=21119
                                ->  Sort  (cost=20047.34..20047.49 rows=59 width=1259) (actual time=44.617..44.632 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=21119
                                      ->  Result  (cost=458.55..20045.61 rows=59 width=1259) (actual time=11.006..44.400 rows=120 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=21119
                                            ->  Bitmap Heap Scan on records r  (cost=458.55..20045.61 rows=59 width=1251) (actual time=10.999..44.326 rows=120 loops=1)
                                                  Recheck Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"site": "s7"}'::jsonb)
                                                  Rows Removed by Filter: 23998
                                                  Heap Blocks: exact=20744
                                                  Buffers: shared hit=21119
                                                  ->  Bitmap Index Scan on records_list_group  (cost=0.00..458.52 rows=5889 width=0) (actual time=6.152..6.152 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=375
                          ->  Limit  (cost=0.42..366.22 rows=1 width=1259) (actual time=0.407..1.022 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..366.22 rows=1 width=1259) (actual time=0.404..1.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=1 width=1251) (actual time=0.389..1.003 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                          ->  Limit  (cost=0.42..50.95 rows=1 width=1259) (actual time=0.134..0.135 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..50.95 rows=1 width=1259) (actual time=0.133..0.134 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=1 width=1251) (actual time=0.130..0.130 rows=0 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.103..0.107 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.102..0.106 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.092..0.096 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.089..0.093 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.057 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.800 ms
Execution Time: 46.588 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.053..0.056 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.034..0.036 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.397 ms
Execution Time: 0.105 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.037..0.040 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.032 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.327 ms
Execution Time: 0.073 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.049..0.050 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.048..0.049 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.039..0.042 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.033..0.036 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.021..0.022 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.212 ms
Execution Time: 0.079 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.007..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.019 ms
Execution Time: 0.020 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}"]`

```
Aggregate  (cost=20509.58..20509.59 rows=1 width=8) (actual time=39.536..39.544 rows=1 loops=1)
  Buffers: shared hit=21734
  ->  Limit  (cost=458.55..20508.81 rows=62 width=4) (actual time=9.548..39.502 rows=121 loops=1)
        Buffers: shared hit=21734
        ->  Result  (cost=458.55..20508.81 rows=62 width=4) (actual time=9.547..39.477 rows=121 loops=1)
              Buffers: shared hit=21734
              ->  Append  (cost=458.55..20508.19 rows=62 width=0) (actual time=9.546..39.442 rows=121 loops=1)
                    Buffers: shared hit=21734
                    ->  Subquery Scan on "*SELECT* 1"  (cost=458.55..20046.20 rows=59 width=0) (actual time=9.545..38.259 rows=120 loops=1)
                          Buffers: shared hit=21119
                          ->  Limit  (cost=458.55..20045.61 rows=59 width=4) (actual time=9.544..38.228 rows=120 loops=1)
                                Buffers: shared hit=21119
                                ->  Result  (cost=458.55..20045.61 rows=59 width=4) (actual time=9.543..38.207 rows=120 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=21119
                                      ->  Bitmap Heap Scan on records r  (cost=458.55..20045.61 rows=59 width=0) (actual time=9.536..38.168 rows=120 loops=1)
                                            Recheck Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"site": "s7"}'::jsonb)
                                            Rows Removed by Filter: 23998
                                            Heap Blocks: exact=20744
                                            Buffers: shared hit=21119
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..458.52 rows=5889 width=0) (actual time=4.450..4.450 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.23 rows=1 width=0) (actual time=0.383..0.926 rows=1 loops=1)
                          Buffers: shared hit=523
                          ->  Limit  (cost=0.42..366.22 rows=1 width=4) (actual time=0.381..0.924 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..366.22 rows=1 width=4) (actual time=0.380..0.922 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=1 width=0) (actual time=0.368..0.910 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..50.96 rows=1 width=0) (actual time=0.126..0.126 rows=0 loops=1)
                          Buffers: shared hit=67
                          ->  Limit  (cost=0.42..50.95 rows=1 width=4) (actual time=0.125..0.125 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..50.95 rows=1 width=4) (actual time=0.124..0.124 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=1 width=0) (actual time=0.118..0.119 rows=0 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (actual time=0.080..0.082 rows=0 loops=1)
                          Buffers: shared hit=25
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (actual time=0.079..0.081 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (actual time=0.079..0.080 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=25
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (actual time=0.076..0.078 rows=0 loops=1)
                                            Buffers: shared hit=25
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.046..0.048 rows=3 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=13
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.038 rows=3 loops=1)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Heap Blocks: exact=3
                                                        Buffers: shared hit=13
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                              Buffers: shared hit=10
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=8
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.911 ms
Execution Time: 39.647 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.053..0.057 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.034..0.037 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.359 ms
Execution Time: 0.106 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.046..0.050 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.276 ms
Execution Time: 0.125 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.050..0.051 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.049..0.049 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.040..0.043 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.033..0.036 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.022..0.023 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.008..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.229 ms
Execution Time: 0.081 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=532.48..532.49 rows=1 width=8) (actual time=0.918..0.921 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..519.97 rows=1001 width=4) (actual time=0.037..0.841 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..577.57 rows=1112 width=4) (actual time=0.036..0.755 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..566.45 rows=1112 width=0) (actual time=0.036..0.637 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..98.41 rows=1001 width=0) (actual time=0.035..0.556 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..88.40 rows=1001 width=4) (actual time=0.035..0.465 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..517.41 rows=5889 width=4) (actual time=0.034..0.390 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.41 rows=5889 width=0) (actual time=0.029..0.282 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.95 rows=98 width=0) (never executed)
                          ->  Limit  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                ->  Result  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..365.97 rows=98 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.04 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.92 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.689 ms
Execution Time: 1.001 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1780.89 rows=1 width=1301) (actual time=0.301..1.813 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1772.55 rows=1 width=1266) (actual time=0.286..1.659 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1764.24 rows=1 width=1259) (actual time=0.276..1.541 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1763.09 rows=51 width=1259) (actual time=0.274..1.500 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3918.13 rows=115 width=1259) (actual time=0.272..1.491 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3630.30 rows=51 width=1259) (actual time=0.140..1.341 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..419144.43 rows=5889 width=1259) (actual time=0.140..1.334 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..419144.43 rows=5889 width=1251) (actual time=0.133..1.314 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..190.66 rows=51 width=1259) (actual time=0.014..0.018 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..365.97 rows=98 width=1259) (actual time=0.014..0.017 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..365.97 rows=98 width=1251) (actual time=0.013..0.015 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..50.92 rows=12 width=1259) (actual time=0.015..0.016 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..50.92 rows=12 width=1259) (actual time=0.015..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.92 rows=12 width=1251) (actual time=0.014..0.014 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.100..0.104 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.100..0.104 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.074..0.095 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.073..0.093 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.058..0.061 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.048..0.054 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.373 ms
Execution Time: 2.066 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.059..0.063 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.039..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.436 ms
Execution Time: 0.115 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.042..0.046 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.240 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.068..0.070 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.066..0.068 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.054..0.058 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.045..0.049 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.029..0.031 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.011..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.330 ms
Execution Time: 0.152 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=532.48..532.49 rows=1 width=8) (actual time=2.011..2.017 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..519.97 rows=1001 width=4) (actual time=0.060..1.837 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..577.57 rows=1112 width=4) (actual time=0.059..1.643 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..566.45 rows=1112 width=0) (actual time=0.058..1.381 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..98.41 rows=1001 width=0) (actual time=0.057..1.197 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..88.40 rows=1001 width=4) (actual time=0.055..0.985 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..517.41 rows=5889 width=4) (actual time=0.055..0.814 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.41 rows=5889 width=0) (actual time=0.043..0.518 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.95 rows=98 width=0) (never executed)
                          ->  Limit  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                ->  Result  (cost=0.42..365.97 rows=98 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..365.97 rows=98 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.04 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..50.92 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.92 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.478 ms
Execution Time: 2.170 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=465.87..2102.53 rows=1 width=1309) (actual time=3.090..5.778 rows=51 loops=1)
  Buffers: shared hit=1718
  ->  Nested Loop  (cost=465.58..2094.20 rows=1 width=1274) (actual time=3.066..5.445 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1565
        ->  Subquery Scan on x  (cost=465.29..2085.88 rows=1 width=1267) (actual time=3.049..5.141 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1412
              ->  Limit  (cost=465.29..2084.74 rows=51 width=1267) (actual time=3.044..5.055 rows=51 loops=1)
                    Buffers: shared hit=1412
                    ->  Merge Append  (cost=465.29..4116.98 rows=115 width=1267) (actual time=3.042..5.038 rows=51 loops=1)
                          Sort Key: r.assignee_id, r.updated_at, r.id
                          Buffers: shared hit=1412
                          ->  Limit  (cost=0.42..3650.22 rows=51 width=1267) (actual time=0.092..2.052 rows=50 loops=1)
                                Buffers: shared hit=808
                                ->  Result  (cost=0.42..421444.43 rows=5889 width=1267) (actual time=0.092..2.039 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=808
                                      ->  Index Scan Backward using records_list_assignee on records r  (cost=0.42..421444.43 rows=5889 width=1251) (actual time=0.083..2.001 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 727
                                            Buffers: shared hit=808
                          ->  Limit  (cost=369.21..369.34 rows=51 width=1267) (actual time=2.486..2.489 rows=2 loops=1)
                                Buffers: shared hit=523
                                ->  Sort  (cost=369.21..369.46 rows=98 width=1267) (actual time=2.485..2.487 rows=2 loops=1)
                                      Sort Key: r_1.assignee_id, r_1.updated_at, r_1.id
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=523
                                      ->  Result  (cost=0.42..365.97 rows=98 width=1267) (actual time=0.028..1.823 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=523
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..365.97 rows=98 width=1251) (actual time=0.025..1.598 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                          ->  Sort  (cost=51.13..51.16 rows=12 width=1267) (actual time=0.304..0.306 rows=1 loops=1)
                                Sort Key: r_2.assignee_id, r_2.updated_at, r_2.id
                                Sort Method: quicksort  Memory: 127kB
                                Buffers: shared hit=56
                                ->  Limit  (cost=0.42..50.92 rows=12 width=1267) (actual time=0.043..0.221 rows=51 loops=1)
                                      Buffers: shared hit=56
                                      ->  Result  (cost=0.42..50.92 rows=12 width=1267) (actual time=0.040..0.208 rows=51 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=56
                                            ->  Index Scan Backward using records_list_assignee on records r_2  (cost=0.42..50.92 rows=12 width=1251) (actual time=0.028..0.173 rows=51 loops=1)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=56
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1267) (actual time=0.150..0.156 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1267) (actual time=0.149..0.155 rows=1 loops=1)
                                      Sort Key: r_3.assignee_id, r_3.updated_at, r_3.id
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1267) (actual time=0.099..0.144 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.095..0.137 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.071..0.076 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.051..0.061 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.039..0.041 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.017..0.017 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.469 ms
Execution Time: 6.111 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.058..0.062 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.037..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.438 ms
Execution Time: 0.115 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.043..0.046 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.156 ms
Execution Time: 0.082 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.116..0.118 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.114..0.115 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.058..0.063 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.046..0.050 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.031..0.032 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.011..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.362 ms
Execution Time: 0.164 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND r.state = ANY($7::text[]) LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21",["2fd59270-9dd1-4d31-8a87-448af03e03ef"],["new","in_progress"]]`

```
Aggregate  (cost=91.17..91.18 rows=1 width=8) (actual time=1.167..1.169 rows=1 loops=1)
  Buffers: shared hit=532
  ->  Limit  (cost=0.42..78.66 rows=1001 width=4) (actual time=0.074..1.029 rows=1001 loops=1)
        Buffers: shared hit=532
        ->  Result  (cost=0.42..1568.23 rows=20061 width=4) (actual time=0.073..0.812 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=532
              ->  Index Only Scan using records_list_state on records r  (cost=0.42..1568.23 rows=20061 width=0) (actual time=0.064..0.565 rows=1001 loops=1)
                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (state = ANY ('{new,in_progress}'::text[])))
                    Heap Fetches: 0
                    Buffers: shared hit=532
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.316 ms
Execution Time: 1.214 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21","6d529afe-013e-426c-818c-1ebb764c9c21",["2fd59270-9dd1-4d31-8a87-448af03e03ef"],["new","in_progress"],51]`

```
Limit  (cost=1.00..1087.51 rows=51 width=1293) (actual time=0.141..1.762 rows=51 loops=1)
  Buffers: shared hit=635
  ->  Result  (cost=1.00..427383.66 rows=20061 width=1293) (actual time=0.139..1.745 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Buffers: shared hit=635
        ->  Nested Loop Left Join  (cost=1.00..427233.20 rows=20061 width=1266) (actual time=0.116..1.667 rows=51 loops=1)
              Buffers: shared hit=635
              ->  Nested Loop  (cost=0.70..418899.94 rows=20061 width=1258) (actual time=0.094..1.315 rows=51 loops=1)
                    Buffers: shared hit=482
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..418640.89 rows=20061 width=1251) (actual time=0.071..1.238 rows=51 loops=1)
                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                          Filter: (state = ANY ('{new,in_progress}'::text[]))
                          Rows Removed by Filter: 419
                          Buffers: shared hit=479
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.016..0.017 rows=1 loops=1)
                                Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                Filter: (id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.41 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.41 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.865 ms
Execution Time: 1.866 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.059..0.063 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.038..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.439 ms
Execution Time: 0.115 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.045..0.047 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.033..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.158 ms
Execution Time: 0.140 ms
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

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.048..0.050 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.047..0.048 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.038..0.041 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.031..0.033 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.022..0.022 rows=1 loops=1)
                          Index Cond: (tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.334 ms
Execution Time: 0.089 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","a14195dc-d95e-4557-9af1-9ce773ed3120",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],"4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"]]`

```
Aggregate  (cost=34.43..34.44 rows=1 width=8) (actual time=3.408..3.419 rows=1 loops=1)
  Buffers: shared hit=4714
  ->  Limit  (cost=0.55..34.38 rows=4 width=4) (actual time=0.068..3.394 rows=179 loops=1)
        Buffers: shared hit=4714
        ->  Result  (cost=0.55..34.38 rows=4 width=4) (actual time=0.067..3.373 rows=179 loops=1)
              Buffers: shared hit=4714
              ->  Append  (cost=0.55..34.34 rows=4 width=0) (actual time=0.067..3.346 rows=179 loops=1)
                    Buffers: shared hit=4714
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.59 rows=1 width=0) (actual time=0.067..0.292 rows=50 loops=1)
                          Buffers: shared hit=22
                          ->  Limit  (cost=0.55..4.58 rows=1 width=4) (actual time=0.066..0.286 rows=50 loops=1)
                                Buffers: shared hit=22
                                ->  Result  (cost=0.55..4.58 rows=1 width=4) (actual time=0.066..0.281 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=22
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.58 rows=1 width=0) (actual time=0.058..0.267 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid) AND (assignment_group_id IS NOT NULL))
                                            Filter: (assignment_group_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))
                                            Rows Removed by Filter: 953
                                            Heap Fetches: 0
                                            Buffers: shared hit=22
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..8.46 rows=1 width=0) (actual time=0.031..0.714 rows=97 loops=1)
                          Buffers: shared hit=1013
                          ->  Limit  (cost=0.42..8.45 rows=1 width=4) (actual time=0.030..0.701 rows=97 loops=1)
                                Buffers: shared hit=1013
                                ->  Result  (cost=0.42..8.45 rows=1 width=4) (actual time=0.029..0.690 rows=97 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=1013
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..8.45 rows=1 width=0) (actual time=0.023..0.671 rows=97 loops=1)
                                            Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                                            Filter: ((requester_id IS NOT NULL) AND (requester_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 906
                                            Buffers: shared hit=1013
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..8.46 rows=1 width=0) (actual time=0.039..0.510 rows=32 loops=1)
                          Buffers: shared hit=1017
                          ->  Limit  (cost=0.42..8.46 rows=1 width=4) (actual time=0.039..0.505 rows=32 loops=1)
                                Buffers: shared hit=1017
                                ->  Result  (cost=0.42..8.46 rows=1 width=4) (actual time=0.037..0.499 rows=32 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=1017
                                      ->  Index Scan using records_list_requester on records r_2  (cost=0.42..8.46 rows=1 width=0) (actual time=0.030..0.487 rows=32 loops=1)
                                            Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                                            Filter: ((assignee_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND (assignee_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 971
                                            Buffers: shared hit=1017
                    ->  Subquery Scan on "*SELECT* 4"  (cost=0.72..12.80 rows=1 width=0) (actual time=1.801..1.803 rows=0 loops=1)
                          Buffers: shared hit=2662
                          ->  Limit  (cost=0.72..12.79 rows=1 width=4) (actual time=1.800..1.801 rows=0 loops=1)
                                Buffers: shared hit=2662
                                ->  Result  (cost=0.72..12.79 rows=1 width=4) (actual time=1.799..1.800 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=2662
                                      ->  Nested Loop Semi Join  (cost=0.72..12.79 rows=1 width=0) (actual time=1.796..1.796 rows=0 loops=1)
                                            Buffers: shared hit=2662
                                            ->  Index Scan using records_list_assignee on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.764 rows=824 loops=1)
                                                  Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 179
                                                  Buffers: shared hit=1013
                                            ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.32 rows=1 width=32) (actual time=0.001..0.001 rows=0 loops=824)
                                                  Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (record_id = r_3.id))
                                                  Filter: (((principal_kind = 'user'::text) AND (principal_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 0
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1649
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.760 ms
Execution Time: 3.500 ms
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

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","a14195dc-d95e-4557-9af1-9ce773ed3120",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],"4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],51]`

```
Nested Loop Left Join  (cost=2.62..54.99 rows=1 width=1301) (actual time=2.339..3.253 rows=51 loops=1)
  Buffers: shared hit=3859
  ->  Nested Loop  (cost=2.32..46.66 rows=1 width=1266) (actual time=2.317..3.013 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3706
        ->  Subquery Scan on x  (cost=2.04..38.34 rows=1 width=1259) (actual time=2.303..2.849 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=3553
              ->  Limit  (cost=2.04..38.25 rows=4 width=1259) (actual time=2.296..2.796 rows=51 loops=1)
                    Buffers: shared hit=3553
                    ->  Merge Append  (cost=2.04..38.25 rows=4 width=1259) (actual time=2.295..2.787 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=3553
                          ->  Limit  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.115..0.272 rows=14 loops=1)
                                Buffers: shared hit=297
                                ->  Result  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.114..0.270 rows=14 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=297
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.111..0.261 rows=14 loops=1)
                                            Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[])))
                                            Rows Removed by Filter: 278
                                            Buffers: shared hit=297
                          ->  Limit  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.030..0.207 rows=30 loops=1)
                                Buffers: shared hit=294
                                ->  Result  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.029..0.202 rows=30 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=294
                                      ->  Index Scan using records_list_updated on records r_1  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.025..0.188 rows=30 loops=1)
                                            Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                                            Filter: ((requester_id IS NOT NULL) AND (requester_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 259
                                            Buffers: shared hit=294
                          ->  Limit  (cost=0.42..8.46 rows=1 width=1259) (actual time=0.028..0.171 rows=9 loops=1)
                                Buffers: shared hit=299
                                ->  Result  (cost=0.42..8.46 rows=1 width=1259) (actual time=0.028..0.169 rows=9 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=299
                                      ->  Index Scan using records_list_updated on records r_2  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.025..0.164 rows=9 loops=1)
                                            Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                                            Filter: ((assignee_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND (assignee_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 285
                                            Buffers: shared hit=299
                          ->  Limit  (cost=0.72..12.79 rows=1 width=1259) (actual time=2.119..2.121 rows=0 loops=1)
                                Buffers: shared hit=2663
                                ->  Result  (cost=0.72..12.79 rows=1 width=1259) (actual time=2.118..2.120 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                                      Buffers: shared hit=2663
                                      ->  Nested Loop Semi Join  (cost=0.72..12.79 rows=1 width=1251) (actual time=2.116..2.117 rows=0 loops=1)
                                            Buffers: shared hit=2663
                                            ->  Index Scan using records_list_updated on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.005..0.978 rows=824 loops=1)
                                                  Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 179
                                                  Buffers: shared hit=1014
                                            ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.32 rows=1 width=32) (actual time=0.001..0.001 rows=0 loops=824)
                                                  Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (record_id = r_3.id))
                                                  Filter: (((principal_kind = 'user'::text) AND (principal_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 0
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1649
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.243 ms
Execution Time: 3.375 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.061..0.065 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.041..0.044 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.441 ms
Execution Time: 0.118 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.044 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.031..0.033 rows=1 loops=1)
        Index Cond: (tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.161 ms
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

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.148..0.150 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.146..0.148 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.132..0.137 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.122..0.127 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.033..0.035 rows=1 loops=1)
                          Index Cond: (tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.083..0.085 rows=1 loops=1)
                          Index Cond: (tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.345 ms
Execution Time: 0.199 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","a14195dc-d95e-4557-9af1-9ce773ed3120","d5a788de-9f13-4a8c-aa4c-e471908a90b0","d5a788de-9f13-4a8c-aa4c-e471908a90b0","d5a788de-9f13-4a8c-aa4c-e471908a90b0",["09e9f886-9f90-424a-b5bf-780040f9937c"]]`

```
Aggregate  (cost=4.46..4.47 rows=1 width=8) (actual time=1.003..1.004 rows=1 loops=1)
  Buffers: shared hit=21
  ->  Limit  (cost=0.42..4.44 rows=1 width=4) (actual time=0.057..0.835 rows=1001 loops=1)
        Buffers: shared hit=21
        ->  Result  (cost=0.42..4.44 rows=1 width=4) (actual time=0.055..0.659 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
              Buffers: shared hit=21
              ->  Index Only Scan using records_list_assignee on records r  (cost=0.42..4.44 rows=1 width=0) (actual time=0.044..0.343 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=21
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.282 ms
Execution Time: 1.055 ms
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

Arguments: `["2b013030-bb35-4a4c-b5a4-ec1a5d35cb39","a14195dc-d95e-4557-9af1-9ce773ed3120","d5a788de-9f13-4a8c-aa4c-e471908a90b0","d5a788de-9f13-4a8c-aa4c-e471908a90b0","d5a788de-9f13-4a8c-aa4c-e471908a90b0",["09e9f886-9f90-424a-b5bf-780040f9937c"],51]`

```
Limit  (cost=1.00..25.09 rows=1 width=1293) (actual time=0.112..0.766 rows=51 loops=1)
  Buffers: shared hit=361
  ->  Result  (cost=1.00..25.09 rows=1 width=1293) (actual time=0.110..0.749 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
        Buffers: shared hit=361
        ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.086..0.671 rows=51 loops=1)
              Buffers: shared hit=361
              ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.065..0.345 rows=51 loops=1)
                    Buffers: shared hit=208
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.046..0.166 rows=51 loops=1)
                          Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (entity_definition_id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid))
                          Buffers: shared hit=55
                    ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Index Cond: (tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                          Filter: (id = 'a14195dc-d95e-4557-9af1-9ce773ed3120'::uuid)
                          Buffers: shared hit=153
              ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '2b013030-bb35-4a4c-b5a4-ec1a5d35cb39'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.778 ms
Execution Time: 0.917 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.038..0.040 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.025..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.260 ms
Execution Time: 0.082 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.031 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.023..0.024 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.105 ms
Execution Time: 0.053 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.059..0.061 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.058..0.059 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.043..0.051 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.035..0.042 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.023..0.024 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.009..0.013 rows=2 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.223 ms
Execution Time: 0.098 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=3628.16..3628.17 rows=1 width=8) (actual time=2.623..2.630 rows=1 loops=1)
  Buffers: shared hit=1023
  ->  Limit  (cost=0.55..3615.64 rows=1001 width=4) (actual time=0.045..2.529 rows=1001 loops=1)
        Buffers: shared hit=1023
        ->  Result  (cost=0.55..4016.52 rows=1112 width=4) (actual time=0.044..2.388 rows=1001 loops=1)
              Buffers: shared hit=1023
              ->  Append  (cost=0.55..4005.40 rows=1112 width=0) (actual time=0.043..2.235 rows=1001 loops=1)
                    Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..3537.08 rows=1001 width=0) (actual time=0.043..2.108 rows=1001 loops=1)
                          Buffers: shared hit=1023
                          ->  Limit  (cost=0.55..3527.07 rows=1001 width=4) (actual time=0.042..1.993 rows=1001 loops=1)
                                Buffers: shared hit=1023
                                ->  Result  (cost=0.55..20736.89 rows=5886 width=4) (actual time=0.042..1.865 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=1023
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..20736.89 rows=5886 width=0) (actual time=0.034..1.725 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 1
                                            Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..367.20 rows=98 width=0) (never executed)
                          ->  Limit  (cost=0.42..366.22 rows=98 width=4) (never executed)
                                ->  Result  (cost=0.42..366.22 rows=98 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=98 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.07 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.95 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..50.95 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.874 ms
Execution Time: 2.740 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1783.72 rows=1 width=1301) (actual time=0.289..1.951 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1775.39 rows=1 width=1266) (actual time=0.269..1.783 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1767.07 rows=1 width=1259) (actual time=0.256..1.657 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1765.93 rows=51 width=1259) (actual time=0.254..1.613 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3924.51 rows=115 width=1259) (actual time=0.253..1.605 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3636.51 rows=51 width=1259) (actual time=0.114..1.445 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..419647.97 rows=5886 width=1259) (actual time=0.113..1.438 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..419647.97 rows=5886 width=1251) (actual time=0.110..1.420 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..190.79 rows=51 width=1259) (actual time=0.016..0.021 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..366.22 rows=98 width=1259) (actual time=0.016..0.020 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=98 width=1251) (actual time=0.014..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..50.95 rows=12 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..50.95 rows=12 width=1259) (actual time=0.016..0.016 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=12 width=1251) (actual time=0.014..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.104..0.109 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.104..0.108 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.068..0.093 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.066..0.090 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.046..0.050 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.042 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.371 ms
Execution Time: 2.148 ms
```

### composed: agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.BJkdT4WmdSnIZySjSLIq--qvP2ZFz-fjLPw-PdVDlCw` as `agent-1`

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.032..0.035 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.020..0.021 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.007..0.007 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.295 ms
Execution Time: 0.067 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.023..0.026 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.019..0.020 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.102 ms
Execution Time: 0.043 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.046..0.047 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.045..0.046 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.030..0.038 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.026..0.032 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.015..0.016 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.008..0.012 rows=2 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.216 ms
Execution Time: 0.075 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=3628.16..3628.17 rows=1 width=8) (actual time=2.802..2.808 rows=1 loops=1)
  Buffers: shared hit=1023
  ->  Limit  (cost=0.55..3615.64 rows=1001 width=4) (actual time=0.052..2.641 rows=1001 loops=1)
        Buffers: shared hit=1023
        ->  Result  (cost=0.55..4016.52 rows=1112 width=4) (actual time=0.051..2.484 rows=1001 loops=1)
              Buffers: shared hit=1023
              ->  Append  (cost=0.55..4005.40 rows=1112 width=0) (actual time=0.050..2.297 rows=1001 loops=1)
                    Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..3537.08 rows=1001 width=0) (actual time=0.049..2.143 rows=1001 loops=1)
                          Buffers: shared hit=1023
                          ->  Limit  (cost=0.55..3527.07 rows=1001 width=4) (actual time=0.048..2.002 rows=1001 loops=1)
                                Buffers: shared hit=1023
                                ->  Result  (cost=0.55..20736.89 rows=5886 width=4) (actual time=0.048..1.888 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=1023
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..20736.89 rows=5886 width=0) (actual time=0.040..1.721 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 1
                                            Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..367.20 rows=98 width=0) (never executed)
                          ->  Limit  (cost=0.42..366.22 rows=98 width=4) (never executed)
                                ->  Result  (cost=0.42..366.22 rows=98 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.22 rows=98 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.07 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.95 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..50.95 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.95 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.031 ms
Execution Time: 2.921 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.39..1811.43 rows=1 width=1301) (actual time=0.216..2.075 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1803.10 rows=1 width=1266) (actual time=0.199..1.880 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.81..1794.79 rows=1 width=1259) (actual time=0.188..1.734 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.81..1793.64 rows=51 width=1259) (actual time=0.186..1.682 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.81..3987.00 rows=115 width=1259) (actual time=0.184..1.671 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..3701.89 rows=51 width=1259) (actual time=0.046..1.503 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..409991.70 rows=5649 width=1259) (actual time=0.045..1.495 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..409991.70 rows=5649 width=1251) (actual time=0.041..1.474 rows=49 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..191.39 rows=51 width=1259) (actual time=0.019..0.024 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..352.40 rows=94 width=1259) (actual time=0.019..0.023 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..352.40 rows=94 width=1251) (actual time=0.016..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..47.46 rows=12 width=1259) (actual time=0.020..0.025 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..47.46 rows=12 width=1259) (actual time=0.020..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..47.46 rows=12 width=1251) (actual time=0.018..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.50 rows=1 width=1259) (actual time=0.096..0.102 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.50 rows=1 width=1259) (actual time=0.096..0.101 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.063..0.092 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.062..0.088 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.045..0.049 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.578 ms
Execution Time: 2.267 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.055..0.059 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.037..0.039 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.010..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.423 ms
Execution Time: 0.115 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.036..0.038 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.027..0.029 rows=1 loops=1)
        Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.125 ms
Execution Time: 0.070 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.060..0.061 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.059..0.060 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.043..0.051 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.035..0.042 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.025..0.026 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.008..0.012 rows=2 loops=1)
                          Index Cond: (tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.301 ms
Execution Time: 0.098 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($9::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=1529.47..1529.48 rows=1 width=8) (actual time=3.113..3.120 rows=1 loops=1)
  Buffers: shared hit=1040
  ->  Limit  (cost=0.55..1525.77 rows=296 width=4) (actual time=0.058..2.990 rows=1001 loops=1)
        Buffers: shared hit=1040
        ->  Result  (cost=0.55..1525.77 rows=296 width=4) (actual time=0.057..2.841 rows=1001 loops=1)
              Buffers: shared hit=1040
              ->  Append  (cost=0.55..1522.81 rows=296 width=0) (actual time=0.056..2.644 rows=1001 loops=1)
                    Buffers: shared hit=1040
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..1059.47 rows=289 width=0) (actual time=0.056..2.485 rows=1001 loops=1)
                          Buffers: shared hit=1040
                          ->  Limit  (cost=0.55..1056.58 rows=289 width=4) (actual time=0.055..2.338 rows=1001 loops=1)
                                Buffers: shared hit=1040
                                ->  Result  (cost=0.55..1056.58 rows=289 width=4) (actual time=0.054..2.218 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=1040
                                      ->  Index Scan using records_list_group_state on records r  (cost=0.55..1056.58 rows=289 width=0) (actual time=0.046..2.042 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 6
                                            Buffers: shared hit=1040
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..366.40 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.42..366.35 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.42..366.35 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.35 rows=5 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (state = ANY ('{new}'::text[]))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..50.98 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..50.97 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..50.97 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.97 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.242 ms
Execution Time: 3.243 ms
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

Arguments: `["e5a182fe-5737-4278-83a6-e7106315eceb","9f6f5444-8fc9-45ec-af5a-eec183a99556",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=1112.18..1496.48 rows=1 width=1301) (actual time=4.055..4.482 rows=51 loops=1)
  Buffers: shared hit=1585
  ->  Nested Loop  (cost=1111.88..1488.14 rows=1 width=1266) (actual time=3.989..4.251 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1432
        ->  Subquery Scan on x  (cost=1111.60..1479.83 rows=1 width=1259) (actual time=3.973..4.108 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1279
              ->  Limit  (cost=1111.60..1478.68 rows=51 width=1259) (actual time=3.970..4.061 rows=51 loops=1)
                    Buffers: shared hit=1279
                    ->  Merge Append  (cost=1111.60..1529.07 rows=58 width=1259) (actual time=3.969..4.053 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1279
                          ->  Limit  (cost=1066.22..1066.35 rows=51 width=1259) (actual time=3.752..3.769 rows=50 loops=1)
                                Buffers: shared hit=1204
                                ->  Sort  (cost=1066.22..1066.95 rows=289 width=1259) (actual time=3.751..3.762 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=1204
                                      ->  Result  (cost=0.55..1056.58 rows=289 width=1259) (actual time=0.049..2.838 rows=1162 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=1204
                                            ->  Index Scan using records_list_group_state on records r  (cost=0.55..1056.58 rows=289 width=1251) (actual time=0.044..2.453 rows=1162 loops=1)
                                                  Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                  Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Rows Removed by Filter: 6
                                                  Buffers: shared hit=1204
                          ->  Limit  (cost=0.42..366.35 rows=5 width=1259) (actual time=0.032..0.075 rows=2 loops=1)
                                Buffers: shared hit=30
                                ->  Result  (cost=0.42..366.35 rows=5 width=1259) (actual time=0.031..0.074 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=30
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..366.35 rows=5 width=1251) (actual time=0.024..0.066 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (state = ANY ('{new}'::text[]))
                                            Rows Removed by Filter: 24
                                            Buffers: shared hit=30
                          ->  Limit  (cost=0.42..50.97 rows=1 width=1259) (actual time=0.074..0.075 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..50.97 rows=1 width=1259) (actual time=0.074..0.074 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..50.97 rows=1 width=1251) (actual time=0.071..0.072 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.105..0.108 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.105..0.107 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.097..0.101 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.095..0.098 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.056 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.047 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'e5a182fe-5737-4278-83a6-e7106315eceb'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '9f6f5444-8fc9-45ec-af5a-eec183a99556'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.752 ms
Execution Time: 4.658 ms
```

### composed: single GET, agent

Request: `GET /api/v1/records/ticket/3f94a2ac-38ee-af8f-f434-143a689a293d` as `agent-1`


## Host

```
### 2026-11-19T22:50:43Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 26260877312 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: buildx_buildkit_nava-ci0 multica-backend-1 multica-frontend-1 multica-postgres-1 
```
