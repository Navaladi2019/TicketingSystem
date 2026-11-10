# P3-36 raw report (TICK-621)

Generated 2026-11-10T13:10:40Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 811322, open 131600, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 407443
record_shares 15815
heap 1056 MB, indexes 768 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.07 / p95 1.71 / p99 2.59 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | large | 549 | 34.16 | 51.57 | 60.97 | 50 | **MISS** |
| create, agent | large | 2344 | 69.45 | 93.68 | 113.73 | 50 | **MISS** |
| create, admin | large | 2085 | 73.02 | 102.34 | 122.43 | 50 | **MISS** |
| create, small tenant agent | small | 3828 | 39.49 | 55.05 | 82.50 | 50 | **MISS** |
| create, agent, gapless type | large | 2310 | 56.03 | 150.68 | 232.01 | 0 | gapless type: reported, not scored (D-3-9) |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.07 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 549 | 2.05 / 3.98 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent | 2344 | 2.20 / 3.77 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, admin | 2085 | 1.96 / 3.97 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, small tenant agent | 3828 | 2.31 / 4.02 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, gapless type | 2310 | 27.90 / 123.56 | 6.76 / 10.06 | 1 | 1.07 / 1.07 | 5.44 / 8.51 | 0.00 / 0.76 |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 599 | 0.04 / 0.25 / 0.48 | 2.49 / 2.25 / 4.76 | 0 | none |
| create, agent | 2394 | 0.16 / 0.28 / 1.42 | 2.49 / 2.19 / 4.84 | 0 | none |
| create, admin | 2086 | 0.14 / 0.28 / 1.46 | 2.44 / 2.17 / 4.88 | 0 | none |
| create, small tenant agent | 3878 | 0.02 / 0.25 / 0.48 | 2.61 / 2.29 / 4.83 | 0 | none |
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
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 1098 | 1.85 | 3.65 | 4.26 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 549 | 1.92 | 3.78 | 4.48 |
| 3 | `commit` | 549 | 4.95 | 6.84 | 7.90 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 1098 | 1.85 | 3.65 | 4.26 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 549 | 4.18 | 7.88 | 9.61 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 549 | 2.41 | 4.61 | 5.58 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 549 | 2.05 | 3.98 | 4.56 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 549 | 3.39 | 6.17 | 7.52 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 549 | 6.12 | 8.84 | 10.84 |

### create, agent

Round trips: 9, statements: 19, statements per batch: [4 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 4688 | 1.81 | 3.53 | 5.09 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 2344 | 1.96 | 3.41 | 4.91 |
| 3 | `commit` | 2344 | 22.31 | 33.02 | 46.42 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 4688 | 1.81 | 3.53 | 5.09 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 2344 | 4.29 | 8.02 | 14.69 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2344 | 2.32 | 4.27 | 6.19 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 2344 | 2.20 | 3.77 | 5.68 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 2344 | 3.52 | 5.73 | 8.36 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2344 | 23.19 | 39.38 | 50.66 |

### create, admin

Round trips: 9, statements: 19, statements per batch: [4 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 4170 | 1.88 | 3.56 | 5.34 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 2085 | 1.81 | 3.55 | 4.80 |
| 3 | `commit` | 2085 | 24.56 | 40.74 | 56.40 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 4170 | 1.88 | 3.56 | 5.34 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 2085 | 4.21 | 7.58 | 16.13 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2085 | 2.27 | 4.46 | 6.14 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 2085 | 1.96 | 3.97 | 5.64 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 2085 | 3.37 | 5.73 | 7.77 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2085 | 25.90 | 40.93 | 56.56 |

### create, small tenant agent

Round trips: 9, statements: 19, statements per batch: [4 2 5 2 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '2f2254bd-8a7d-4134-a258-9b84937ab17a', true)` | 20 | 2.18 | 2.84 | 3.20 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 3828 | 2.25 | 3.69 | 5.71 |
| 3 | `commit` | 3828 | 6.92 | 10.22 | 12.46 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '2f2254bd-8a7d-4134-a258-9b84937ab17a', true)` | 20 | 2.18 | 2.84 | 3.20 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 3828 | 4.57 | 8.77 | 15.04 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 3828 | 2.56 | 4.41 | 6.88 |
| 7 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 3828 | 2.31 | 4.02 | 6.44 |
| 8 | `BATCH: INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, …` | 3828 | 4.10 | 6.19 | 8.78 |
| 9 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 3828 | 8.82 | 11.80 | 13.98 |

### create, agent, gapless type

Round trips: 8, statements: 15, statements per batch: [4 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 4620 | 1.67 | 2.98 | 4.53 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 2310 | 1.74 | 3.34 | 4.60 |
| 3 | `commit` | 2310 | 6.67 | 9.01 | 10.67 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)` | 4620 | 1.67 | 2.98 | 4.53 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 2310 | 3.58 | 6.31 | 9.05 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2310 | 1.89 | 3.58 | 4.75 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 2310 | 27.90 | 123.56 | 204.68 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2310 | 6.51 | 9.58 | 11.50 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### create, agent (1 client)

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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
BEGIN; SELECT set_config('app.tenant_id', '2f2254bd-8a7d-4134-a258-9b84937ab17a', true)
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
BEGIN; SELECT set_config('app.tenant_id', '2f2254bd-8a7d-4134-a258-9b84937ab17a', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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
BEGIN; SELECT set_config('app.tenant_id', 'ed52aaaa-ef96-485c-a65a-3852ebf69850', true)
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

