### 2026-11-09T18:41:52Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 12545507328 bytes, kernel 5.15.167.4-microsoft-standard-WSL2

# P3-36 raw report (TICK-621)

Generated 2026-11-09T18:48:35Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
heap 1042 MB, indexes 650 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 4.02 / p95 8.11 / p99 14.22 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| agent list by group, default sort, page 1 | large | 2687 | 52.31 | 96.35 | 131.39 | 150 | meets |
| agent list by group, default sort, page 20 | large | 2473 | 57.03 | 92.46 | 203.55 | 150 | meets |
| requester list (heavy requesters), page 1 | large | 3022 | 49.27 | 75.44 | 88.63 | 150 | meets |
| requester list (heavy requesters), page 20 | large | 3023 | 49.04 | 76.75 | 90.87 | 150 | meets |
| admin list, default sort, page 1 | large | 3462 | 43.24 | 66.57 | 86.45 | 150 | meets |
| admin list, default sort, page 20 | large | 3805 | 39.46 | 59.76 | 74.54 | 150 | meets |
| agent list, filter[state]=new | large | 1054 | 147.50 | 192.59 | 213.33 | 150 | **MISS** |
| agent list, filter[priority]=1 (data) | large | 854 | 184.19 | 223.84 | 240.89 | 150 | **MISS** |
| agent list, filter[site]=s7 (rare data, < 1% of visible rows) | large | 858 | 184.05 | 224.44 | 244.69 | 150 | **MISS** |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (4.02 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### agent list by group, default sort, page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 2687 | 2.91 | 8.87 | 24.15 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 2687 | 3.74 | 8.07 | 17.49 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2687 | 2.93 | 7.53 | 20.41 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2687 | 3.30 | 7.30 | 17.39 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2687 | 6.22 | 12.46 | 23.18 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2687 | 20.20 | 46.34 | 68.44 |
| 7 | `commit` | 2687 | 4.00 | 22.40 | 46.26 |

### agent list by group, default sort, page 20

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 2473 | 3.13 | 9.71 | 24.60 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 2473 | 4.18 | 10.12 | 24.80 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2473 | 3.33 | 8.74 | 23.30 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2473 | 3.69 | 9.49 | 22.64 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2473 | 7.20 | 14.55 | 29.02 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2473 | 21.49 | 41.35 | 66.22 |
| 7 | `commit` | 2473 | 3.77 | 14.73 | 34.02 |

### requester list (heavy requesters), page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 3022 | 3.08 | 8.31 | 17.04 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 3022 | 4.17 | 8.71 | 16.05 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3022 | 3.40 | 7.59 | 13.52 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3022 | 3.72 | 8.06 | 12.25 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3022 | 6.20 | 11.48 | 17.68 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3022 | 16.30 | 30.38 | 40.54 |
| 7 | `commit` | 3022 | 3.60 | 12.25 | 21.41 |

### requester list (heavy requesters), page 20

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 3023 | 3.10 | 8.51 | 17.48 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 3023 | 4.16 | 8.68 | 14.76 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3023 | 3.40 | 8.01 | 15.42 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3023 | 3.69 | 8.05 | 13.61 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 3023 | 6.09 | 11.36 | 17.61 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 3023 | 16.05 | 30.25 | 41.55 |
| 7 | `commit` | 3023 | 3.56 | 11.39 | 20.96 |

### admin list, default sort, page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 3462 | 3.23 | 9.23 | 18.70 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 3462 | 4.25 | 9.10 | 16.39 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3462 | 3.54 | 8.46 | 16.51 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3462 | 3.97 | 8.86 | 14.25 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 3462 | 4.33 | 9.59 | 16.59 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 3462 | 11.26 | 24.14 | 34.91 |
| 7 | `commit` | 3462 | 3.32 | 10.07 | 19.42 |

### admin list, default sort, page 20

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 3805 | 2.97 | 7.78 | 15.64 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 3805 | 3.99 | 8.44 | 14.00 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3805 | 3.12 | 7.33 | 12.35 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3805 | 3.58 | 7.63 | 13.39 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 3805 | 4.00 | 8.11 | 13.17 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 3805 | 10.58 | 19.43 | 27.00 |
| 7 | `commit` | 3805 | 2.98 | 8.35 | 15.12 |

### agent list, filter[state]=new

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 1054 | 2.93 | 5.43 | 7.83 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1054 | 3.69 | 6.28 | 8.16 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1054 | 3.01 | 5.47 | 6.87 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1054 | 3.53 | 6.15 | 7.84 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1054 | 53.26 | 76.81 | 90.78 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1054 | 71.84 | 101.18 | 115.26 |
| 7 | `commit` | 1054 | 2.92 | 5.78 | 8.06 |

### agent list, filter[priority]=1 (data)

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 854 | 2.92 | 5.29 | 9.43 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 854 | 3.67 | 6.08 | 9.34 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 854 | 2.98 | 5.35 | 7.88 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 854 | 3.44 | 5.54 | 7.33 |
| 5 | `SELECT current_setting('statement_timeout')` | 854 | 2.62 | 4.90 | 6.98 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 854 | 2.41 | 4.34 | 6.11 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 854 | 65.75 | 80.95 | 94.39 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 854 | 87.88 | 111.49 | 127.37 |
| 9 | `SET LOCAL statement_timeout = '0'` | 854 | 2.92 | 7.58 | 17.58 |
| 10 | `commit` | 854 | 2.60 | 5.25 | 9.38 |

### agent list, filter[site]=s7 (rare data, < 1% of visible rows)

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'f2445c9c-2b64-4091-a693-42e7912aff23', true)` | 858 | 2.89 | 5.54 | 8.84 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 858 | 3.61 | 5.97 | 10.45 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 858 | 2.97 | 5.03 | 8.35 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 858 | 3.43 | 5.44 | 7.03 |
| 5 | `SELECT current_setting('statement_timeout')` | 858 | 2.44 | 4.88 | 6.67 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 858 | 2.39 | 3.92 | 6.10 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 858 | 74.36 | 93.46 | 103.71 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 858 | 80.65 | 101.56 | 113.82 |
| 9 | `SET LOCAL statement_timeout = '0'` | 858 | 2.85 | 5.43 | 9.04 |
| 10 | `commit` | 858 | 2.44 | 4.80 | 6.43 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### agent list by group, default sort, page 1

Request: `GET /api/v1/records/ticket` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.051..0.054 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.033..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.371 ms
Execution Time: 0.105 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.037..0.040 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.028..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.150 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.052..0.053 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.050..0.051 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.042..0.045 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.035..0.038 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.024..0.025 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.241 ms
Execution Time: 0.085 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=571.41..571.42 rows=1 width=8) (actual time=1.334..1.338 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..558.90 rows=1001 width=4) (actual time=0.058..1.217 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..633.09 rows=1134 width=4) (actual time=0.056..1.089 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..621.75 rows=1134 width=0) (actual time=0.056..0.915 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.44 rows=1001 width=0) (actual time=0.055..0.796 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.43 rows=1001 width=4) (actual time=0.054..0.656 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.29 rows=5723 width=4) (actual time=0.054..0.544 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.29 rows=5723 width=0) (actual time=0.046..0.374 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..428.14 rows=120 width=0) (never executed)
                          ->  Limit  (cost=0.42..426.94 rows=120 width=4) (never executed)
                                ->  Result  (cost=0.42..426.94 rows=120 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..426.94 rows=120 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..46.01 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..45.89 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..45.89 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.89 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.466 ms
Execution Time: 1.502 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1729.58 rows=1 width=1301) (actual time=0.328..2.616 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1721.25 rows=1 width=1266) (actual time=0.309..2.355 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1712.94 rows=1 width=1259) (actual time=0.294..2.148 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1711.79 rows=51 width=1259) (actual time=0.292..2.032 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..3802.44 rows=115 width=1259) (actual time=0.290..2.018 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3528.60 rows=51 width=1259) (actual time=0.129..1.825 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..395917.51 rows=5723 width=1259) (actual time=0.129..1.814 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..395917.51 rows=5723 width=1251) (actual time=0.122..1.784 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..181.69 rows=51 width=1259) (actual time=0.026..0.033 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..426.94 rows=120 width=1259) (actual time=0.025..0.032 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..426.94 rows=120 width=1251) (actual time=0.023..0.028 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..45.89 rows=12 width=1259) (actual time=0.021..0.022 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..45.89 rows=12 width=1259) (actual time=0.021..0.022 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.89 rows=12 width=1251) (actual time=0.019..0.019 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.111..0.116 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.111..0.115 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.076..0.103 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.074..0.100 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.057 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.047 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.904 ms
Execution Time: 2.929 ms
```

### agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ._assdT-c99e95sQ8ki0TktUA89iFsW3YSNKCIg2KPvs` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.059..0.062 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.039..0.041 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.416 ms
Execution Time: 0.118 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.043 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.032 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.137 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.183..0.186 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.180..0.182 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.081..0.086 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.065..0.070 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.045..0.046 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.015..0.017 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.327 ms
Execution Time: 0.228 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=571.41..571.42 rows=1 width=8) (actual time=1.672..1.679 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..558.90 rows=1001 width=4) (actual time=0.067..1.535 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..633.09 rows=1134 width=4) (actual time=0.066..1.382 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..621.75 rows=1134 width=0) (actual time=0.065..1.178 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.44 rows=1001 width=0) (actual time=0.064..0.955 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.43 rows=1001 width=4) (actual time=0.063..0.789 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.29 rows=5723 width=4) (actual time=0.063..0.655 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.29 rows=5723 width=0) (actual time=0.053..0.460 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..428.14 rows=120 width=0) (never executed)
                          ->  Limit  (cost=0.42..426.94 rows=120 width=4) (never executed)
                                ->  Result  (cost=0.42..426.94 rows=120 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..426.94 rows=120 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..46.01 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..45.89 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..45.89 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.89 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.065 ms
Execution Time: 1.928 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.38..1757.68 rows=1 width=1301) (actual time=0.307..2.534 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1749.34 rows=1 width=1266) (actual time=0.282..2.290 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.80..1741.03 rows=1 width=1259) (actual time=0.262..2.106 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.80..1739.88 rows=51 width=1259) (actual time=0.259..2.036 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.80..3865.79 rows=115 width=1259) (actual time=0.258..2.023 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..3591.62 rows=51 width=1259) (actual time=0.063..1.786 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..386581.76 rows=5490 width=1259) (actual time=0.063..1.775 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..386581.76 rows=5490 width=1251) (actual time=0.055..1.745 rows=49 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..182.16 rows=51 width=1259) (actual time=0.030..0.037 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..410.22 rows=115 width=1259) (actual time=0.030..0.036 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..410.22 rows=115 width=1251) (actual time=0.027..0.031 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..45.75 rows=12 width=1259) (actual time=0.027..0.035 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..45.75 rows=12 width=1259) (actual time=0.026..0.033 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.75 rows=12 width=1251) (actual time=0.024..0.029 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.134..0.143 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.133..0.142 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.091..0.125 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.089..0.121 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.062..0.068 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.049..0.058 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.039..0.041 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.020..0.021 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.016..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.003 ms
Execution Time: 3.042 ms
```

### requester list (heavy requesters), page 1

Request: `GET /api/v1/records/ticket` as `req-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.086..0.090 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.056..0.059 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.017..0.017 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.578 ms
Execution Time: 0.166 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.151..0.157 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.135..0.139 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.198 ms
Execution Time: 0.206 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.060..0.062 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.058..0.059 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.047..0.051 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.038..0.042 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.282 ms
Execution Time: 0.101 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=190.41..190.42 rows=1 width=8) (actual time=1.691..1.697 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..177.90 rows=1001 width=4) (actual time=0.135..1.546 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..180.21 rows=1014 width=4) (actual time=0.132..1.385 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..170.07 rows=1014 width=0) (actual time=0.131..1.178 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..97.94 rows=1001 width=0) (actual time=0.130..1.032 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..87.93 rows=1001 width=4) (actual time=0.128..0.862 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..129.28 rows=1474 width=4) (actual time=0.127..0.726 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..129.28 rows=1474 width=0) (actual time=0.062..0.474 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..45.95 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..45.83 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..45.83 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..45.83 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.077 ms
Execution Time: 1.939 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],51]`

```
Nested Loop Left Join  (cost=22.56..216.33 rows=1 width=1301) (actual time=0.155..0.754 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..207.99 rows=1 width=1266) (actual time=0.124..0.493 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.98..199.68 rows=1 width=1259) (actual time=0.104..0.337 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.98..198.53 rows=51 width=1259) (actual time=0.102..0.285 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.98..243.53 rows=64 width=1259) (actual time=0.100..0.274 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..175.74 rows=51 width=1259) (actual time=0.047..0.207 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..5067.33 rows=1474 width=1259) (actual time=0.047..0.199 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..5067.33 rows=1474 width=1251) (actual time=0.040..0.173 rows=51 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..45.83 rows=12 width=1259) (actual time=0.028..0.028 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..45.83 rows=12 width=1259) (actual time=0.027..0.028 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..45.83 rows=12 width=1251) (actual time=0.025..0.025 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.023..0.026 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.023..0.025 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.10 rows=1 width=1259) (actual time=0.017..0.019 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.10 rows=1 width=1251) (actual time=0.015..0.017 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.015..0.016 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.524 ms
Execution Time: 1.024 ms
```

### requester list (heavy requesters), page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTI5IDAzOjIwOjI1LjYwNTY3NSswMCIsImkiOiJkOWZlM2ZiMi02ZWU5LTdhMTktNjE1Ny0wODUxOWU2M2U3OTcifQ.mE3LRKDMeQnwttwyMvNmiH_GS1V0O7RPANKkyodtHwo` as `req-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.044..0.046 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.029..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.329 ms
Execution Time: 0.088 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.024..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.129 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.053..0.055 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.052..0.053 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.043..0.046 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.034..0.037 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.022..0.023 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.274 ms
Execution Time: 0.090 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=190.41..190.42 rows=1 width=8) (actual time=1.542..1.547 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..177.90 rows=1001 width=4) (actual time=0.086..1.418 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..180.21 rows=1014 width=4) (actual time=0.085..1.280 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..170.07 rows=1014 width=0) (actual time=0.084..1.070 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..97.94 rows=1001 width=0) (actual time=0.083..0.942 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..87.93 rows=1001 width=4) (actual time=0.082..0.785 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..129.28 rows=1474 width=4) (actual time=0.081..0.599 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..129.28 rows=1474 width=0) (actual time=0.072..0.419 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..45.95 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..45.83 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..45.83 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..45.83 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.793 ms
Execution Time: 1.656 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],"2026-09-29 03:20:25.605675+00","d9fe3fb2-6ee9-7a19-6157-08519e63e797",51]`

```
Nested Loop Left Join  (cost=22.57..216.64 rows=1 width=1301) (actual time=0.147..0.718 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..208.30 rows=1 width=1266) (actual time=0.121..0.477 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.99..199.99 rows=1 width=1259) (actual time=0.102..0.329 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.99..198.84 rows=51 width=1259) (actual time=0.100..0.280 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.99..236.99 rows=62 width=1259) (actual time=0.098..0.270 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..176.09 rows=51 width=1259) (actual time=0.048..0.207 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..4206.04 rows=1221 width=1259) (actual time=0.048..0.200 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..4206.04 rows=1221 width=1251) (actual time=0.042..0.176 rows=51 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..38.95 rows=10 width=1259) (actual time=0.027..0.028 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..38.95 rows=10 width=1259) (actual time=0.027..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..38.95 rows=10 width=1251) (actual time=0.025..0.025 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.021..0.023 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.020..0.022 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.11 rows=1 width=1259) (actual time=0.015..0.017 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.11 rows=1 width=1251) (actual time=0.014..0.015 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.013..0.015 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.606 ms
Execution Time: 1.016 ms
```

### admin list, default sort, page 1

Request: `GET /api/v1/records/ticket` as `admin-large`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.047..0.050 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.031..0.033 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.372 ms
Execution Time: 0.091 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.024..0.026 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.132 ms
Execution Time: 0.066 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.063..0.065 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.062..0.063 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.051..0.055 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.041..0.045 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.011 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.306 ms
Execution Time: 0.109 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND true LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0",["0119c445-10a8-4873-b476-6198cccbfc71"]]`

```
Aggregate  (cost=78.88..78.89 rows=1 width=8) (actual time=1.069..1.072 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.37 rows=1001 width=4) (actual time=0.059..0.879 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..12837.65 rows=194861 width=4) (actual time=0.057..0.690 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..12837.65 rows=194861 width=0) (actual time=0.044..0.432 rows=1001 loops=1)
                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.306 ms
Execution Time: 1.128 ms
```

Statement 6:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND true ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $7
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0",["0119c445-10a8-4873-b476-6198cccbfc71"],51]`

```
Limit  (cost=1.00..122.10 rows=51 width=1293) (actual time=0.095..0.569 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..462700.75 rows=194861 width=1293) (actual time=0.093..0.557 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..461239.29 rows=194861 width=1266) (actual time=0.074..0.497 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..397387.26 rows=194861 width=1258) (actual time=0.057..0.249 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..394943.20 rows=194861 width=1251) (actual time=0.038..0.188 rows=51 loops=1)
                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.013..0.015 rows=1 loops=1)
                                Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                Filter: (id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.562 ms
Execution Time: 0.648 ms
```

### admin list, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjU5OjU5LjU2MzExOCswMCIsImkiOiJkNmRhMTEwZi1jOGM3LTEzZmItZjRmNy1jNWUzYjFkMmMzMjQifQ.UdBdSEH7jJEp8_1J69YRmpLHuX20B-wLu4PinUxszRc` as `admin-large`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.044..0.047 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.029..0.031 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.322 ms
Execution Time: 0.086 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.037 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.026..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.125 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.057..0.059 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.056..0.057 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.047..0.050 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.037..0.040 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.024..0.025 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.011 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.258 ms
Execution Time: 0.094 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND true LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0",["0119c445-10a8-4873-b476-6198cccbfc71"]]`

```
Aggregate  (cost=78.88..78.89 rows=1 width=8) (actual time=0.613..0.614 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.37 rows=1001 width=4) (actual time=0.043..0.516 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..12837.65 rows=194861 width=4) (actual time=0.042..0.397 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..12837.65 rows=194861 width=0) (actual time=0.035..0.223 rows=1001 loops=1)
                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.195 ms
Execution Time: 0.648 ms
```

Statement 6:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND true AND ((r.updated_at, r.id) < ($7::timestamptz, $8::uuid)) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0","44f5be37-6af4-455b-9cea-5d1f1ccd49e0",["0119c445-10a8-4873-b476-6198cccbfc71"],"2026-09-30 23:59:59.563118+00","d6da110f-c8c7-13fb-f4f7-c5e3b1d2c324",51]`

```
Limit  (cost=1.00..122.23 rows=51 width=1293) (actual time=0.098..0.678 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..463152.40 rows=194838 width=1293) (actual time=0.095..0.664 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..461691.12 rows=194838 width=1266) (actual time=0.077..0.606 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..397846.39 rows=194838 width=1258) (actual time=0.058..0.345 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..395402.62 rows=194838 width=1251) (actual time=0.041..0.249 rows=51 loops=1)
                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:59:59.563118+00'::timestamp with time zone, 'd6da110f-c8c7-13fb-f4f7-c5e3b1d2c324'::uuid)))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.011..0.013 rows=1 loops=1)
                                Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                Filter: (id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.538 ms
Execution Time: 0.775 ms
```

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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.057..0.061 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.037..0.039 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.348 ms
Execution Time: 0.109 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.035..0.037 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.027..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.127 ms
Execution Time: 0.066 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.072..0.075 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.070..0.072 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.058..0.063 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.047..0.052 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.031..0.032 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.012..0.015 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.282 ms
Execution Time: 0.123 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=19906.80..19906.81 rows=1 width=8) (actual time=44.034..44.041 rows=1 loops=1)
  Buffers: shared hit=20873
  ->  Limit  (cost=0.55..19903.16 rows=291 width=4) (actual time=0.070..43.784 rows=1001 loops=1)
        Buffers: shared hit=20873
        ->  Result  (cost=0.55..19903.16 rows=291 width=4) (actual time=0.069..43.565 rows=1001 loops=1)
              Buffers: shared hit=20873
              ->  Append  (cost=0.55..19900.25 rows=291 width=0) (actual time=0.068..43.230 rows=1001 loops=1)
                    Buffers: shared hit=20873
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..19381.24 rows=283 width=0) (actual time=0.067..42.989 rows=1001 loops=1)
                          Buffers: shared hit=20873
                          ->  Limit  (cost=0.55..19378.41 rows=283 width=4) (actual time=0.066..42.755 rows=1001 loops=1)
                                Buffers: shared hit=20873
                                ->  Result  (cost=0.55..19378.41 rows=283 width=4) (actual time=0.066..42.562 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=20873
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..19378.41 rows=283 width=0) (actual time=0.057..42.282 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (state = ANY ('{new}'::text[]))
                                            Rows Removed by Filter: 19547
                                            Buffers: shared hit=20873
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..427.15 rows=6 width=0) (never executed)
                          ->  Limit  (cost=0.42..427.09 rows=6 width=4) (never executed)
                                ->  Result  (cost=0.42..427.09 rows=6 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..427.09 rows=6 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..45.91 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..45.90 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..45.90 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.90 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.192 ms
Execution Time: 44.168 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=19433.81..19860.03 rows=1 width=1301) (actual time=38.149..38.604 rows=51 loops=1)
  Buffers: shared hit=24889
  ->  Nested Loop  (cost=19433.51..19851.69 rows=1 width=1266) (actual time=38.122..38.406 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=24736
        ->  Subquery Scan on x  (cost=19433.23..19843.38 rows=1 width=1259) (actual time=38.104..38.263 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=24583
              ->  Limit  (cost=19433.23..19842.23 rows=51 width=1259) (actual time=38.101..38.214 rows=51 loops=1)
                    Buffers: shared hit=24583
                    ->  Merge Append  (cost=19433.23..19906.39 rows=59 width=1259) (actual time=38.099..38.204 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=24583
                          ->  Limit  (cost=19387.85..19387.98 rows=51 width=1259) (actual time=37.848..37.866 rows=50 loops=1)
                                Buffers: shared hit=24492
                                ->  Sort  (cost=19387.85..19388.56 rows=283 width=1259) (actual time=37.847..37.858 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=24492
                                      ->  Result  (cost=0.55..19378.41 rows=283 width=1259) (actual time=0.070..36.672 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=24492
                                            ->  Index Scan using records_list_group on records r  (cost=0.55..19378.41 rows=283 width=1251) (actual time=0.062..36.203 rows=1168 loops=1)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (state = ANY ('{new}'::text[]))
                                                  Rows Removed by Filter: 22950
                                                  Buffers: shared hit=24492
                          ->  Limit  (cost=0.42..427.09 rows=6 width=1259) (actual time=0.075..0.137 rows=2 loops=1)
                                Buffers: shared hit=46
                                ->  Result  (cost=0.42..427.09 rows=6 width=1259) (actual time=0.072..0.133 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=46
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..427.09 rows=6 width=1251) (actual time=0.059..0.119 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=46
                          ->  Limit  (cost=0.42..45.90 rows=1 width=1259) (actual time=0.052..0.053 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..45.90 rows=1 width=1259) (actual time=0.052..0.052 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.90 rows=1 width=1251) (actual time=0.047..0.047 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.117..0.119 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.116..0.119 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.103..0.106 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.099..0.102 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.060..0.063 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.043..0.050 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.018 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.769 ms
Execution Time: 38.859 ms
```

### agent list, filter[priority]=1 (data)

Request: `GET /api/v1/records/ticket?filter%5Bpriority%5D=1` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.036..0.038 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.022..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.286 ms
Execution Time: 0.072 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.025..0.027 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.019..0.020 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.104 ms
Execution Time: 0.051 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.048..0.049 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.046..0.047 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.037..0.041 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.031..0.033 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.020..0.021 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.254 ms
Execution Time: 0.081 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.016 ms
Execution Time: 0.017 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}"]`

```
Aggregate  (cost=19905.46..19905.47 rows=1 width=8) (actual time=38.374..38.381 rows=1 loops=1)
  Buffers: shared hit=20546
  ->  Limit  (cost=0.55..19904.71 rows=60 width=4) (actual time=0.084..38.201 rows=1001 loops=1)
        Buffers: shared hit=20546
        ->  Result  (cost=0.55..19904.71 rows=60 width=4) (actual time=0.083..38.011 rows=1001 loops=1)
              Buffers: shared hit=20546
              ->  Append  (cost=0.55..19904.11 rows=60 width=0) (actual time=0.082..37.773 rows=1001 loops=1)
                    Buffers: shared hit=20546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..19386.13 rows=57 width=0) (actual time=0.081..37.590 rows=1001 loops=1)
                          Buffers: shared hit=20546
                          ->  Limit  (cost=0.55..19385.56 rows=57 width=4) (actual time=0.080..37.417 rows=1001 loops=1)
                                Buffers: shared hit=20546
                                ->  Result  (cost=0.55..19385.56 rows=57 width=4) (actual time=0.080..37.273 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=20546
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..19385.56 rows=57 width=0) (actual time=0.071..37.040 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"priority": "1"}'::jsonb)
                                            Rows Removed by Filter: 19225
                                            Buffers: shared hit=20546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..427.26 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..427.25 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..427.25 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..427.25 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..45.93 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..45.92 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..45.92 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.92 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.925 ms
Execution Time: 38.500 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}",51]`

```
Nested Loop Left Join  (cost=19433.18..19897.36 rows=1 width=1301) (actual time=43.464..43.894 rows=51 loops=1)
  Buffers: shared hit=24863
  ->  Nested Loop  (cost=19432.89..19889.03 rows=1 width=1266) (actual time=43.443..43.687 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=24710
        ->  Subquery Scan on x  (cost=19432.60..19880.72 rows=1 width=1259) (actual time=43.427..43.554 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=24557
              ->  Limit  (cost=19432.60..19879.57 rows=51 width=1259) (actual time=43.424..43.508 rows=51 loops=1)
                    Buffers: shared hit=24557
                    ->  Merge Append  (cost=19432.60..19905.86 rows=54 width=1259) (actual time=43.422..43.500 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=24557
                          ->  Limit  (cost=19387.23..19387.35 rows=51 width=1259) (actual time=43.199..43.220 rows=50 loops=1)
                                Buffers: shared hit=24492
                                ->  Sort  (cost=19387.23..19387.37 rows=57 width=1259) (actual time=43.197..43.212 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=24492
                                      ->  Result  (cost=0.55..19385.56 rows=57 width=1259) (actual time=0.075..42.129 rows=1182 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=24492
                                            ->  Index Scan using records_list_group on records r  (cost=0.55..19385.56 rows=57 width=1251) (actual time=0.070..41.733 rows=1182 loops=1)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"priority": "1"}'::jsonb)
                                                  Rows Removed by Filter: 22936
                                                  Buffers: shared hit=24492
                          ->  Limit  (cost=0.42..427.25 rows=1 width=1259) (actual time=0.078..0.079 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.42..427.25 rows=1 width=1259) (actual time=0.076..0.076 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=23
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..427.25 rows=1 width=1251) (actual time=0.063..0.063 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 19
                                            Buffers: shared hit=23
                          ->  Limit  (cost=0.42..45.92 rows=1 width=1259) (actual time=0.023..0.053 rows=2 loops=1)
                                Buffers: shared hit=17
                                ->  Result  (cost=0.42..45.92 rows=1 width=1259) (actual time=0.022..0.052 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=17
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.92 rows=1 width=1251) (actual time=0.019..0.048 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 12
                                            Buffers: shared hit=17
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.116..0.119 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.115..0.118 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.108..0.110 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.105..0.107 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.056 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.388 ms
Execution Time: 44.071 ms
```

### agent list, filter[site]=s7 (rare data, < 1% of visible rows)

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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.037..0.039 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.024..0.025 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.285 ms
Execution Time: 0.072 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.027..0.029 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.021..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.102 ms
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

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.046..0.048 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.045..0.046 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.037..0.040 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.032..0.035 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.020..0.021 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.221 ms
Execution Time: 0.078 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.017 ms
Execution Time: 0.019 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}"]`

```
Aggregate  (cost=19905.46..19905.47 rows=1 width=8) (actual time=45.334..45.341 rows=1 loops=1)
  Buffers: shared hit=25107
  ->  Limit  (cost=0.55..19904.71 rows=60 width=4) (actual time=0.221..45.291 rows=121 loops=1)
        Buffers: shared hit=25107
        ->  Result  (cost=0.55..19904.71 rows=60 width=4) (actual time=0.220..45.259 rows=121 loops=1)
              Buffers: shared hit=25107
              ->  Append  (cost=0.55..19904.11 rows=60 width=0) (actual time=0.220..45.217 rows=121 loops=1)
                    Buffers: shared hit=25107
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..19386.13 rows=57 width=0) (actual time=0.219..43.993 rows=120 loops=1)
                          Buffers: shared hit=24492
                          ->  Limit  (cost=0.55..19385.56 rows=57 width=4) (actual time=0.218..43.958 rows=120 loops=1)
                                Buffers: shared hit=24492
                                ->  Result  (cost=0.55..19385.56 rows=57 width=4) (actual time=0.218..43.927 rows=120 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=24492
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..19385.56 rows=57 width=0) (actual time=0.210..43.878 rows=120 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"site": "s7"}'::jsonb)
                                            Rows Removed by Filter: 23998
                                            Buffers: shared hit=24492
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..427.26 rows=1 width=0) (actual time=0.347..0.961 rows=1 loops=1)
                          Buffers: shared hit=523
                          ->  Limit  (cost=0.42..427.25 rows=1 width=4) (actual time=0.346..0.959 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..427.25 rows=1 width=4) (actual time=0.344..0.956 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..427.25 rows=1 width=0) (actual time=0.329..0.940 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..45.93 rows=1 width=0) (actual time=0.135..0.136 rows=0 loops=1)
                          Buffers: shared hit=67
                          ->  Limit  (cost=0.42..45.92 rows=1 width=4) (actual time=0.134..0.135 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..45.92 rows=1 width=4) (actual time=0.133..0.133 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.92 rows=1 width=0) (actual time=0.126..0.126 rows=0 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (actual time=0.087..0.090 rows=0 loops=1)
                          Buffers: shared hit=25
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (actual time=0.086..0.089 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (actual time=0.085..0.087 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=25
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (actual time=0.083..0.085 rows=0 loops=1)
                                            Buffers: shared hit=25
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.050..0.052 rows=3 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=13
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.040 rows=3 loops=1)
                                                        Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Heap Blocks: exact=3
                                                        Buffers: shared hit=13
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                              Buffers: shared hit=10
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=8
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.920 ms
Execution Time: 45.447 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $9
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

Arguments: `["f2445c9c-2b64-4091-a693-42e7912aff23","7572bf4b-47c9-40b2-8a23-6f8149d0784a",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}",51]`

```
Nested Loop Left Join  (cost=19433.18..19897.36 rows=1 width=1301) (actual time=42.715..43.615 rows=51 loops=1)
  Buffers: shared hit=25413
  ->  Nested Loop  (cost=19432.89..19889.03 rows=1 width=1266) (actual time=42.696..43.448 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=25260
        ->  Subquery Scan on x  (cost=19432.60..19880.72 rows=1 width=1259) (actual time=42.684..43.324 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=25107
              ->  Limit  (cost=19432.60..19879.57 rows=51 width=1259) (actual time=42.681..43.281 rows=51 loops=1)
                    Buffers: shared hit=25107
                    ->  Merge Append  (cost=19432.60..19905.86 rows=54 width=1259) (actual time=42.680..43.273 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=25107
                          ->  Limit  (cost=19387.23..19387.35 rows=51 width=1259) (actual time=42.119..42.135 rows=50 loops=1)
                                Buffers: shared hit=24492
                                ->  Sort  (cost=19387.23..19387.37 rows=57 width=1259) (actual time=42.117..42.128 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 187kB
                                      Buffers: shared hit=24492
                                      ->  Result  (cost=0.55..19385.56 rows=57 width=1259) (actual time=0.217..41.947 rows=120 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=24492
                                            ->  Index Scan using records_list_group on records r  (cost=0.55..19385.56 rows=57 width=1251) (actual time=0.211..41.871 rows=120 loops=1)
                                                  Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"site": "s7"}'::jsonb)
                                                  Rows Removed by Filter: 23998
                                                  Buffers: shared hit=24492
                          ->  Limit  (cost=0.42..427.25 rows=1 width=1259) (actual time=0.339..0.893 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..427.25 rows=1 width=1259) (actual time=0.336..0.890 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..427.25 rows=1 width=1251) (actual time=0.325..0.878 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                          ->  Limit  (cost=0.42..45.92 rows=1 width=1259) (actual time=0.126..0.127 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..45.92 rows=1 width=1259) (actual time=0.126..0.126 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..45.92 rows=1 width=1251) (actual time=0.123..0.123 rows=0 loops=1)
                                            Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.090..0.094 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.090..0.093 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.082..0.085 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.080..0.082 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'f2445c9c-2b64-4091-a693-42e7912aff23'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7572bf4b-47c9-40b2-8a23-6f8149d0784a'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.436 ms
Execution Time: 43.821 ms
```

