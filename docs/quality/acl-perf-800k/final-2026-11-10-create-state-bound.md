### 2026-11-10T01:32:33Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 12545507328 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: buildx_buildkit_nava-ci0 multica-backend-1 multica-frontend-1 multica-postgres-1 

# P3-36 raw report (TICK-621)

Generated 2026-11-10T01:38:57Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 805592, open 125870, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 403624
record_shares 15815
heap 1049 MB, indexes 655 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 2.60 / p95 4.03 / p99 4.98 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| agent list, filter[state]=new | large | 590 | 256.83 | 390.25 | 440.26 | 150 | **MISS** |
| agent list, filter[site]=s7, short data-filter timeout (bound check) | large | 4396 | 29.87 | 69.02 | 109.56 | 0 | n/a (bound check) |
| create, agent (1 client) | large | 354 | 51.33 | 83.71 | 141.62 | 50 | **MISS** |
| create, agent | large | 1679 | 78.41 | 202.76 | 306.25 | 50 | **MISS** |
| create, admin | large | 1487 | 92.08 | 223.03 | 336.99 | 50 | **MISS** |
| create, small tenant agent | small | 1917 | 71.56 | 157.44 | 247.65 | 50 | **MISS** |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (2.60 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 354 | 6.21 / 10.10 | 10.20 / 19.40 | 1 | 2.60 / 2.60 | 6.88 / 16.23 | 0.59 / 1.46 |
| create, agent | 1679 | 31.17 / 149.57 | 8.45 / 15.19 | 1 | 2.60 / 2.60 | 5.25 / 11.52 | 0.58 / 1.80 |
| create, admin | 1487 | 33.53 / 165.99 | 9.31 / 18.73 | 1 | 2.60 / 2.60 | 6.18 / 14.67 | 0.57 / 2.06 |
| create, small tenant agent | 1917 | 8.28 / 18.62 | 11.21 / 23.99 | 1 | 2.60 / 2.60 | 7.95 / 19.22 | 0.55 / 2.64 |

## Data-filter statement timeout (TICK-647)

Default timeout 2s (store.DataFilterTimeout); the bound check runs with 2ms. A 422 is `filter_too_expensive`. Latency includes the 422s.

| Scenario | Requests | 422 | p50 | p95 | max |
| --- | --- | --- | --- | --- | --- |
| agent list, filter[site]=s7, short data-filter timeout (bound check) | 4396 | 4396 | 29.87 | 69.02 | 201.93 |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### agent list, filter[state]=new

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 590 | 5.02 | 15.24 | 34.98 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 590 | 6.30 | 14.10 | 23.22 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 590 | 5.22 | 14.86 | 25.67 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 590 | 5.77 | 14.27 | 30.70 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 590 | 88.87 | 136.83 | 180.14 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 590 | 118.28 | 190.67 | 252.71 |
| 7 | `commit` | 590 | 5.96 | 23.23 | 46.54 |

### agent list, filter[site]=s7, short data-filter timeout (bound check)

Round trips: 8, statements: 8, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 4396 | 2.79 | 8.22 | 14.48 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 4396 | 3.59 | 9.00 | 15.91 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 4396 | 3.03 | 8.68 | 18.52 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 4396 | 3.41 | 8.44 | 14.13 |
| 5 | `SELECT current_setting('statement_timeout')` | 4396 | 2.76 | 7.74 | 13.38 |
| 6 | `SET LOCAL statement_timeout = '2'` | 4396 | 2.54 | 7.72 | 13.61 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 4396 | 5.03 | 9.68 | 15.64 |
| 8 | `rollback` | 4396 | 2.48 | 7.62 | 14.20 |

### create, agent (1 client)

Round trips: 8, statements: 15, statements per batch: [4 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 708 | 3.14 | 6.32 | 12.20 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 354 | 3.25 | 6.77 | 12.53 |
| 3 | `commit` | 354 | 6.96 | 15.12 | 22.50 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 708 | 3.14 | 6.32 | 12.20 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 354 | 7.26 | 12.28 | 21.32 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 354 | 4.09 | 7.51 | 12.75 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 354 | 6.21 | 10.10 | 17.04 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 354 | 9.48 | 18.83 | 34.06 |

### create, agent

Round trips: 8, statements: 15, statements per batch: [4 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 3358 | 2.95 | 7.81 | 14.43 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 1679 | 3.27 | 7.89 | 14.69 |
| 3 | `commit` | 1679 | 6.25 | 11.45 | 18.94 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 3358 | 2.95 | 7.81 | 14.43 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 1679 | 6.91 | 16.13 | 25.58 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1679 | 3.63 | 8.02 | 11.96 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 1679 | 31.17 | 149.57 | 251.87 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1679 | 7.85 | 14.12 | 23.38 |

### create, admin

Round trips: 8, statements: 15, statements per batch: [4 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 2974 | 3.48 | 10.75 | 18.66 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 1487 | 3.86 | 11.42 | 19.13 |
| 3 | `commit` | 1487 | 7.10 | 14.08 | 24.24 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)` | 2974 | 3.48 | 10.75 | 18.66 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 1487 | 7.42 | 20.06 | 29.60 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1487 | 4.14 | 10.40 | 17.36 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 1487 | 33.53 | 165.99 | 278.85 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1487 | 8.78 | 17.27 | 31.78 |

### create, small tenant agent

Round trips: 8, statements: 15, statements per batch: [4 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '79dd83ff-d024-4b4f-8643-79590f7de576', true)` | 10 | 4.74 | 15.84 | 15.84 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 1917 | 5.02 | 13.65 | 21.36 |
| 3 | `commit` | 1917 | 8.40 | 17.95 | 48.43 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '79dd83ff-d024-4b4f-8643-79590f7de576', true)` | 10 | 4.74 | 15.84 | 15.84 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 1917 | 9.60 | 29.57 | 56.41 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1917 | 5.60 | 15.91 | 27.00 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 1917 | 8.28 | 18.62 | 33.29 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1917 | 10.55 | 21.82 | 57.99 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### agent list, filter[state]=new

Request: `GET /api/v1/records/ticket?filter%5Bstate%5D=new` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.083..0.089 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.058..0.061 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.015..0.015 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.575 ms
Execution Time: 0.165 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e92d0472-fb8a-4750-a52d-ed16b3564885","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.045..0.048 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.033..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.177 ms
Execution Time: 0.096 ms
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

Arguments: `["e92d0472-fb8a-4750-a52d-ed16b3564885","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.080..0.082 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.078..0.079 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.065..0.070 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.054..0.058 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.034..0.035 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.015..0.017 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.343 ms
Execution Time: 0.135 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e92d0472-fb8a-4750-a52d-ed16b3564885","003e583b-d7e7-46b2-901d-2adcf9eabb35",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=20822.35..20822.36 rows=1 width=8) (actual time=48.515..48.533 rows=1 loops=1)
  Buffers: shared hit=18568
  ->  Limit  (cost=459.57..20818.39 rows=317 width=4) (actual time=14.124..48.271 rows=1001 loops=1)
        Buffers: shared hit=18568
        ->  Result  (cost=459.57..20818.39 rows=317 width=4) (actual time=14.121..47.992 rows=1001 loops=1)
              Buffers: shared hit=18568
              ->  Append  (cost=459.57..20815.22 rows=317 width=0) (actual time=14.119..47.708 rows=1001 loops=1)
                    Buffers: shared hit=18568
                    ->  Subquery Scan on "*SELECT* 1"  (cost=459.57..20233.55 rows=308 width=0) (actual time=14.117..47.507 rows=1001 loops=1)
                          Buffers: shared hit=18568
                          ->  Limit  (cost=459.57..20230.47 rows=308 width=4) (actual time=14.115..47.177 rows=1001 loops=1)
                                Buffers: shared hit=18568
                                ->  Result  (cost=459.57..20230.47 rows=308 width=4) (actual time=14.114..47.008 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      Buffers: shared hit=18568
                                      ->  Bitmap Heap Scan on records r  (cost=459.57..20230.47 rows=308 width=0) (actual time=14.101..46.715 rows=1001 loops=1)
                                            Recheck Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (state = ANY ('{new}'::text[]))
                                            Rows Removed by Filter: 20147
                                            Heap Blocks: exact=18193
                                            Buffers: shared hit=18568
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..459.48 rows=5953 width=0) (actual time=7.887..7.888 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..484.55 rows=7 width=0) (never executed)
                          ->  Limit  (cost=0.42..484.48 rows=7 width=4) (never executed)
                                ->  Result  (cost=0.42..484.48 rows=7 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.48 rows=7 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.05 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.04 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..51.04 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.04 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.704 ms
Execution Time: 48.929 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["e92d0472-fb8a-4750-a52d-ed16b3564885","003e583b-d7e7-46b2-901d-2adcf9eabb35",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=20286.70..20759.26 rows=1 width=1301) (actual time=49.781..50.321 rows=51 loops=1)
  Buffers: shared hit=21516
  ->  Nested Loop  (cost=20286.41..20750.93 rows=1 width=1266) (actual time=49.760..50.126 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21363
        ->  Subquery Scan on x  (cost=20286.12..20742.61 rows=1 width=1259) (actual time=49.746..49.982 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21210
              ->  Limit  (cost=20286.12..20741.47 rows=51 width=1259) (actual time=49.743..49.933 rows=51 loops=1)
                    Buffers: shared hit=21210
                    ->  Merge Append  (cost=20286.12..20821.82 rows=60 width=1259) (actual time=49.741..49.923 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21210
                          ->  Limit  (cost=20240.75..20240.88 rows=51 width=1259) (actual time=49.480..49.503 rows=50 loops=1)
                                Buffers: shared hit=21119
                                ->  Sort  (cost=20240.75..20241.52 rows=308 width=1259) (actual time=49.478..49.495 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=21119
                                      ->  Result  (cost=459.57..20230.47 rows=308 width=1259) (actual time=13.244..48.162 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                            Buffers: shared hit=21119
                                            ->  Bitmap Heap Scan on records r  (cost=459.57..20230.47 rows=308 width=1251) (actual time=13.237..47.669 rows=1168 loops=1)
                                                  Recheck Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (state = ANY ('{new}'::text[]))
                                                  Rows Removed by Filter: 22950
                                                  Heap Blocks: exact=20744
                                                  Buffers: shared hit=21119
                                                  ->  Bitmap Index Scan on records_list_group  (cost=0.00..459.48 rows=5953 width=0) (actual time=7.325..7.326 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=375
                          ->  Limit  (cost=0.42..484.48 rows=7 width=1259) (actual time=0.069..0.197 rows=2 loops=1)
                                Buffers: shared hit=46
                                ->  Result  (cost=0.42..484.48 rows=7 width=1259) (actual time=0.066..0.193 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      Buffers: shared hit=46
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.48 rows=7 width=1251) (actual time=0.050..0.176 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=46
                          ->  Limit  (cost=0.42..51.04 rows=1 width=1259) (actual time=0.073..0.074 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..51.04 rows=1 width=1259) (actual time=0.072..0.073 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.04 rows=1 width=1251) (actual time=0.068..0.068 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.113..0.116 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.112..0.115 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.099..0.103 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.096..0.099 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.057 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.956 ms
Execution Time: 50.618 ms
```

### agent list, filter[site]=s7, short data-filter timeout (bound check)

Request: `GET /api/v1/records/ticket?filter%5Bsite%5D=s7` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.045..0.047 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.028..0.029 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.348 ms
Execution Time: 0.090 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["e92d0472-fb8a-4750-a52d-ed16b3564885","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.047..0.050 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
        Index Cond: (tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.187 ms
Execution Time: 0.089 ms
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

Arguments: `["e92d0472-fb8a-4750-a52d-ed16b3564885","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.071..0.073 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.070..0.071 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.059..0.063 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.051..0.054 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.037..0.038 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.010..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.296 ms
Execution Time: 0.113 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.029 ms
Execution Time: 0.030 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["e92d0472-fb8a-4750-a52d-ed16b3564885","003e583b-d7e7-46b2-901d-2adcf9eabb35",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}"]`

```
Aggregate  (cost=20820.39..20820.40 rows=1 width=8) (actual time=81.854..81.864 rows=1 loops=1)
  Buffers: shared hit=21734
  ->  Limit  (cost=459.51..20819.60 rows=63 width=4) (actual time=15.907..81.757 rows=121 loops=1)
        Buffers: shared hit=21734
        ->  Result  (cost=459.51..20819.60 rows=63 width=4) (actual time=15.904..81.696 rows=121 loops=1)
              Buffers: shared hit=21734
              ->  Append  (cost=459.51..20818.97 rows=63 width=0) (actual time=15.903..81.633 rows=121 loops=1)
                    Buffers: shared hit=21734
                    ->  Subquery Scan on "*SELECT* 1"  (cost=459.51..20238.45 rows=60 width=0) (actual time=15.901..79.548 rows=120 loops=1)
                          Buffers: shared hit=21119
                          ->  Limit  (cost=459.51..20237.85 rows=60 width=4) (actual time=15.899..79.481 rows=120 loops=1)
                                Buffers: shared hit=21119
                                ->  Result  (cost=459.51..20237.85 rows=60 width=4) (actual time=15.897..79.433 rows=120 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      Buffers: shared hit=21119
                                      ->  Bitmap Heap Scan on records r  (cost=459.51..20237.85 rows=60 width=0) (actual time=15.881..79.333 rows=120 loops=1)
                                            Recheck Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"site": "s7"}'::jsonb)
                                            Rows Removed by Filter: 23998
                                            Heap Blocks: exact=20744
                                            Buffers: shared hit=21119
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..459.48 rows=5953 width=0) (actual time=7.976..7.976 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..484.65 rows=1 width=0) (actual time=0.746..1.653 rows=1 loops=1)
                          Buffers: shared hit=523
                          ->  Limit  (cost=0.42..484.64 rows=1 width=4) (actual time=0.745..1.651 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..484.64 rows=1 width=4) (actual time=0.742..1.647 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.64 rows=1 width=0) (actual time=0.724..1.627 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.06 rows=1 width=0) (actual time=0.219..0.220 rows=0 loops=1)
                          Buffers: shared hit=67
                          ->  Limit  (cost=0.42..51.05 rows=1 width=4) (actual time=0.218..0.219 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..51.05 rows=1 width=4) (actual time=0.216..0.216 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.05 rows=1 width=0) (actual time=0.205..0.205 rows=0 loops=1)
                                            Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (actual time=0.148..0.151 rows=0 loops=1)
                          Buffers: shared hit=25
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (actual time=0.147..0.150 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (actual time=0.146..0.148 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid)
                                      Buffers: shared hit=25
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (actual time=0.142..0.144 rows=0 loops=1)
                                            Buffers: shared hit=25
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.060..0.063 rows=3 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=13
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.043..0.050 rows=3 loops=1)
                                                        Recheck Cond: (((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Heap Blocks: exact=3
                                                        Buffers: shared hit=13
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.032 rows=0 loops=1)
                                                              Buffers: shared hit=10
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                    Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=8
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.025..0.025 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'e92d0472-fb8a-4750-a52d-ed16b3564885'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '003e583b-d7e7-46b2-901d-2adcf9eabb35'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.313 ms
Execution Time: 82.146 ms
```

### create, agent (1 client)

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)
```

Statement 2 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 3 (write, not EXPLAINed):

```sql
commit
```

Statement 4 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 ; INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4), $6, $7, $8, $9, $10, $11, $11, $12, $13, $14) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)
```

Statement 2 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 3 (write, not EXPLAINed):

```sql
commit
```

Statement 4 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 ; INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4), $6, $7, $8, $9, $10, $11, $11, $12, $13, $14) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, admin

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)
```

Statement 2 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 3 (write, not EXPLAINed):

```sql
commit
```

Statement 4 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'e92d0472-fb8a-4750-a52d-ed16b3564885', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 ; INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4), $6, $7, $8, $9, $10, $11, $11, $12, $13, $14) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, small tenant agent

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '79dd83ff-d024-4b4f-8643-79590f7de576', true)
```

Statement 2 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 3 (write, not EXPLAINed):

```sql
commit
```

Statement 4 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '79dd83ff-d024-4b4f-8643-79590f7de576', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 ; INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4), $6, $7, $8, $9, $10, $11, $11, $12, $13, $14) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

