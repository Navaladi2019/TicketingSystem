# P3-36 raw report (TICK-621)

Generated 2026-11-10T15:40:15Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 822166, open 142444, priority 1 40021, avg data bytes 1074
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 416518
record_shares 15815
heap 1071 MB, indexes 781 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.17 / p95 2.20 / p99 5.78 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row is scored only when its commit p50 (the create's own commit round trip, from its trace) is at or under 10 ms; a gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts (storage-window gate, CTO 2026-11-10). Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Commit p50 | Round trips | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 1 | large | 499 | 34.87 | 70.53 | 119.60 | 10.96 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, agent (1 client) | 2 | large | 518 | 35.05 | 59.90 | 139.32 | 8.15 | 6 | 50 | **MISS** |
| create, agent | 1 | large | 2261 | 59.77 | 127.54 | 228.87 | 12.11 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, agent | 2 | large | 2837 | 49.58 | 101.48 | 143.06 | 10.36 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, agent | 3 | large | 3194 | 44.20 | 83.50 | 125.74 | 10.08 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, admin | 1 | large | 2496 | 51.66 | 133.74 | 194.88 | 11.08 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, admin | 2 | large | 1723 | 75.55 | 204.81 | 321.03 | 13.17 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, admin | 3 | large | 1661 | 82.77 | 186.68 | 288.14 | 22.72 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, small tenant agent | 1 | small | 1397 | 79.03 | 293.15 | 632.59 | 14.62 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, small tenant agent | 2 | small | 1724 | 76.33 | 187.57 | 328.40 | 13.97 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, small tenant agent | 3 | small | 2374 | 60.05 | 120.00 | 173.13 | 11.89 | 6 | 50 | gated (commit p50 > 10 ms) |
| create, agent, gapless type | 1 | large | 1017 | 121.40 | 383.02 | 617.52 | 12.55 | 5 | 0 | gapless type: reported, not scored (D-3-9) |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.17 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client), attempt 1 | 499 | 2.81 / 7.16 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent (1 client), attempt 2 | 518 | 3.07 / 6.58 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, attempt 1 | 2261 | 5.20 / 15.79 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, attempt 2 | 2837 | 3.83 / 12.43 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, attempt 3 | 3194 | 3.56 / 8.81 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, admin, attempt 1 | 2496 | 4.30 / 13.76 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, admin, attempt 2 | 1723 | 7.05 / 22.84 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, admin, attempt 3 | 1661 | 5.94 / 19.27 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, small tenant agent, attempt 1 | 1397 | 7.76 / 32.94 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, small tenant agent, attempt 2 | 1724 | 7.22 / 22.48 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, small tenant agent, attempt 3 | 2374 | 5.22 / 14.55 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, gapless type | 1017 | 58.79 / 291.39 | 13.57 / 36.22 | 1 | 1.17 / 1.17 | 11.38 / 33.17 | 0.68 / 3.44 |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |
| create, agent (1 client), attempt 1 | 549 | 0.07 / 0.25 / 0.48 | 3.32 / 2.89 / 8.44 | 0 | none |
| create, agent (1 client), attempt 2 | 568 | 0.01 / 0.25 / 0.48 | 3.65 / 3.37 / 8.53 | 0 | none |
| create, agent, attempt 1 | 2312 | 0.07 / 0.26 / 0.49 | 7.13 / 5.59 / 21.79 | 1 | 503 retry_later x1 |
| create, agent, attempt 2 | 2887 | 0.05 / 0.26 / 0.49 | 5.29 / 4.19 / 17.74 | 0 | none |
| create, agent, attempt 3 | 3244 | 0.05 / 0.26 / 0.49 | 4.46 / 3.93 / 9.76 | 0 | none |
| create, admin, attempt 1 | 2497 | 0.05 / 0.26 / 0.48 | 6.04 / 4.57 / 20.25 | 0 | none |
| create, admin, attempt 2 | 1724 | 0.10 / 0.26 / 0.49 | 9.82 / 7.68 / 24.66 | 0 | none |
| create, admin, attempt 3 | 1662 | 0.20 / 0.26 / 0.54 | 8.45 / 6.65 / 23.68 | 0 | none |
| create, small tenant agent, attempt 1 | 1448 | 0.20 / 0.26 / 0.49 | 12.44 / 8.51 / 40.77 | 1 | 503 retry_later x1 |
| create, small tenant agent, attempt 2 | 1774 | 0.11 / 0.26 / 0.49 | 9.93 / 7.91 / 24.63 | 0 | none |
| create, small tenant agent, attempt 3 | 2424 | 0.06 / 0.26 / 0.49 | 6.75 / 5.64 / 20.03 | 0 | none |
| create, agent, gapless type | 0 | - | - | 0 | none |

## Data-filter statement timeout (TICK-647)

Default timeout 2s (store.DataFilterTimeout); the bound check runs with 2ms. A 422 is `filter_too_expensive`. Latency includes the 422s.

| Scenario | Requests | 422 | p50 | p95 | max |
| --- | --- | --- | --- | --- | --- |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### create, agent (1 client), attempt 1

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 499 | 2.83 | 7.85 | 14.89 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 499 | 5.39 | 12.79 | 22.57 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 499 | 3.06 | 7.62 | 14.10 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 499 | 2.81 | 7.16 | 13.96 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 499 | 4.65 | 9.83 | 16.02 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 499 | 10.96 | 22.98 | 34.99 |

### create, agent (1 client), attempt 2

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 518 | 2.97 | 5.38 | 11.39 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 518 | 6.26 | 11.69 | 20.75 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 518 | 3.56 | 6.59 | 12.63 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 518 | 3.07 | 6.58 | 13.97 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 518 | 5.04 | 9.35 | 17.73 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 518 | 8.15 | 13.29 | 21.85 |

### create, agent, attempt 1

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 2261 | 4.66 | 16.37 | 37.52 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 2261 | 10.22 | 30.49 | 53.91 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2261 | 5.51 | 15.67 | 29.26 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 2261 | 5.20 | 15.79 | 33.63 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 2261 | 7.67 | 19.60 | 34.41 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2261 | 12.11 | 24.29 | 44.95 |

### create, agent, attempt 2

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 2837 | 3.77 | 12.01 | 20.90 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 2837 | 8.19 | 25.21 | 37.88 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2837 | 4.26 | 13.05 | 23.46 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 2837 | 3.83 | 12.43 | 21.96 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 2837 | 5.61 | 14.52 | 26.34 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2837 | 10.36 | 18.60 | 35.03 |

### create, agent, attempt 3

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 3194 | 3.16 | 9.06 | 16.34 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 3194 | 7.50 | 21.07 | 35.95 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 3194 | 3.83 | 10.05 | 16.15 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 3194 | 3.56 | 8.81 | 14.66 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 3194 | 5.50 | 11.64 | 18.02 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 3194 | 10.08 | 16.07 | 25.76 |

### create, admin, attempt 1

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 2496 | 4.00 | 14.37 | 34.32 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 2496 | 9.27 | 27.86 | 51.91 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2496 | 4.66 | 14.10 | 31.19 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 2496 | 4.30 | 13.76 | 23.09 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 2496 | 6.30 | 16.60 | 30.96 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2496 | 11.08 | 20.99 | 40.10 |

### create, admin, attempt 2

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 1723 | 6.49 | 23.02 | 49.92 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 1723 | 13.49 | 42.52 | 86.40 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1723 | 7.20 | 22.08 | 42.63 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 1723 | 7.05 | 22.84 | 43.75 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 1723 | 8.65 | 25.12 | 48.95 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1723 | 13.17 | 29.61 | 56.57 |

### create, admin, attempt 3

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 1661 | 5.86 | 19.37 | 41.37 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 1661 | 11.87 | 40.21 | 76.09 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1661 | 6.30 | 18.81 | 33.07 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 1661 | 5.94 | 19.27 | 40.13 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 1661 | 7.78 | 22.12 | 34.06 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1661 | 22.72 | 76.88 | 110.46 |

### create, small tenant agent, attempt 1

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '16d2b05e-064d-49ae-814b-3c8c08f21fa5', true)` | 4 | 3.88 | 17.96 | 17.96 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 1397 | 13.71 | 60.85 | 135.04 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1397 | 7.67 | 30.90 | 65.81 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 1397 | 7.76 | 32.94 | 64.71 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 1397 | 10.39 | 36.68 | 75.41 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1397 | 14.62 | 41.29 | 91.10 |

### create, small tenant agent, attempt 2

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '16d2b05e-064d-49ae-814b-3c8c08f21fa5', true)` | 5 | 4.41 | 16.30 | 16.30 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 1724 | 13.29 | 40.82 | 83.38 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1724 | 7.49 | 22.76 | 57.46 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 1724 | 7.22 | 22.48 | 52.01 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 1724 | 9.50 | 26.36 | 69.08 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1724 | 13.97 | 30.48 | 63.26 |

### create, small tenant agent, attempt 3

Round trips: 6, statements: 18, statements per batch: [6 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '16d2b05e-064d-49ae-814b-3c8c08f21fa5', true)` | 6 | 4.16 | 26.18 | 26.18 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 2374 | 10.14 | 29.17 | 48.39 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2374 | 5.46 | 14.60 | 29.84 |
| 4 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 2374 | 5.22 | 14.55 | 26.28 |
| 5 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 2374 | 7.21 | 17.20 | 34.47 |
| 6 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2374 | 11.89 | 24.04 | 38.30 |

### create, agent, gapless type

Round trips: 5, statements: 14, statements per batch: [6 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)` | 1017 | 6.07 | 21.37 | 38.27 |
| 2 | `BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_…` | 1017 | 11.92 | 36.24 | 59.18 |
| 3 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1017 | 6.60 | 21.16 | 46.44 |
| 4 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 1017 | 58.79 | 291.39 | 471.26 |
| 5 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1017 | 12.55 | 34.34 | 63.86 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### create, agent (1 client), attempt 1

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent (1 client), attempt 2

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent, attempt 1

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent, attempt 2

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent, attempt 3

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, admin, attempt 1

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, admin, attempt 2

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, admin, attempt 3

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, small tenant agent, attempt 1

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '16d2b05e-064d-49ae-814b-3c8c08f21fa5', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, small tenant agent, attempt 2

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '16d2b05e-064d-49ae-814b-3c8c08f21fa5', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, small tenant agent, attempt 3

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '16d2b05e-064d-49ae-814b-3c8c08f21fa5', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent, gapless type

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '286b9503-61bc-41a7-87ed-b15c1259a7a6', true)
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 ; INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4), $6, $7, $8, $9, $10, $11, $11, $12, $13, $14) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

