# P3-36 raw report (TICK-621)

Generated 2026-11-10T13:02:05Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 812900, open 133178, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 410263
record_shares 15815
heap 1058 MB, indexes 772 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.09 / p95 1.44 / p99 1.81 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | large | 635 | 29.94 | 41.35 | 46.96 | 50 | meets |
| create, agent | large | 3856 | 39.61 | 52.08 | 73.12 | 50 | **MISS** |
| create, admin | large | 3649 | 39.95 | 64.03 | 83.40 | 50 | **MISS** |
| create, small tenant agent | small | 2586 | 66.05 | 97.45 | 115.33 | 50 | **MISS** |
| create, agent, gapless type | large | 1968 | 65.68 | 183.60 | 269.02 | 0 | gapless type: reported, not scored (D-3-9) |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.09 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 635 | 1.79 / 3.37 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent | 3856 | 2.35 / 3.86 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, admin | 3649 | 2.32 / 4.44 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, small tenant agent | 2586 | 1.92 / 3.65 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, gapless type | 1968 | 33.84 / 151.37 | 8.01 / 12.71 | 1 | 1.09 / 1.09 | 6.67 / 10.96 | 0.00 / 0.75 |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 685 | 0.05 / 0.25 / 0.48 | 2.06 / 1.93 / 4.42 | 0 | none |
| create, agent | 3906 | 0.03 / 0.25 / 0.48 | 2.60 / 2.25 / 4.79 | 0 | none |
| create, admin | 3650 | 0.02 / 0.25 / 0.48 | 2.63 / 2.22 / 4.91 | 0 | none |
| create, small tenant agent | 2636 | 0.07 / 0.26 / 0.50 | 2.34 / 2.05 / 4.73 | 0 | none |
| create, agent, gapless type | 0 | - | - | 0 | none |

## Data-filter statement timeout (TICK-647)

Default timeout 2s (store.DataFilterTimeout); the bound check runs with 2ms. A 422 is `filter_too_expensive`. Latency includes the 422s.

| Scenario | Requests | 422 | p50 | p95 | max |
| --- | --- | --- | --- | --- | --- |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### create, agent (1 client)

Round trips: 9, statements: 19, statements per batch: [4 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 1270 | 1.64 | 3.11 | 3.85 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 635 | 1.75 | 3.14 | 3.88 |
| 3 | `commit` | 635 | 4.80 | 6.50 | 8.37 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 1270 | 1.64 | 3.11 | 3.85 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 635 | 3.43 | 6.91 | 9.02 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 635 | 1.91 | 3.78 | 4.58 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 635 | 1.79 | 3.37 | 4.21 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 635 | 2.87 | 5.16 | 6.31 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 635 | 5.87 | 7.98 | 9.52 |

### create, agent

Round trips: 9, statements: 19, statements per batch: [4 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 7712 | 1.81 | 3.46 | 5.15 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 3856 | 2.28 | 3.56 | 5.19 |
| 3 | `commit` | 3856 | 6.95 | 10.15 | 12.13 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 7712 | 1.81 | 3.46 | 5.15 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 3856 | 4.55 | 8.16 | 13.35 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 3856 | 2.46 | 4.11 | 6.50 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 3856 | 2.35 | 3.86 | 5.46 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 3856 | 4.09 | 5.88 | 7.76 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 3856 | 8.73 | 11.75 | 15.02 |

### create, admin

Round trips: 9, statements: 19, statements per batch: [4 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 7298 | 1.79 | 3.85 | 6.20 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 3649 | 2.23 | 3.88 | 6.31 |
| 3 | `commit` | 3649 | 7.14 | 15.92 | 25.87 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 7298 | 1.79 | 3.85 | 6.20 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 3649 | 4.48 | 9.71 | 16.27 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 3649 | 2.42 | 4.50 | 6.87 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 3649 | 2.32 | 4.44 | 7.11 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 3649 | 3.95 | 6.14 | 8.49 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 3649 | 8.86 | 17.76 | 30.75 |

### create, small tenant agent

Round trips: 9, statements: 19, statements per batch: [4 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'cb337266-3ed7-4a6b-bd4d-10853cf350f4', true)` | 14 | 1.76 | 3.73 | 3.73 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 2586 | 1.82 | 3.00 | 4.39 |
| 3 | `commit` | 2586 | 21.47 | 36.40 | 56.06 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'cb337266-3ed7-4a6b-bd4d-10853cf350f4', true)` | 14 | 1.76 | 3.73 | 3.73 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 2586 | 4.05 | 7.28 | 13.12 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2586 | 2.27 | 4.07 | 5.92 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 2586 | 1.92 | 3.65 | 5.61 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 2586 | 3.48 | 5.27 | 7.17 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2586 | 20.77 | 38.60 | 53.88 |

### create, agent, gapless type

Round trips: 8, statements: 15, statements per batch: [4 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 3936 | 1.61 | 3.02 | 4.89 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 1968 | 1.73 | 3.17 | 5.96 |
| 3 | `commit` | 1968 | 8.33 | 12.96 | 16.31 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)` | 3936 | 1.61 | 3.02 | 4.89 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 1968 | 3.55 | 6.20 | 10.28 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1968 | 2.02 | 3.59 | 5.13 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 1968 | 33.84 | 151.37 | 230.70 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1968 | 7.76 | 12.05 | 15.59 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### create, agent (1 client)

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
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
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 9 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
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
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 9 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, admin

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
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
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 9 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, small tenant agent

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'cb337266-3ed7-4a6b-bd4d-10853cf350f4', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'cb337266-3ed7-4a6b-bd4d-10853cf350f4', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 6 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 7 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 8 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 9 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent, gapless type

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
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
BEGIN; SELECT set_config('app.tenant_id', '39ca0840-b1a8-4532-9ac9-4fb11965c638', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
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

