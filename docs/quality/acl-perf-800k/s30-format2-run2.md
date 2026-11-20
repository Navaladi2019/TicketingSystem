# P3-36 raw report (TICK-621)

Generated 2026-11-19T23:25:54Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.12 / p95 1.83 / p99 2.35 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_logical_emit_message(true, ...); COMMIT` probe, a commit that flushes WAL with no INSERT, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). An attempt is also void if a container other than the run's database, the buildkit builder and the three Multica platform containers runs during it. A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| agent list by group, default sort, page 1 | - | large | 2010 | 72.63 | 124.36 | 143.51 | - | - | - | - | - | 150 | meets |
| agent list by group, default sort, page 20 | - | large | 1549 | 80.97 | 246.30 | 428.85 | - | - | - | - | - | 150 | **MISS** |
| requester list (heavy requesters), page 1 | - | large | 2290 | 63.53 | 111.35 | 143.67 | - | - | - | - | - | 150 | meets |
| requester list (heavy requesters), page 20 | - | large | 2283 | 63.58 | 113.86 | 143.46 | - | - | - | - | - | 150 | meets |
| admin list, default sort, page 1 | - | large | 2411 | 60.88 | 111.57 | 146.47 | - | - | - | - | - | 150 | meets |
| admin list, default sort, page 20 | - | large | 2138 | 67.71 | 130.23 | 164.14 | - | - | - | - | - | 150 | meets |
| composed: agent list by group, default sort, page 1 | - | large | 2271 | 63.55 | 108.90 | 135.83 | - | - | - | - | - | 150 | meets |
| composed: agent list by group, default sort, page 20 | - | large | 2327 | 63.21 | 102.12 | 118.82 | - | - | - | - | - | 150 | meets |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.12 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

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

## Per statement (ms, as pgx traced it, all requests of the scenario)

### agent list by group, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 2010 | 4.31 | 12.72 | 21.68 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2010 | 4.99 | 13.17 | 26.01 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2010 | 4.38 | 12.27 | 22.81 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2010 | 4.42 | 12.61 | 22.06 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2010 | 6.71 | 14.77 | 31.89 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2010 | 20.86 | 54.00 | 74.89 |
| 7 | `commit` | 2010 | 4.74 | 13.67 | 29.24 |

### agent list by group, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 1549 | 4.76 | 19.24 | 42.12 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1549 | 5.15 | 19.31 | 32.47 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1549 | 4.52 | 20.41 | 39.82 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1549 | 4.62 | 18.39 | 49.36 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1549 | 6.91 | 22.48 | 58.26 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1549 | 23.07 | 82.59 | 169.65 |
| 7 | `commit` | 1549 | 5.19 | 22.34 | 42.46 |

### requester list (heavy requesters), page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 2290 | 3.92 | 12.15 | 22.24 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2290 | 4.54 | 11.94 | 23.42 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2290 | 4.12 | 13.08 | 27.26 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2290 | 4.16 | 12.37 | 23.23 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2290 | 5.91 | 13.53 | 27.28 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2290 | 16.78 | 44.27 | 65.54 |
| 7 | `commit` | 2290 | 4.28 | 12.52 | 21.94 |

### requester list (heavy requesters), page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 2283 | 3.83 | 12.32 | 24.95 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2283 | 4.40 | 11.86 | 21.74 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2283 | 3.83 | 12.59 | 23.27 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2283 | 4.04 | 10.47 | 19.72 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2283 | 5.67 | 13.99 | 24.19 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2283 | 16.46 | 46.50 | 71.20 |
| 7 | `commit` | 2283 | 4.14 | 14.62 | 32.76 |

### admin list, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 2411 | 3.98 | 12.94 | 22.49 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2411 | 4.67 | 12.57 | 23.90 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2411 | 4.23 | 12.46 | 22.83 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2411 | 4.40 | 12.47 | 23.24 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 2411 | 4.64 | 13.83 | 26.61 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 2411 | 15.07 | 43.34 | 66.64 |
| 7 | `commit` | 2411 | 3.95 | 13.65 | 23.26 |

### admin list, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 2138 | 4.59 | 13.71 | 28.07 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2138 | 5.09 | 13.96 | 23.75 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2138 | 4.53 | 14.67 | 30.43 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2138 | 4.83 | 12.68 | 26.79 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 2138 | 4.91 | 15.26 | 32.15 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 2138 | 16.79 | 49.33 | 71.14 |
| 7 | `commit` | 2138 | 4.52 | 15.14 | 30.23 |

### composed: agent list by group, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 2271 | 3.50 | 11.18 | 20.27 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2271 | 4.18 | 10.42 | 17.93 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2271 | 3.54 | 9.67 | 17.54 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2271 | 3.74 | 9.26 | 19.87 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2271 | 8.76 | 17.05 | 28.70 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2271 | 17.92 | 42.36 | 66.07 |
| 7 | `commit` | 2271 | 3.70 | 12.00 | 22.50 |

### composed: agent list by group, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2a552b54-08bb-4a4b-945c-a3f8d37b056b', true)` | 2327 | 3.38 | 9.52 | 19.40 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2327 | 4.14 | 9.30 | 17.41 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2327 | 3.50 | 9.15 | 17.94 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2327 | 3.63 | 8.63 | 15.09 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2327 | 8.63 | 15.51 | 21.80 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2327 | 18.56 | 40.20 | 61.21 |
| 7 | `commit` | 2327 | 3.69 | 11.36 | 22.18 |

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.057..0.061 rows=1 loops=1)
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
Planning Time: 0.445 ms
Execution Time: 0.114 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.044..0.048 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.163 ms
Execution Time: 0.141 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.073..0.075 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.071..0.072 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.059..0.063 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.047..0.052 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.031..0.033 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.011..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.404 ms
Execution Time: 0.125 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=489.95..489.96 rows=1 width=8) (actual time=2.021..2.027 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..477.43 rows=1001 width=4) (actual time=0.081..1.848 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..529.84 rows=1111 width=4) (actual time=0.080..1.618 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..518.73 rows=1111 width=0) (actual time=0.079..1.359 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.71 rows=1001 width=0) (actual time=0.078..1.181 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.70 rows=1001 width=4) (actual time=0.076..0.972 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..519.10 rows=5956 width=4) (actual time=0.076..0.801 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..519.10 rows=5956 width=0) (actual time=0.064..0.553 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..324.63 rows=97 width=0) (never executed)
                          ->  Limit  (cost=0.42..323.66 rows=97 width=4) (never executed)
                                ->  Result  (cost=0.42..323.66 rows=97 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..323.66 rows=97 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..46.34 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..46.22 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..46.22 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..46.22 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.530 ms
Execution Time: 2.326 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1581.06 rows=1 width=1301) (actual time=0.481..2.914 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1572.72 rows=1 width=1266) (actual time=0.444..2.618 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1564.41 rows=1 width=1259) (actual time=0.420..2.390 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1563.26 rows=51 width=1259) (actual time=0.417..2.313 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3467.53 rows=115 width=1259) (actual time=0.415..2.298 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3204.68 rows=51 width=1259) (actual time=0.174..2.024 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..374206.93 rows=5956 width=1259) (actual time=0.173..2.012 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..374206.93 rows=5956 width=1251) (actual time=0.164..1.977 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..170.38 rows=51 width=1259) (actual time=0.038..0.045 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..323.66 rows=97 width=1259) (actual time=0.037..0.044 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..323.66 rows=97 width=1251) (actual time=0.034..0.039 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..46.22 rows=12 width=1259) (actual time=0.034..0.036 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..46.22 rows=12 width=1259) (actual time=0.034..0.035 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..46.22 rows=12 width=1251) (actual time=0.031..0.031 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.165..0.172 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.164..0.170 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.113..0.150 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.110..0.145 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.076..0.081 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.057..0.068 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.045..0.046 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.023..0.023 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.019..0.019 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.762 ms
Execution Time: 3.317 ms
```

### agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.KSGu2SEzDw2JzDODruoDKlDAAEvwN1-X2WIsJMsQ-j0` as `agent-1`

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.058..0.063 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.039..0.042 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.475 ms
Execution Time: 0.115 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.039..0.042 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.030..0.032 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.070..0.072 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.069..0.070 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.056..0.061 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.047..0.051 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.032..0.033 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.011..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.339 ms
Execution Time: 0.120 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=489.95..489.96 rows=1 width=8) (actual time=2.135..2.142 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..477.43 rows=1001 width=4) (actual time=0.066..1.909 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..529.84 rows=1111 width=4) (actual time=0.065..1.667 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..518.73 rows=1111 width=0) (actual time=0.064..1.396 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.71 rows=1001 width=0) (actual time=0.063..1.214 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.70 rows=1001 width=4) (actual time=0.062..1.001 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..519.10 rows=5956 width=4) (actual time=0.061..0.830 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..519.10 rows=5956 width=0) (actual time=0.049..0.578 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..324.63 rows=97 width=0) (never executed)
                          ->  Limit  (cost=0.42..323.66 rows=97 width=4) (never executed)
                                ->  Result  (cost=0.42..323.66 rows=97 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..323.66 rows=97 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..46.34 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..46.22 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..46.22 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..46.22 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.606 ms
Execution Time: 2.312 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.38..1604.19 rows=1 width=1301) (actual time=0.211..1.985 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1595.85 rows=1 width=1266) (actual time=0.194..1.809 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.80..1587.54 rows=1 width=1259) (actual time=0.182..1.676 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.80..1586.39 rows=51 width=1259) (actual time=0.179..1.630 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.80..3519.68 rows=115 width=1259) (actual time=0.178..1.620 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..3259.27 rows=51 width=1259) (actual time=0.043..1.456 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..365949.72 rows=5727 width=1259) (actual time=0.043..1.449 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..365949.72 rows=5727 width=1251) (actual time=0.039..1.429 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..171.00 rows=51 width=1259) (actual time=0.017..0.023 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..311.48 rows=93 width=1259) (actual time=0.017..0.021 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..311.48 rows=93 width=1251) (actual time=0.015..0.018 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..43.15 rows=12 width=1259) (actual time=0.017..0.022 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..43.15 rows=12 width=1259) (actual time=0.016..0.021 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..43.15 rows=12 width=1251) (actual time=0.015..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.098..0.103 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.098..0.103 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.061..0.092 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.060..0.088 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.045..0.048 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.752 ms
Execution Time: 2.197 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.040..0.042 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.027..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.288 ms
Execution Time: 0.077 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.030..0.032 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.022..0.024 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.107 ms
Execution Time: 0.057 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.057..0.059 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.056..0.057 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.046..0.049 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.037..0.040 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.245 ms
Execution Time: 0.093 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=191.61..191.62 rows=1 width=8) (actual time=1.296..1.299 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..179.10 rows=1001 width=4) (actual time=0.048..1.182 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..181.59 rows=1015 width=4) (actual time=0.047..1.056 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..171.44 rows=1015 width=0) (actual time=0.046..0.853 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..98.97 rows=1001 width=0) (actual time=0.045..0.737 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..88.96 rows=1001 width=4) (actual time=0.044..0.606 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..123.10 rows=1387 width=4) (actual time=0.044..0.500 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..123.10 rows=1387 width=0) (actual time=0.036..0.342 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..46.29 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..46.16 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..46.16 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..46.16 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.801 ms
Execution Time: 1.417 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],51]`

```
Nested Loop Left Join  (cost=22.56..203.91 rows=1 width=1301) (actual time=0.164..0.807 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..195.57 rows=1 width=1266) (actual time=0.139..0.538 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.98..187.26 rows=1 width=1259) (actual time=0.115..0.359 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.98..186.11 rows=51 width=1259) (actual time=0.113..0.303 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.98..231.16 rows=65 width=1259) (actual time=0.111..0.291 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..163.03 rows=51 width=1259) (actual time=0.050..0.215 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..4422.54 rows=1387 width=1259) (actual time=0.050..0.206 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..4422.54 rows=1387 width=1251) (actual time=0.043..0.179 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..46.16 rows=13 width=1259) (actual time=0.034..0.035 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..46.16 rows=13 width=1259) (actual time=0.034..0.034 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..46.16 rows=13 width=1251) (actual time=0.031..0.032 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.025..0.028 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.024..0.028 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.10 rows=1 width=1259) (actual time=0.017..0.020 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.10 rows=1 width=1251) (actual time=0.015..0.018 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.015..0.017 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.454 ms
Execution Time: 1.126 ms
```

### requester list (heavy requesters), page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTI5IDAzOjIwOjI1LjYwNTY3NSswMCIsImkiOiJkOWZlM2ZiMi02ZWU5LTdhMTktNjE1Ny0wODUxOWU2M2U3OTcifQ.K4mOY1wZRIt8oswC_7OGPSOWfru2VXbnrFUoM0OQ-Zw` as `req-1`

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
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.029..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.304 ms
Execution Time: 0.081 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.030..0.032 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.022..0.024 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.118 ms
Execution Time: 0.057 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.047..0.048 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.046..0.047 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.037..0.040 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.031..0.034 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.020..0.021 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.232 ms
Execution Time: 0.080 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=191.61..191.62 rows=1 width=8) (actual time=1.411..1.416 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..179.10 rows=1001 width=4) (actual time=0.075..1.248 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..181.59 rows=1015 width=4) (actual time=0.073..1.116 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..171.44 rows=1015 width=0) (actual time=0.072..0.939 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..98.97 rows=1001 width=0) (actual time=0.071..0.818 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..88.96 rows=1001 width=4) (actual time=0.070..0.677 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..123.10 rows=1387 width=4) (actual time=0.070..0.563 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..123.10 rows=1387 width=0) (actual time=0.061..0.397 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..46.29 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..46.16 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..46.16 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..46.16 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.871 ms
Execution Time: 1.528 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],"2026-09-29 03:20:25.605675+00","d9fe3fb2-6ee9-7a19-6157-08519e63e797",51]`

```
Nested Loop Left Join  (cost=22.57..204.17 rows=1 width=1301) (actual time=0.137..0.681 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..195.84 rows=1 width=1266) (actual time=0.114..0.458 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.99..187.52 rows=1 width=1259) (actual time=0.096..0.312 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.99..186.38 rows=51 width=1259) (actual time=0.094..0.265 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.99..225.06 rows=63 width=1259) (actual time=0.092..0.256 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..163.35 rows=51 width=1259) (actual time=0.042..0.194 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..3690.17 rows=1155 width=1259) (actual time=0.042..0.188 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..3690.17 rows=1155 width=1251) (actual time=0.037..0.166 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..39.75 rows=11 width=1259) (actual time=0.026..0.026 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..39.75 rows=11 width=1259) (actual time=0.025..0.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..39.75 rows=11 width=1251) (actual time=0.023..0.023 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.023..0.026 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.022..0.025 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.11 rows=1 width=1259) (actual time=0.016..0.019 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.11 rows=1 width=1251) (actual time=0.014..0.016 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.014..0.015 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.506 ms
Execution Time: 0.925 ms
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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.036..0.038 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.025..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.007..0.007 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.269 ms
Execution Time: 0.085 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.060..0.062 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.022..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.127 ms
Execution Time: 0.090 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.057..0.058 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.056..0.057 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.048..0.050 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.041..0.043 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.012..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.222 ms
Execution Time: 0.093 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df",["3f67ff28-a74f-4926-927f-e48fa8f56f68"]]`

```
Aggregate  (cost=78.80..78.81 rows=1 width=8) (actual time=0.456..0.457 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.29 rows=1001 width=4) (actual time=0.035..0.388 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..13166.04 rows=200081 width=4) (actual time=0.033..0.302 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..13166.04 rows=200081 width=0) (actual time=0.028..0.167 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.161 ms
Execution Time: 0.546 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df",["3f67ff28-a74f-4926-927f-e48fa8f56f68"],51]`

```
Limit  (cost=1.00..113.85 rows=51 width=1293) (actual time=0.081..0.621 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..442721.82 rows=200081 width=1293) (actual time=0.080..0.611 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..441221.22 rows=200081 width=1266) (actual time=0.064..0.556 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..375715.83 rows=200081 width=1258) (actual time=0.047..0.318 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..373206.53 rows=200081 width=1251) (actual time=0.032..0.267 rows=51 loops=1)
                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.009..0.011 rows=1 loops=1)
                                Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                Filter: (id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.484 ms
Execution Time: 0.694 ms
```

### admin list, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjU5OjU5LjU2MzExOCswMCIsImkiOiJkNmRhMTEwZi1jOGM3LTEzZmItZjRmNy1jNWUzYjFkMmMzMjQifQ.5_MyZZuC5PQ_dxCb1ddmj7jKtKzciw_YkK9Wu2ufiLE` as `admin-large`

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.040..0.043 rows=1 loops=1)
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
Planning Time: 0.306 ms
Execution Time: 0.079 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.025..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.114..0.116 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.112..0.114 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.101..0.105 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.091..0.094 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.035..0.036 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.052..0.053 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.339 ms
Execution Time: 0.178 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND true LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df",["3f67ff28-a74f-4926-927f-e48fa8f56f68"]]`

```
Aggregate  (cost=78.80..78.81 rows=1 width=8) (actual time=0.633..0.635 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.29 rows=1001 width=4) (actual time=0.060..0.539 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..13166.04 rows=200081 width=4) (actual time=0.058..0.421 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..13166.04 rows=200081 width=0) (actual time=0.045..0.254 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.340 ms
Execution Time: 0.685 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df","e3e5f6b3-5ced-4136-b389-47c4881eb5df",["3f67ff28-a74f-4926-927f-e48fa8f56f68"],"2026-09-30 23:59:59.563118+00","d6da110f-c8c7-13fb-f4f7-c5e3b1d2c324",51]`

```
Limit  (cost=1.00..113.98 rows=51 width=1293) (actual time=0.128..0.557 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..443183.49 rows=200056 width=1293) (actual time=0.127..0.546 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..441683.07 rows=200056 width=1266) (actual time=0.101..0.484 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..376185.63 rows=200056 width=1258) (actual time=0.070..0.245 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..373676.63 rows=200056 width=1251) (actual time=0.043..0.180 rows=51 loops=1)
                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:59:59.563118+00'::timestamp with time zone, 'd6da110f-c8c7-13fb-f4f7-c5e3b1d2c324'::uuid)))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.019..0.021 rows=1 loops=1)
                                Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                Filter: (id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.565 ms
Execution Time: 0.653 ms
```

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.054..0.057 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.033..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.013..0.013 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.329 ms
Execution Time: 0.098 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.025..0.027 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.126 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.063..0.065 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.062..0.063 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.047..0.055 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.035..0.042 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.025..0.026 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.007..0.012 rows=2 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.289 ms
Execution Time: 0.101 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=3242.56..3242.57 rows=1 width=8) (actual time=3.676..3.684 rows=1 loops=1)
  Buffers: shared hit=1023
  ->  Limit  (cost=0.55..3230.05 rows=1001 width=4) (actual time=0.082..3.523 rows=1001 loops=1)
        Buffers: shared hit=1023
        ->  Result  (cost=0.55..3584.94 rows=1111 width=4) (actual time=0.080..3.299 rows=1001 loops=1)
              Buffers: shared hit=1023
              ->  Append  (cost=0.55..3573.83 rows=1111 width=0) (actual time=0.079..3.090 rows=1001 loops=1)
                    Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..3152.53 rows=1001 width=0) (actual time=0.078..2.913 rows=1001 loops=1)
                          Buffers: shared hit=1023
                          ->  Limit  (cost=0.55..3142.52 rows=1001 width=4) (actual time=0.076..2.737 rows=1001 loops=1)
                                Buffers: shared hit=1023
                                ->  Result  (cost=0.55..18685.99 rows=5953 width=4) (actual time=0.076..2.611 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=1023
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..18685.99 rows=5953 width=0) (actual time=0.061..2.391 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 1
                                            Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..324.88 rows=97 width=0) (never executed)
                          ->  Limit  (cost=0.42..323.91 rows=97 width=4) (never executed)
                                ->  Result  (cost=0.42..323.91 rows=97 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..323.91 rows=97 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..46.38 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..46.26 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..46.26 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..46.26 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.959 ms
Execution Time: 4.102 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1583.75 rows=1 width=1301) (actual time=0.443..2.289 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1575.41 rows=1 width=1266) (actual time=0.415..2.081 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1567.10 rows=1 width=1259) (actual time=0.393..1.910 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1565.95 rows=51 width=1259) (actual time=0.388..1.846 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3473.60 rows=115 width=1259) (actual time=0.386..1.834 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3210.58 rows=51 width=1259) (actual time=0.146..1.568 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..374707.13 rows=5953 width=1259) (actual time=0.146..1.560 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..374707.13 rows=5953 width=1251) (actual time=0.137..1.531 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..170.51 rows=51 width=1259) (actual time=0.028..0.033 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..323.91 rows=97 width=1259) (actual time=0.028..0.032 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..323.91 rows=97 width=1251) (actual time=0.023..0.026 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..46.26 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..46.26 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..46.26 rows=12 width=1251) (actual time=0.024..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.182..0.186 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.181..0.185 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.142..0.171 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.139..0.167 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.067..0.070 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.053..0.061 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.042..0.043 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.026..0.026 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.030..0.030 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.134 ms
Execution Time: 2.624 ms
```

### composed: agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.KSGu2SEzDw2JzDODruoDKlDAAEvwN1-X2WIsJMsQ-j0` as `agent-1`

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
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.037..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.496 ms
Execution Time: 0.117 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.027..0.030 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.021..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.099 ms
Execution Time: 0.050 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=893) (actual time=0.078..0.081 rows=1 loops=1)
  Buffers: shared hit=7
  ->  Sort  (cost=16.62..16.62 rows=1 width=893) (actual time=0.077..0.079 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 28kB
        Buffers: shared hit=7
        ->  Result  (cost=0.56..16.61 rows=1 width=893) (actual time=0.057..0.068 rows=2 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
              Buffers: shared hit=7
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=893) (actual time=0.046..0.056 rows=2 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=7
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.031..0.033 rows=1 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=909) (actual time=0.010..0.016 rows=2 loops=1)
                          Index Cond: (tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                          Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.358 ms
Execution Time: 0.134 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $7::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($8::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])) AND (NOT (r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $5::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $6::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=3242.56..3242.57 rows=1 width=8) (actual time=2.703..2.709 rows=1 loops=1)
  Buffers: shared hit=1023
  ->  Limit  (cost=0.55..3230.05 rows=1001 width=4) (actual time=0.055..2.601 rows=1001 loops=1)
        Buffers: shared hit=1023
        ->  Result  (cost=0.55..3584.94 rows=1111 width=4) (actual time=0.053..2.477 rows=1001 loops=1)
              Buffers: shared hit=1023
              ->  Append  (cost=0.55..3573.83 rows=1111 width=0) (actual time=0.053..2.316 rows=1001 loops=1)
                    Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..3152.53 rows=1001 width=0) (actual time=0.052..2.184 rows=1001 loops=1)
                          Buffers: shared hit=1023
                          ->  Limit  (cost=0.55..3142.52 rows=1001 width=4) (actual time=0.051..2.065 rows=1001 loops=1)
                                Buffers: shared hit=1023
                                ->  Result  (cost=0.55..18685.99 rows=5953 width=4) (actual time=0.051..1.968 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=1023
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..18685.99 rows=5953 width=0) (actual time=0.043..1.823 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Rows Removed by Filter: 1
                                            Buffers: shared hit=1023
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..324.88 rows=97 width=0) (never executed)
                          ->  Limit  (cost=0.42..323.91 rows=97 width=4) (never executed)
                                ->  Result  (cost=0.42..323.91 rows=97 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..323.91 rows=97 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..46.38 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..46.26 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..46.26 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..46.26 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.083 ms
Execution Time: 2.916 ms
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

Arguments: `["2a552b54-08bb-4a4b-945c-a3f8d37b056b","d2ede654-ffb7-41e0-88f9-3e9f40dcbfe9",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.39..1606.92 rows=1 width=1301) (actual time=0.229..2.018 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1598.58 rows=1 width=1266) (actual time=0.209..1.846 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.81..1590.27 rows=1 width=1259) (actual time=0.194..1.714 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.81..1589.12 rows=51 width=1259) (actual time=0.192..1.670 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.81..3525.84 rows=115 width=1259) (actual time=0.191..1.661 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..3265.26 rows=51 width=1259) (actual time=0.047..1.492 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..366430.65 rows=5724 width=1259) (actual time=0.047..1.485 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..366430.65 rows=5724 width=1251) (actual time=0.043..1.466 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..171.14 rows=51 width=1259) (actual time=0.020..0.024 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..311.72 rows=93 width=1259) (actual time=0.019..0.023 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..311.72 rows=93 width=1251) (actual time=0.017..0.020 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..43.18 rows=12 width=1259) (actual time=0.023..0.027 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..43.18 rows=12 width=1259) (actual time=0.023..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..43.18 rows=12 width=1251) (actual time=0.022..0.025 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.50 rows=1 width=1259) (actual time=0.099..0.103 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.50 rows=1 width=1259) (actual time=0.099..0.102 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.069..0.092 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.067..0.089 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.047..0.050 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '2a552b54-08bb-4a4b-945c-a3f8d37b056b'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd2ede654-ffb7-41e0-88f9-3e9f40dcbfe9'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.512 ms
Execution Time: 2.248 ms
```


## Host

```
### 2026-11-19T23:19:27Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 26260877312 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: buildx_buildkit_nava-ci0 multica-backend-1 multica-frontend-1 multica-postgres-1 
```
