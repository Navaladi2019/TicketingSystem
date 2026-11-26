### 2026-11-22T07:47:59Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 26260877312 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: buildx_buildkit_nava-ci0 multica-backend-1 multica-frontend-1 multica-postgres-1 
# P3-36 raw report (TICK-621)

Generated 2026-11-22T07:52:13Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.19 / p95 2.37 / p99 8.34 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_logical_emit_message(true, ...); COMMIT` probe, a commit that flushes WAL with no INSERT, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). Every scored row, list and get rows included (TICK-1040), waits before each attempt until no container runs other than the run's database, the buildkit builder and the three Multica platform containers, and the attempt is void if another one runs during it (docker ps every 5 s). A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| agent list, sort=-updated_at | 1 | large | 1933 | 77.22 | 128.25 | 154.88 | - | - | - | - | - | 150 | meets |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.19 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

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

### agent list, sort=-updated_at, attempt 1

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7', true)` | 1933 | 4.29 | 13.73 | 27.27 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1933 | 4.94 | 13.01 | 23.02 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1933 | 4.48 | 12.22 | 26.05 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1933 | 4.55 | 12.46 | 26.51 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1933 | 7.11 | 16.21 | 33.31 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1933 | 21.56 | 50.16 | 65.69 |
| 7 | `commit` | 1933 | 4.88 | 17.00 | 29.78 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### agent list, sort=-updated_at, attempt 1

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.036..0.038 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.024..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.007..0.007 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.257 ms
Execution Time: 0.069 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["5c4d76e8-6bd9-4920-aee8-4de25eab1ac7","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.028..0.029 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.021..0.022 rows=1 loops=1)
        Index Cond: (tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.094 ms
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

Arguments: `["5c4d76e8-6bd9-4920-aee8-4de25eab1ac7","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.043..0.044 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.042..0.043 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.035..0.038 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.029..0.031 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.019..0.020 rows=1 loops=1)
                          Index Cond: (tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.200 ms
Execution Time: 0.071 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["5c4d76e8-6bd9-4920-aee8-4de25eab1ac7","01a2023b-2426-7c12-a3ab-8386d703442f",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=629.30..629.31 rows=1 width=8) (actual time=1.340..1.347 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..616.79 rows=1001 width=4) (actual time=0.059..1.199 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..701.13 rows=1138 width=4) (actual time=0.057..1.077 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..689.75 rows=1138 width=0) (actual time=0.057..0.924 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.78 rows=1001 width=0) (actual time=0.056..0.815 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.77 rows=1001 width=4) (actual time=0.055..0.685 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..541.39 rows=6207 width=4) (actual time=0.054..0.580 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..541.39 rows=6207 width=0) (actual time=0.045..0.435 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..488.34 rows=124 width=0) (never executed)
                          ->  Limit  (cost=0.42..487.10 rows=124 width=4) (never executed)
                                ->  Result  (cost=0.42..487.10 rows=124 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..487.10 rows=124 width=0) (never executed)
                                            Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..53.46 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..53.34 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..53.34 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.34 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 0.914 ms
Execution Time: 1.511 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["5c4d76e8-6bd9-4920-aee8-4de25eab1ac7","01a2023b-2426-7c12-a3ab-8386d703442f",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1787.51 rows=1 width=1317) (actual time=0.273..1.704 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1779.18 rows=1 width=1282) (actual time=0.254..1.552 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1770.86 rows=1 width=1275) (actual time=0.230..1.425 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1769.72 rows=51 width=1275) (actual time=0.228..1.387 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3933.06 rows=115 width=1275) (actual time=0.227..1.379 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3632.88 rows=51 width=1275) (actual time=0.101..1.236 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..442091.97 rows=6207 width=1275) (actual time=0.101..1.230 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..442091.97 rows=6207 width=1267) (actual time=0.095..1.211 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..200.59 rows=51 width=1275) (actual time=0.017..0.021 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..487.10 rows=124 width=1275) (actual time=0.017..0.020 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..487.10 rows=124 width=1267) (actual time=0.015..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..53.34 rows=12 width=1275) (actual time=0.021..0.022 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..53.34 rows=12 width=1275) (actual time=0.021..0.022 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.34 rows=12 width=1267) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1275) (actual time=0.086..0.089 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1275) (actual time=0.086..0.088 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1275) (actual time=0.059..0.078 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1267) (actual time=0.057..0.076 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.041 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '5c4d76e8-6bd9-4920-aee8-4de25eab1ac7'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a2023b-2426-7c12-a3ab-8386d703442f'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.306 ms
Execution Time: 1.907 ms
```


## docker ps every 10 s (nava-it-/nava-ci-/nava-aclperf- only)

```
07:48:01 
07:48:11 
07:48:22 
07:48:33 
07:48:44 
07:48:54 
07:49:04 
07:49:15 
07:49:25 
07:49:35 
07:49:46 
07:49:56 
07:50:07 
07:50:17 
07:50:27 
07:50:38 
07:50:48 
07:50:59 
07:51:09 
07:51:20 
07:51:30 
07:51:40 
07:51:51 
07:52:01 
07:52:12 
```
