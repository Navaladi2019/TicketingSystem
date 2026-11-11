# P3-36 raw report (TICK-621): VOID, not scored

> **VOID, not scored.** The storage probe in this run, `BEGIN; SELECT pg_current_xact_id(); COMMIT`, did not flush WAL. A commit that only takes a transaction id writes no WAL of its own, so the storage gate measured nothing; the "meets" verdicts below do not count. The scored run is `final-2026-11-11-idle-machine.md`, and the verdict is in `../acl-perf-800k.md` ("Final verdict (P3-36)").

Generated 2026-11-11T02:58:24Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 819309, open 139587, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 413949
record_shares 15815
heap 1067 MB, indexes 779 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.10 / p95 1.74 / p99 2.52 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

A create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= 3 ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= 2.0 ms, run just before the attempt) and the storage window (a dedicated 1-client `BEGIN; SELECT pg_current_xact_id(); COMMIT` probe, p50 <= 10 ms, run just before the attempt; the in-trace commit round trip, which since TICK-692 carries the INSERT, is reported only). An attempt is also void if a container other than the run's database, the buildkit builder and the three Multica platform containers runs during it. A gated attempt is neither a pass nor a fail and is re-run, at most 3 attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.

| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Storage probe p50 | In-trace commit p50 | Round trips | retry_later | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 1 | large | 1080 | 17.68 | 22.47 | 25.41 | 1.08 | 0.62 | 6.92 | 4 | 0 | 50 | meets |
| create, agent | 1 | large | 5069 | 29.67 | 41.17 | 50.65 | 1.04 | 1.00 | 11.43 | 4 | 0 | 50 | meets |
| create, admin | 1 | large | 5224 | 29.47 | 40.55 | 52.66 | 1.13 | 0.54 | 11.52 | 4 | 0 | 50 | meets |
| create, small tenant agent | 1 | small | 5309 | 28.66 | 38.00 | 44.39 | 1.07 | 0.95 | 11.53 | 4 | 0 | 50 | meets |
| create, agent, gapless type | 1 | large | 2421 | 52.66 | 157.48 | 236.95 | 1.09 | 0.55 | 5.71 | 4 | 0 | 0 | gapless type: reported, not scored (D-3-9) |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.10 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client), attempt 1 | 1080 | 1.76 / 2.93 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, attempt 1 | 5069 | 2.93 / 4.97 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, admin, attempt 1 | 5224 | 2.90 / 4.96 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, small tenant agent, attempt 1 | 5309 | 2.89 / 4.70 | 0.00 / 0.00 | 0 | 0.00 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| create, agent, gapless type | 2421 | 36.94 / 140.68 | 5.98 / 7.96 | 1 | 1.10 / 1.10 | 4.61 / 6.38 | 0.00 / 0.67 |

## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)

Per scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.

| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |
| --- | --- | --- | --- | --- | --- |
| create, agent (1 client), attempt 1 | 1130 | 0.02 / 0.25 / 0.48 | 1.97 / 1.86 / 4.12 | 0 | none |
| create, agent, attempt 1 | 5119 | 0.03 / 0.25 / 0.48 | 3.27 / 3.37 / 5.93 | 0 | none |
| create, admin, attempt 1 | 5225 | 0.03 / 0.26 / 0.49 | 3.24 / 3.38 / 5.89 | 0 | none |
| create, small tenant agent, attempt 1 | 5359 | 0.02 / 0.26 / 0.48 | 3.19 / 3.37 / 4.98 | 0 | none |
| create, agent, gapless type | 0 | - | - | 0 | none |

## Attempt setup and host-probe samples (creates)

Before each create attempt: wait until no container runs other than the run's database, `buildx_buildkit_nava-ci0` and the three `multica-*` platform containers (docker ps is polled every 5 s during the attempt too; another container voids it), then until no autovacuum or analyze backend works in the bench database, then the host-speed probe and the storage probe. `pg_stat_activity` is read at the probe's midpoint (every non-idle backend and every non-client backend, other than the reader); `docker stats` is one sample taken while the probe runs. The seed was quiesced once with `VACUUM (ANALYZE)` and `CHECKPOINT`.

### create, agent (1 client), attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 1.08 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1975 cpu=0.03% mem=8.62%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.08%
  multica-backend-1 cpu=0.49% mem=0.14%
  multica-frontend-1 cpu=0.12% mem=0.60%
  multica-postgres-1 cpu=4.40% mem=0.88%
```

### create, agent, attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 1.04 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1975 cpu=0.02% mem=8.67%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.08%
  multica-backend-1 cpu=0.39% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=0.36% mem=0.88%
```

### create, admin, attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 1.13 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1975 cpu=0.03% mem=8.92%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.08%
  multica-backend-1 cpu=0.61% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=5.70% mem=0.88%
```

### create, small tenant agent, attempt 1

Isolation wait 0s, quiesce wait 0s, host probe p50 1.07 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1975 cpu=0.04% mem=9.07%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.08%
  multica-backend-1 cpu=0.68% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=1.07% mem=0.88%
```

### create, agent, gapless type

Isolation wait 0s, quiesce wait 0s, host probe p50 1.09 ms.

```
pg_stat_activity:
  autovacuum launcher db=- state=- running=- query=
  background writer db=- state=- running=- query=
  checkpointer db=- state=- running=- query=
  logical replication launcher db=- state=- running=- query=
  walwriter db=- state=- running=- query=
docker stats:
  nava-aclperf-pg-1975 cpu=0.03% mem=9.27%
  buildx_buildkit_nava-ci0 cpu=0.00% mem=0.08%
  multica-backend-1 cpu=0.27% mem=0.14%
  multica-frontend-1 cpu=0.00% mem=0.60%
  multica-postgres-1 cpu=0.18% mem=0.88%
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
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 1080 | 3.99 | 5.93 | 7.31 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 1080 | 2.22 | 3.44 | 4.18 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 1080 | 1.76 | 2.93 | 3.54 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 1080 | 6.92 | 8.87 | 10.50 |

### create, agent, attempt 1

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 5069 | 6.52 | 12.27 | 17.68 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 5069 | 3.15 | 5.24 | 7.49 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 5069 | 2.93 | 4.97 | 6.57 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 5069 | 11.43 | 15.09 | 18.37 |

### create, admin, attempt 1

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 5224 | 6.42 | 11.82 | 16.59 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 5224 | 3.23 | 5.24 | 6.87 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 5224 | 2.90 | 4.96 | 6.63 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 5224 | 11.52 | 15.44 | 18.24 |

### create, small tenant agent, attempt 1

Round trips: 4, statements: 19, statements per batch: [8 2 5 4] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 5309 | 6.32 | 10.69 | 14.09 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 5309 | 3.21 | 5.04 | 6.55 |
| 3 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SET LOCAL synchronous_commit…` | 5309 | 2.89 | 4.70 | 6.11 |
| 4 | `BATCH: WITH ins AS ( INSERT INTO records (tenant_id, id, version, entity_definition_id, sc…` | 5309 | 11.53 | 15.23 | 17.53 |

### create, agent, gapless type

Round trips: 4, statements: 15, statements per batch: [8 2 3 2] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BATCH: BEGIN ; SELECT set_config('app.tenant_id', $1, true) ; SELECT set_config('lock_time…` | 2421 | 3.99 | 5.92 | 8.10 |
| 2 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2421 | 2.79 | 4.33 | 5.18 |
| 3 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 2421 | 36.94 | 140.68 | 217.87 |
| 4 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2421 | 5.71 | 7.48 | 9.13 |

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

