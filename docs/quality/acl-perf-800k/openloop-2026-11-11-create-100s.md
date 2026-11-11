<!-- TICK-700, run at 712fe7cb on 2026-11-11 04:43 UTC: NAVA_PERF_RATE=100,200 NAVA_PERF_SECONDS=60 NAVA_PERF_WARMUP=10 NAVA_PERF_ONLY='^create, agent(, gapless type)?$'. Note: the open-loop table's "Allocator acquire p95" column reads 0.00 because the allocator pool acquires outside the request's trace context; it was removed from the harness after this run. The allocator's real acquire wait is in the number allocation table (nava_numbering_alloc_acquire). -->

# P3-36 raw report (TICK-621)

Generated 2026-11-11T04:48:58Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 60 s per scenario, through the real handlers.

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
records 828153, open 148431, priority 1 40021, avg data bytes 1074
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 428153
record_shares 15815
heap 1078 MB, indexes 791 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 0.56 / p95 1.12 / p99 1.44 ms. Each statement in the tables below pays about this much before the database does any work.

## Open-loop create (TICK-700, informational: no target, no verdict)

Creates are sent on a fixed schedule at the offered rate, whether or not earlier ones have answered, after a 10s warm-up at the same rate; the window below is 1m0s. Latency runs from each request's **scheduled** send time, so queueing counts (no coordinated omission). Achieved is the successful creates answered inside the window, per second. In flight is sampled at each arrival, itself included; an arrival that finds 2000 in flight is dropped. API pool: 10 connections (NAVA_PERF_CLIENTS + 2); allocator pool: 4. In-process `httptest`, no PgBouncer, no network. Host probes are reported, not gated.

| Row | Offered /s | Achieved /s | Sent | Dropped | OK | p50 | p95 | p99 | max | In flight max / p95 | 409 | 503 retry_later | Other errors | Counter wait p50 / p95 | Commit p50 | API pool acquire p50 / p95 / max | Allocator acquire p95 | Drain after window | Host probe p50 | Storage probe p50 | Other containers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent, open loop 100/s | 100 | 100.0 | 6000 | 0 | 6000 | 15.43 | 21.34 | 29.62 | 57.09 | 6 / 3 | 0 | 0 | 0 | 1.40 / 2.31 | 6.14 | 0.00 / 0.00 / 3.12 | 0.00 | 5ms | 0.55 | 2.90 | none |
| create, agent, open loop 200/s | 200 | 199.9 | 12000 | 0 | 12000 | 20.73 | 33.15 | 47.77 | 124.72 | 20 / 7 | 0 | 0 | 0 | 2.28 / 3.63 | 7.68 | 0.00 / 0.00 / 43.35 | 0.00 | 22ms | 0.55 | 2.99 | none |
| create, agent, gapless type, open loop 100/s | 100 | 99.7 | 6000 | 0 | 6000 | 17.32 | 648.68 | 971.28 | 1433.38 | 102 / 64 | 0 | 0 | 0 | 3.38 / 89.43 | 5.18 | 0.00 / 527.15 / 912.66 | 0.00 | 183ms | 1.13 | 3.13 | none |

API pool, from `pgxpool.Stat` over the window: acquires that found no idle connection, and the mean acquire wait.

| Row | Empty acquires | Mean acquire | Scheduler worst lag |
| --- | --- | --- | --- |
| create, agent, open loop 100/s | 0 | 0.00 ms | 1.95 ms |
| create, agent, open loop 200/s | 87 | 0.10 ms | 5.43 ms |
| create, agent, gapless type, open loop 100/s | 915 | 54.83 ms | 37.69 ms |

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_logical_emit_message(true, ...); COMMIT` probe, a commit that flushes WAL with no INSERT, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). An attempt is also void if a container other than the run's database, the buildkit builder and the three Multica platform containers runs during it. A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent, open loop 100/s | 1 | large | 6000 | 15.43 | 21.34 | 29.62 | 0.55 | 2.90 | 6.14 | 4 | 0 | 0 | open loop, informational (TICK-700) |
| create, agent, open loop 200/s | 1 | large | 12000 | 20.73 | 33.15 | 47.77 | 0.55 | 2.99 | 7.68 | 4 | 0 | 0 | open loop, informational (TICK-700) |
| create, agent, gapless type, open loop 100/s | 1 | large | 6000 | 17.32 | 648.68 | 971.28 | 1.13 | 3.13 | 5.18 | 4 | 0 | 0 | open loop, informational (TICK-700) |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (0.56 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent, open loop 100/s | 6000 | 1.40 / 2.31 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, open loop 200/s | 12000 | 2.28 / 3.63 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, gapless type, open loop 100/s | 6000 | 3.38 / 89.43 | 5.45 / 9.02 | 1 | 0.56 / 0.56 | 4.63 / 7.96 | 0.00 / 0.71 |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |
| create, agent, open loop 100/s | 6001 | 0.01 / 0.25 / 0.48 | 1.57 / 1.77 / 2.48 | 0 | none |
| create, agent, open loop 200/s | 12002 | 0.02 / 0.25 / 0.48 | 2.46 / 2.20 / 4.74 | 0 | none |
| create, agent, gapless type, open loop 100/s | 0 | - | - | 0 | none |

## Attempt setup and host-probe samples (creates)

Before each create attempt: wait until no container runs other than the run's database, `buildx_buildkit_nava-ci0` and the three `multica-*` platform containers (docker ps is polled every 5 s during the attempt too; another container voids it), then until no autovacuum or analyze backend works in the bench database, then the host-speed probe and the storage probe. `pg_stat_activity` is read at the probe's midpoint (every non-idle backend and every non-client backend, other than the reader); `docker stats` is one sample taken while the probe runs. The seed was quiesced once with `VACUUM (ANALYZE)` and `CHECKPOINT`.

### create, agent, open loop 100/s

Isolation wait 0s, quiesce wait 0s, host probe p50 0.55 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-524 cpu=0.22% mem=8.60%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.09%
  multica-backend-1 cpu=0.38% mem=0.15%
  multica-frontend-1 cpu=0.00% mem=0.62%
  multica-postgres-1 cpu=0.21% mem=0.65%
```

### create, agent, open loop 200/s

Isolation wait 0s, quiesce wait 0s, host probe p50 0.55 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-524 cpu=14.46% mem=8.91%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.09%
  multica-backend-1 cpu=0.23% mem=0.15%
  multica-frontend-1 cpu=0.00% mem=0.62%
  multica-postgres-1 cpu=0.19% mem=0.66%
```

### create, agent, gapless type, open loop 100/s

Isolation wait 0s, quiesce wait 0s, host probe p50 1.13 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-524 cpu=0.03% mem=9.36%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.09%
  multica-backend-1 cpu=0.09% mem=0.15%
  multica-frontend-1 cpu=0.00% mem=0.62%
  multica-postgres-1 cpu=0.09% mem=0.66%
```


## Data-filter statement timeout (TICK-647)

Default timeout 2s (store.DataFilterTimeout); the bound check runs with 2ms. A 422 is `filter_too_expensive`. Latency includes the 422s.

| Scenario | Requests | 422 | p50 | p95 | max |
| --- | --- | --- | --- | --- | --- |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### create, agent, open loop 100/s

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 6000 | 3.37 | 5.23 | 7.26 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 6000 | 1.71 | 2.75 | 3.51 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 6000 | 1.40 | 2.31 | 3.21 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 6000 | 6.14 | 8.17 | 9.97 |

### create, agent, open loop 200/s

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 12000 | 4.53 | 8.05 | 12.81 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 12000 | 2.30 | 3.99 | 5.36 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 12000 | 2.28 | 3.63 | 4.88 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 12000 | 7.68 | 11.49 | 14.49 |

### create, agent, gapless type, open loop 100/s

Round trips: 4, statements: 15, statements per batch: [8 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 6000 | 3.48 | 7.41 | 15.17 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 6000 | 1.77 | 3.69 | 5.64 |
| 3 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 6000 | 3.38 | 89.43 | 240.08 |
| 4 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 6000 | 5.18 | 8.51 | 11.85 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### create, agent, open loop 100/s

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

### create, agent, open loop 200/s

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

### create, agent, gapless type, open loop 100/s

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


## Host (bench/acl-perf/run.sh)

```
### 2026-11-11T04:43:05Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 26260873216 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: buildx_buildkit_nava-ci0 multica-backend-1 multica-frontend-1 multica-postgres-1 
```
