# P3-36 raw report (TICK-621)

Generated 2026-11-21T05:53:37Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.13 / p95 1.74 / p99 2.27 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_logical_emit_message(true, ...); COMMIT` probe, a commit that flushes WAL with no INSERT, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). An attempt is also void if a container other than the run's database, the buildkit builder and the three Multica platform containers runs during it. A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| agent list, sort=-updated_at | - | large | 2224 | 63.72 | 119.99 | 163.63 | - | - | - | - | - | 150 | meets |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.13 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

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

### agent list, sort=-updated_at

Round trips: 7, statements: 7, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '5733b772-5b5e-4f4d-afeb-ca9d7115ed47', true)` | 2224 | 3.57 | 10.92 | 21.57 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 2224 | 4.37 | 12.10 | 19.64 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2224 | 3.75 | 10.86 | 21.59 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2224 | 3.91 | 10.40 | 20.33 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2224 | 6.68 | 15.26 | 27.94 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2224 | 19.37 | 46.50 | 71.42 |
| 7 | `commit` | 2224 | 3.88 | 13.69 | 27.39 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

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
Nested Loop  (cost=0.56..16.63 rows=1 width=658) (actual time=0.066..0.070 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=59) (actual time=0.044..0.046 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1488) (actual time=0.014..0.014 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.449 ms
Execution Time: 0.134 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["5733b772-5b5e-4f4d-afeb-ca9d7115ed47","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.042..0.045 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.147 ms
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

Arguments: `["5733b772-5b5e-4f4d-afeb-ca9d7115ed47","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.053..0.055 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.052..0.054 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.044..0.047 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.035..0.038 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.024..0.025 rows=1 loops=1)
                          Index Cond: (tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.297 ms
Execution Time: 0.092 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["5733b772-5b5e-4f4d-afeb-ca9d7115ed47","01a1fca7-50ca-7b80-9093-b923ad47b7ff",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=577.90..577.91 rows=1 width=8) (actual time=1.977..1.987 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..565.38 rows=1001 width=4) (actual time=0.081..1.817 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..632.53 rows=1120 width=4) (actual time=0.079..1.638 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..621.33 rows=1120 width=0) (actual time=0.078..1.413 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..98.97 rows=1001 width=0) (actual time=0.077..1.223 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..88.96 rows=1001 width=4) (actual time=0.075..0.989 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..538.49 rows=6091 width=4) (actual time=0.074..0.836 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..538.49 rows=6091 width=0) (actual time=0.058..0.618 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..418.27 rows=105 width=0) (never executed)
                          ->  Limit  (cost=0.42..417.22 rows=105 width=4) (never executed)
                                ->  Result  (cost=0.42..417.22 rows=105 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..417.22 rows=105 width=0) (never executed)
                                            Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..54.01 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..53.88 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..53.88 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.88 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.542 ms
Execution Time: 2.317 ms
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

Arguments: `["5733b772-5b5e-4f4d-afeb-ca9d7115ed47","01a1fca7-50ca-7b80-9093-b923ad47b7ff",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1828.39 rows=1 width=1317) (actual time=0.531..4.555 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1820.05 rows=1 width=1282) (actual time=0.495..4.038 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1811.74 rows=1 width=1275) (actual time=0.469..3.582 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1810.59 rows=51 width=1275) (actual time=0.463..3.436 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..4059.84 rows=116 width=1275) (actual time=0.460..3.415 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3756.82 rows=51 width=1275) (actual time=0.182..3.068 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..448632.30 rows=6091 width=1275) (actual time=0.182..3.005 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..448632.30 rows=6091 width=1267) (actual time=0.166..2.949 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..202.87 rows=51 width=1275) (actual time=0.040..0.054 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..417.22 rows=105 width=1275) (actual time=0.040..0.051 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..417.22 rows=105 width=1267) (actual time=0.034..0.042 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..53.88 rows=13 width=1275) (actual time=0.041..0.044 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..53.88 rows=13 width=1275) (actual time=0.040..0.042 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.88 rows=13 width=1267) (actual time=0.035..0.035 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1275) (actual time=0.191..0.210 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1275) (actual time=0.190..0.206 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1275) (actual time=0.133..0.183 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1267) (actual time=0.129..0.176 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.087..0.098 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.066..0.081 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.054..0.059 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.026..0.027 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.027..0.027 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.022..0.023 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '5733b772-5b5e-4f4d-afeb-ca9d7115ed47'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fca7-50ca-7b80-9093-b923ad47b7ff'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.004..0.004 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.005..0.006 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.925 ms
Execution Time: 5.112 ms
```

