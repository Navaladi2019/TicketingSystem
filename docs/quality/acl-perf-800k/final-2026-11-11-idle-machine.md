# P3-36 raw report (TICK-621)

Generated 2026-11-11T03:04:38Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 818749, open 139027, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 413579
record_shares 15815
heap 1066 MB, indexes 778 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.08 / p95 1.69 / p99 2.81 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_logical_emit_message(true, ...); COMMIT` probe, a commit that flushes WAL with no INSERT, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). An attempt is also void if a container other than the run's database, the buildkit builder and the three Multica platform containers runs during it. A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 1 | large | 1132 | 16.34 | 22.98 | 30.56 | 0.58 | 2.91 | 6.48 | 4 | 0 | 50 | meets |
| create, agent | 1 | large | 5127 | 29.00 | 42.39 | 54.86 | 1.08 | 3.78 | 11.42 | 4 | 0 | 50 | meets |
| create, admin | 1 | large | 5171 | 29.05 | 42.10 | 51.49 | 0.55 | 3.25 | 11.32 | 4 | 0 | 50 | meets |
| create, small tenant agent | 1 | small | 5119 | 27.95 | 47.20 | 75.78 | 1.09 | 2.87 | 11.21 | 4 | 0 | 50 | meets |
| create, agent, gapless type | 1 | large | 1994 | 58.94 | 205.70 | 339.66 | 1.59 | 3.20 | 6.85 | 4 | 0 | 0 | gapless type: reported, not scored (D-3-9) |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.08 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client), attempt 1 | 1132 | 1.69 / 2.76 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, attempt 1 | 5127 | 2.88 / 5.02 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, admin, attempt 1 | 5171 | 2.85 / 4.73 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, small tenant agent, attempt 1 | 5119 | 2.83 / 5.04 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, gapless type | 1994 | 39.34 / 186.20 | 7.18 / 10.42 | 1 | 1.08 / 1.08 | 5.77 / 8.77 | 0.54 / 1.12 |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |
| create, agent (1 client), attempt 1 | 1182 | 0.02 / 0.25 / 0.48 | 1.81 / 1.81 / 3.35 | 0 | none |
| create, agent, attempt 1 | 5177 | 0.04 / 0.26 / 0.49 | 3.26 / 3.37 / 6.36 | 0 | none |
| create, admin, attempt 1 | 5172 | 0.02 / 0.26 / 0.48 | 3.16 / 3.33 / 5.01 | 0 | none |
| create, small tenant agent, attempt 1 | 5169 | 0.03 / 0.26 / 0.49 | 3.19 / 3.20 / 6.01 | 0 | none |
| create, agent, gapless type | 0 | - | - | 0 | none |

## Attempt setup and host-probe samples (creates)

Before each create attempt: wait until no container runs other than the run's database, `buildx_buildkit_nava-ci0` and the three `multica-*` platform containers (docker ps is polled every 5 s during the attempt too; another container voids it), then until no autovacuum or analyze backend works in the bench database, then the host-speed probe and the storage probe. `pg_stat_activity` is read at the probe's midpoint (every non-idle backend and every non-client backend, other than the reader); `docker stats` is one sample taken while the probe runs. The seed was quiesced once with `VACUUM (ANALYZE)` and `CHECKPOINT`.

### create, agent (1 client), attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 0.58 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1390 cpu=0.02% mem=8.49%
  buildx_buildkit_nava-ci0 cpu=0.01% mem=0.08%
  multica-backend-1 cpu=0.34% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=0.32% mem=0.89%
```

### create, agent, attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 1.08 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1390 cpu=0.03% mem=8.54%
  buildx_buildkit_nava-ci0 cpu=0.01% mem=0.08%
  multica-backend-1 cpu=0.10% mem=0.14%
  multica-frontend-1 cpu=0.02% mem=0.60%
  multica-postgres-1 cpu=0.10% mem=0.89%
```

### create, admin, attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 0.55 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1390 cpu=0.20% mem=8.79%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.08%
  multica-backend-1 cpu=0.09% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=0.10% mem=0.89%
```

### create, small tenant agent, attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 1.09 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1390 cpu=9.80% mem=8.97%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.08%
  multica-backend-1 cpu=0.23% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=0.19% mem=0.90%
```

### create, agent, gapless type

Isolation wait 0s, quiesce wait 0s, host probe p50 1.59 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1390 cpu=0.03% mem=9.14%
  buildx_buildkit_nava-ci0 cpu=0.01% mem=0.08%
  multica-backend-1 cpu=0.17% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=0.24% mem=0.90%
```


## Data-filter statement timeout (TICK-647)

Default timeout 2s (store.DataFilterTimeout); the bound check runs with 2ms. A 422 is `filter_too_expensive`. Latency includes the 422s.

| Scenario | Requests | 422 | p50 | p95 | max |
| --- | --- | --- | --- | --- | --- |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### create, agent (1 client), attempt 1

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 1132 | 3.60 | 6.07 | 9.06 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1132 | 1.87 | 3.14 | 4.01 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 1132 | 1.69 | 2.76 | 3.36 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 1132 | 6.48 | 9.11 | 11.47 |

### create, agent, attempt 1

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 5127 | 6.39 | 11.69 | 16.60 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 5127 | 3.09 | 5.23 | 7.27 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 5127 | 2.88 | 5.02 | 7.04 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 5127 | 11.42 | 16.53 | 22.45 |

### create, admin, attempt 1

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 5171 | 6.39 | 12.25 | 17.30 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 5171 | 2.96 | 5.12 | 7.38 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 5171 | 2.85 | 4.73 | 6.32 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 5171 | 11.32 | 15.48 | 17.95 |

### create, small tenant agent, attempt 1

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 5119 | 6.16 | 12.92 | 23.14 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 5119 | 2.96 | 5.51 | 8.92 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 5119 | 2.83 | 5.04 | 7.83 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 5119 | 11.21 | 16.91 | 25.10 |

### create, agent, gapless type

Round trips: 4, statements: 15, statements per batch: [8 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 1994 | 5.18 | 8.61 | 12.17 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1994 | 3.39 | 5.43 | 7.39 |
| 3 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 1994 | 39.34 | 186.20 | 316.25 |
| 4 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 1994 | 6.85 | 9.85 | 12.13 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### create, agent (1 client), attempt 1

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text AS payload) SELECT 1 / count(*)::int FROM ins WHERE number = $6::text AND created_at = $12::timestamptz AND updated_at = $12::timestamptz AND payload = $21::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22) ; WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent, attempt 1

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text AS payload) SELECT 1 / count(*)::int FROM ins WHERE number = $6::text AND created_at = $12::timestamptz AND updated_at = $12::timestamptz AND payload = $21::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22) ; WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, admin, attempt 1

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text AS payload) SELECT 1 / count(*)::int FROM ins WHERE number = $6::text AND created_at = $12::timestamptz AND updated_at = $12::timestamptz AND payload = $21::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22) ; WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, small tenant agent, attempt 1

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit = off ; INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1, now() ; COMMIT
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text AS payload) SELECT 1 / count(*)::int FROM ins WHERE number = $6::text AND created_at = $12::timestamptz AND updated_at = $12::timestamptz AND payload = $21::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22) ; WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

### create, agent, gapless type

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_timeout', $1, true) ; INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at) VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second', now() + $6::double precision * interval '1 second') ON CONFLICT (tenant_id, principal_id, key) DO NOTHING ; SELECT set_config('lock_timeout', '0', true) ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
```

Statement 2 (write, not EXPLAINed):

```sql
BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL) ; SELECT ((data - $6::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($8::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($9::text, requester_id) END || jsonb_build_object($10::text, state))::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, $4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records
```

Statement 3 (write, not EXPLAINed):

```sql
BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 ; INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, $15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4), $6, $7, $8, $9, $10, $11, $11, $12, $13, $14) RETURNING number, created_at, updated_at, ((data - $16::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($17::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($18::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($19::text, requester_id) END || jsonb_build_object($20::text, state))::text ; INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256, diff_truncated, diff, diff_salt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 4 (write, not EXPLAINed):

```sql
BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' RETURNING 1) SELECT 1 / count(*) FROM done ; COMMIT
```

