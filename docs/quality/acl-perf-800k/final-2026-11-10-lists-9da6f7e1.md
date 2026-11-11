# P3-36 raw report (TICK-621)

Generated 2026-11-10T13:14:48Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.05 / p95 1.55 / p99 1.77 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| agent list by group, default sort, page 1 | large | 3381 | 43.01 | 73.50 | 90.44 | 150 | meets |
| agent list by group, default sort, page 20 | large | 3414 | 42.32 | 71.16 | 88.71 | 150 | meets |
| agent list, filter[state]=new | large | 3121 | 47.06 | 76.23 | 91.57 | 150 | meets |
| admin list, filter[state][in]=new,in_progress, sort=-updated_at | large | 4084 | 34.26 | 68.21 | 90.40 | 150 | meets |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.05 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |

## Data-filter statement timeout (TICK-647)

Default timeout 2s (store.DataFilterTimeout); the bound check runs with 2ms. A 422 is `filter_too_expensive`. Latency includes the 422s.

| Scenario | Requests | 422 | p50 | p95 | max |
| --- | --- | --- | --- | --- | --- |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### agent list by group, default sort, page 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '20e3e9e3-b6e5-4ba4-b920-77702309b27a', true)` | 3381 | 2.44 | 6.23 | 11.42 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3381 | 3.22 | 7.11 | 11.87 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3381 | 2.71 | 6.32 | 10.08 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3381 | 2.94 | 6.49 | 10.43 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3381 | 5.29 | 9.32 | 14.69 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3381 | 13.57 | 27.41 | 39.24 |
| 7 | `commit` | 3381 | 2.66 | 6.32 | 11.79 |

### agent list by group, default sort, page 20

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '20e3e9e3-b6e5-4ba4-b920-77702309b27a', true)` | 3414 | 2.38 | 6.06 | 11.90 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3414 | 3.09 | 6.75 | 11.57 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3414 | 2.54 | 6.03 | 10.79 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3414 | 2.85 | 6.10 | 11.34 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3414 | 4.95 | 8.58 | 13.96 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3414 | 13.20 | 28.04 | 40.78 |
| 7 | `commit` | 3414 | 2.58 | 6.31 | 10.71 |

### agent list, filter[state]=new

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '20e3e9e3-b6e5-4ba4-b920-77702309b27a', true)` | 3121 | 2.56 | 6.33 | 10.69 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 3121 | 3.25 | 7.59 | 11.50 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3121 | 2.76 | 6.59 | 11.21 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3121 | 2.95 | 6.70 | 11.30 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3121 | 5.40 | 9.05 | 12.97 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3121 | 16.86 | 31.98 | 44.33 |
| 7 | `commit` | 3121 | 2.72 | 6.97 | 11.68 |

### admin list, filter[state][in]=new,in_progress, sort=-updated_at

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '20e3e9e3-b6e5-4ba4-b920-77702309b27a', true)` | 4084 | 2.29 | 6.59 | 12.67 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 4084 | 2.95 | 7.01 | 12.48 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 4084 | 2.54 | 6.80 | 13.11 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 4084 | 2.73 | 6.61 | 11.21 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 4084 | 3.21 | 7.61 | 12.21 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 4084 | 8.57 | 24.08 | 36.49 |
| 7 | `commit` | 4084 | 2.29 | 6.79 | 15.81 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### agent list by group, default sort, page 1

Request: `GET /api/v1/records/ticket` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=606) (actual time=0.047..0.050 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.032..0.033 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.388 ms
Execution Time: 0.096 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.026..0.028 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.020..0.021 rows=1 loops=1)
        Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.119 ms
Execution Time: 0.049 ms
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

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.047..0.048 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.046..0.047 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.038..0.041 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.032..0.034 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.022..0.022 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.232 ms
Execution Time: 0.079 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=614.84..614.85 rows=1 width=8) (actual time=1.118..1.123 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..602.33 rows=1001 width=4) (actual time=0.086..1.019 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..702.73 rows=1168 width=4) (actual time=0.053..0.887 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..691.05 rows=1168 width=0) (actual time=0.053..0.749 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..99.49 rows=1001 width=0) (actual time=0.052..0.653 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..89.48 rows=1001 width=4) (actual time=0.051..0.542 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..537.21 rows=6041 width=4) (actual time=0.050..0.453 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..537.21 rows=6041 width=0) (actual time=0.042..0.316 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..496.30 rows=154 width=0) (never executed)
                          ->  Limit  (cost=0.42..494.76 rows=154 width=4) (never executed)
                                ->  Result  (cost=0.42..494.76 rows=154 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..494.76 rows=154 width=0) (never executed)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..44.94 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..44.82 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..44.82 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..44.82 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.161 ms
Execution Time: 1.291 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1505.54 rows=1 width=1301) (actual time=0.303..1.929 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1497.20 rows=1 width=1266) (actual time=0.285..1.762 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1488.89 rows=1 width=1259) (actual time=0.272..1.631 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1487.74 rows=51 width=1259) (actual time=0.270..1.585 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3297.24 rows=115 width=1259) (actual time=0.268..1.576 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3042.03 rows=51 width=1259) (actual time=0.126..1.412 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..360282.06 rows=6041 width=1259) (actual time=0.126..1.405 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..360282.06 rows=6041 width=1251) (actual time=0.119..1.383 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..164.13 rows=51 width=1259) (actual time=0.020..0.026 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..494.76 rows=154 width=1259) (actual time=0.020..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..494.76 rows=154 width=1251) (actual time=0.018..0.021 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..44.82 rows=12 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..44.82 rows=12 width=1259) (actual time=0.019..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..44.82 rows=12 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.100..0.105 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.100..0.103 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.069..0.091 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.067..0.089 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.037..0.044 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.031..0.031 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.533 ms
Execution Time: 2.226 ms
```

### agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.UplotAQ6ILeOiMy835o2uzYX1C5StdFXL0moeUOYlOE` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=606) (actual time=0.040..0.042 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.026..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.278 ms
Execution Time: 0.075 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.026..0.028 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.020..0.022 rows=1 loops=1)
        Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.101 ms
Execution Time: 0.047 ms
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

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.043..0.044 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.042..0.042 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.033..0.037 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.027..0.030 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.018..0.019 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.208 ms
Execution Time: 0.071 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=614.84..614.85 rows=1 width=8) (actual time=1.830..1.835 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..602.33 rows=1001 width=4) (actual time=0.065..1.672 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..702.73 rows=1168 width=4) (actual time=0.064..1.495 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..691.05 rows=1168 width=0) (actual time=0.063..1.258 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..99.49 rows=1001 width=0) (actual time=0.061..1.095 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..89.48 rows=1001 width=4) (actual time=0.060..0.905 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..537.21 rows=6041 width=4) (actual time=0.060..0.751 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..537.21 rows=6041 width=0) (actual time=0.050..0.489 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..496.30 rows=154 width=0) (never executed)
                          ->  Limit  (cost=0.42..494.76 rows=154 width=4) (never executed)
                                ->  Result  (cost=0.42..494.76 rows=154 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..494.76 rows=154 width=0) (never executed)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..44.94 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..44.82 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..44.82 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..44.82 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.600 ms
Execution Time: 2.030 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) AND ((r.updated_at, r.id) < ($8::timestamptz, $9::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.38..1527.94 rows=1 width=1301) (actual time=0.255..2.018 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1519.60 rows=1 width=1266) (actual time=0.234..1.828 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.80..1511.29 rows=1 width=1259) (actual time=0.214..1.678 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.80..1510.14 rows=51 width=1259) (actual time=0.212..1.630 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.80..3347.75 rows=115 width=1259) (actual time=0.211..1.620 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..3094.98 rows=51 width=1259) (actual time=0.049..1.428 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..352112.43 rows=5803 width=1259) (actual time=0.049..1.421 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..352112.43 rows=5803 width=1251) (actual time=0.044..1.400 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..164.64 rows=51 width=1259) (actual time=0.022..0.027 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..476.98 rows=148 width=1259) (actual time=0.021..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..476.98 rows=148 width=1251) (actual time=0.019..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..41.86 rows=12 width=1259) (actual time=0.020..0.026 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..41.86 rows=12 width=1259) (actual time=0.020..0.024 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..41.86 rows=12 width=1251) (actual time=0.018..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.117..0.122 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.117..0.121 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.072..0.107 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.070..0.104 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.049..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.645 ms
Execution Time: 2.272 ms
```

### agent list, filter[state]=new

Request: `GET /api/v1/records/ticket?filter%5Bstate%5D=new` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=606) (actual time=0.032..0.034 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.020..0.022 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.006..0.006 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.290 ms
Execution Time: 0.068 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.020..0.022 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.015..0.017 rows=1 loops=1)
        Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.098 ms
Execution Time: 0.041 ms
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

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.039..0.040 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.038..0.039 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.031..0.033 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.025..0.028 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.016..0.017 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.006..0.007 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.203 ms
Execution Time: 0.067 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=638.56..638.57 rows=1 width=8) (actual time=1.323..1.327 rows=1 loops=1)
  Buffers: shared hit=539
  ->  Limit  (cost=0.55..634.65 rows=313 width=4) (actual time=0.044..1.213 rows=1001 loops=1)
        Buffers: shared hit=539
        ->  Result  (cost=0.55..634.65 rows=313 width=4) (actual time=0.043..1.093 rows=1001 loops=1)
              Buffers: shared hit=539
              ->  Append  (cost=0.55..631.52 rows=313 width=0) (actual time=0.042..0.928 rows=1001 loops=1)
                    Buffers: shared hit=539
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..45.58 rows=303 width=0) (actual time=0.041..0.816 rows=1001 loops=1)
                          Buffers: shared hit=539
                          ->  Limit  (cost=0.55..42.55 rows=303 width=4) (actual time=0.041..0.651 rows=1001 loops=1)
                                Buffers: shared hit=539
                                ->  Result  (cost=0.55..42.55 rows=303 width=4) (actual time=0.040..0.545 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=539
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..42.55 rows=303 width=0) (actual time=0.035..0.390 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=539
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..495.04 rows=8 width=0) (never executed)
                          ->  Limit  (cost=0.42..494.96 rows=8 width=4) (never executed)
                                ->  Result  (cost=0.42..494.96 rows=8 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..494.96 rows=8 width=0) (never executed)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..44.85 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..44.84 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..44.84 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..44.84 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.947 ms
Execution Time: 1.421 ms
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

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=1007.86..1476.55 rows=1 width=1301) (actual time=3.657..4.073 rows=51 loops=1)
  Buffers: shared hit=1601
  ->  Nested Loop  (cost=1007.56..1468.21 rows=1 width=1266) (actual time=3.637..3.901 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1448
        ->  Subquery Scan on x  (cost=1007.28..1459.90 rows=1 width=1259) (actual time=3.616..3.765 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1295
              ->  Limit  (cost=1007.28..1458.75 rows=51 width=1259) (actual time=3.614..3.723 rows=51 loops=1)
                    Buffers: shared hit=1295
                    ->  Merge Append  (cost=1007.28..1547.27 rows=61 width=1259) (actual time=3.613..3.716 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1295
                          ->  Limit  (cost=961.91..962.03 rows=51 width=1259) (actual time=3.378..3.394 rows=50 loops=1)
                                Buffers: shared hit=1204
                                ->  Sort  (cost=961.91..962.66 rows=303 width=1259) (actual time=3.377..3.387 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=1204
                                      ->  Result  (cost=0.55..951.80 rows=303 width=1259) (actual time=0.041..2.453 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                            Buffers: shared hit=1204
                                            ->  Index Scan using records_list_group_state on records r  (cost=0.55..951.80 rows=303 width=1251) (actual time=0.038..2.091 rows=1168 loops=1)
                                                  Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                  Buffers: shared hit=1204
                          ->  Limit  (cost=0.42..494.96 rows=8 width=1259) (actual time=0.041..0.106 rows=2 loops=1)
                                Buffers: shared hit=46
                                ->  Result  (cost=0.42..494.96 rows=8 width=1259) (actual time=0.040..0.104 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=46
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..494.96 rows=8 width=1251) (actual time=0.034..0.096 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=46
                          ->  Limit  (cost=0.42..44.84 rows=1 width=1259) (actual time=0.079..0.080 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..44.84 rows=1 width=1259) (actual time=0.079..0.079 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..44.84 rows=1 width=1251) (actual time=0.075..0.075 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.110..0.112 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.109..0.111 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.100..0.102 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.096..0.098 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.057 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.041..0.048 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.018..0.018 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.501 ms
Execution Time: 4.237 ms
```

### admin list, filter[state][in]=new,in_progress, sort=-updated_at

Request: `GET /api/v1/records/ticket?filter%5Bstate%5D%5Bin%5D=new,in_progress&sort=-updated_at` as `admin-large`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=606) (actual time=0.034..0.037 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.022..0.024 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.007..0.007 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.301 ms
Execution Time: 0.070 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.025..0.027 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.020..0.021 rows=1 loops=1)
        Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.107 ms
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

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.047..0.048 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.045..0.046 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.036..0.039 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.031..0.034 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.020..0.021 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.232 ms
Execution Time: 0.079 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND r.state = ANY($7::text[]) LIMIT 1001) c
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d","a5ff1f6f-5bb5-4088-912b-1204545d798f","a5ff1f6f-5bb5-4088-912b-1204545d798f","a5ff1f6f-5bb5-4088-912b-1204545d798f",["b7e28ccf-410b-4d9f-8634-b9a938fd45d1"],["new","in_progress"]]`

```
Aggregate  (cost=90.98..90.99 rows=1 width=8) (actual time=0.658..0.659 rows=1 loops=1)
  Buffers: shared hit=532
  ->  Limit  (cost=0.42..78.47 rows=1001 width=4) (actual time=0.041..0.577 rows=1001 loops=1)
        Buffers: shared hit=532
        ->  Result  (cost=0.42..1547.24 rows=19839 width=4) (actual time=0.039..0.475 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
              Buffers: shared hit=532
              ->  Index Only Scan using records_list_state on records r  (cost=0.42..1547.24 rows=19839 width=0) (actual time=0.034..0.328 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid) AND (state = ANY ('{new,in_progress}'::text[])))
                    Heap Fetches: 0
                    Buffers: shared hit=532
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.181 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND r.state = ANY($7::text[]) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8
```

Arguments: `["20e3e9e3-b6e5-4ba4-b920-77702309b27a","047dbf23-14f6-48f9-9149-3c8bd432d82d","a5ff1f6f-5bb5-4088-912b-1204545d798f","a5ff1f6f-5bb5-4088-912b-1204545d798f","a5ff1f6f-5bb5-4088-912b-1204545d798f",["b7e28ccf-410b-4d9f-8634-b9a938fd45d1"],["new","in_progress"],51]`

```
Limit  (cost=1.00..948.24 rows=51 width=1293) (actual time=0.078..1.210 rows=51 loops=1)
  Buffers: shared hit=635
  ->  Result  (cost=1.00..368480.10 rows=19839 width=1293) (actual time=0.077..1.199 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
        Buffers: shared hit=635
        ->  Nested Loop Left Join  (cost=1.00..368331.30 rows=19839 width=1266) (actual time=0.068..1.153 rows=51 loops=1)
              Buffers: shared hit=635
              ->  Nested Loop  (cost=0.70..360040.54 rows=19839 width=1258) (actual time=0.054..0.936 rows=51 loops=1)
                    Buffers: shared hit=482
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..359784.25 rows=19839 width=1251) (actual time=0.041..0.886 rows=51 loops=1)
                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (entity_definition_id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid))
                          Filter: (state = ANY ('{new,in_progress}'::text[]))
                          Rows Removed by Filter: 419
                          Buffers: shared hit=479
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.009..0.010 rows=1 loops=1)
                                Index Cond: (tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                                Filter: (id = '047dbf23-14f6-48f9-9149-3c8bd432d82d'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.41 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.41 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = '20e3e9e3-b6e5-4ba4-b920-77702309b27a'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.446 ms
Execution Time: 1.271 ms
```

