### 2026-11-09T11:53:24Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 20806184960 bytes, kernel 5.15.167.4-microsoft-standard-WSL2

# P3-36 raw report (TICK-621)

Generated 2026-11-09T11:59:24Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 802315, open 122593, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 401513
record_shares 15815
heap 1045 MB, indexes 651 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.65 / p95 2.76 / p99 3.93 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| single GET, agent | large | 4506 | 34.05 | 47.58 | 58.87 | 50 | meets |
| single GET, admin | large | 4376 | 34.43 | 48.75 | 75.73 | 50 | meets |
| single GET, small tenant agent | small | 4527 | 31.32 | 55.28 | 115.09 | 50 | **MISS** |
| create, agent (1 client) | large | 271 | 71.41 | 98.59 | 136.00 | 50 | **MISS** |
| create, agent | large | 630 | 183.15 | 647.84 | 1197.48 | 50 | **MISS** |
| create, admin | large | 508 | 220.03 | 858.41 | 1426.51 | 50 | **MISS** |
| create, small tenant agent | small | 751 | 189.92 | 370.86 | 650.94 | 50 | **MISS** |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.65 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 271 | 1.99 / 3.17 | 29.16 / 42.40 | 11 | 18.11 / 18.11 | 9.50 / 22.89 | 1.16 / 2.53 |
| create, agent | 630 | 108.60 / 580.79 | 30.59 / 41.61 | 11 | 18.11 / 18.11 | 11.14 / 21.72 | 1.18 / 2.48 |
| create, admin | 508 | 127.32 / 771.72 | 35.49 / 70.48 | 11 | 18.11 / 18.11 | 15.76 / 50.42 | 1.31 / 2.98 |
| create, small tenant agent | 751 | 4.62 / 12.75 | 78.68 / 154.89 | 11 | 18.11 / 18.11 | 57.67 / 132.08 | 2.32 / 5.72 |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### single GET, agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `begin` | 4506 | 1.86 | 3.35 | 4.99 |
| 2 | `SELECT set_config('app.tenant_id', $1, true)` | 4506 | 2.11 | 3.61 | 5.04 |
| 3 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 4506 | 2.96 | 4.69 | 6.65 |
| 4 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 4506 | 3.47 | 5.43 | 7.34 |
| 5 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22530 | 2.12 | 3.60 | 5.19 |
| 6 | `SELECT coalesce(array_agg(principal_id ORDER BY principal_id), '{}') FROM record_shares WH…` | 4506 | 2.58 | 4.12 | 5.75 |
| 7 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22530 | 2.12 | 3.60 | 5.19 |
| 8 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22530 | 2.12 | 3.60 | 5.19 |
| 9 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 9012 | 2.71 | 4.29 | 6.09 |
| 10 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22530 | 2.12 | 3.60 | 5.19 |
| 11 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22530 | 2.12 | 3.60 | 5.19 |
| 12 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 9012 | 2.71 | 4.29 | 6.09 |
| 13 | `commit` | 4506 | 1.96 | 3.52 | 4.94 |

### single GET, admin

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `begin` | 4376 | 1.90 | 3.62 | 6.18 |
| 2 | `SELECT set_config('app.tenant_id', $1, true)` | 4376 | 2.19 | 3.78 | 7.10 |
| 3 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 4376 | 2.96 | 4.79 | 7.96 |
| 4 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 4376 | 3.50 | 5.51 | 8.11 |
| 5 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 21880 | 2.17 | 3.80 | 6.58 |
| 6 | `SELECT coalesce(array_agg(principal_id ORDER BY principal_id), '{}') FROM record_shares WH…` | 4376 | 2.56 | 4.44 | 7.52 |
| 7 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 21880 | 2.17 | 3.80 | 6.58 |
| 8 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 21880 | 2.17 | 3.80 | 6.58 |
| 9 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 8752 | 2.73 | 4.50 | 7.06 |
| 10 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 21880 | 2.17 | 3.80 | 6.58 |
| 11 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 21880 | 2.17 | 3.80 | 6.58 |
| 12 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 8752 | 2.73 | 4.50 | 7.06 |
| 13 | `commit` | 4376 | 2.03 | 3.82 | 7.39 |

### single GET, small tenant agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `begin` | 4527 | 1.79 | 3.54 | 7.58 |
| 2 | `SELECT set_config('app.tenant_id', $1, true)` | 4527 | 1.89 | 3.65 | 7.73 |
| 3 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 4527 | 2.80 | 5.06 | 10.31 |
| 4 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 4527 | 3.15 | 5.61 | 12.68 |
| 5 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22635 | 1.90 | 3.80 | 8.59 |
| 6 | `SELECT coalesce(array_agg(principal_id ORDER BY principal_id), '{}') FROM record_shares WH…` | 4527 | 2.39 | 4.50 | 8.97 |
| 7 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22635 | 1.90 | 3.80 | 8.59 |
| 8 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22635 | 1.90 | 3.80 | 8.59 |
| 9 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 9054 | 2.44 | 4.69 | 9.58 |
| 10 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22635 | 1.90 | 3.80 | 8.59 |
| 11 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 22635 | 1.90 | 3.80 | 8.59 |
| 12 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 9054 | 2.44 | 4.69 | 9.58 |
| 13 | `commit` | 4527 | 1.83 | 3.76 | 8.89 |

### create, agent (1 client)

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `begin` | 542 | 1.75 | 2.65 | 3.22 |
| 2 | `SELECT set_config('app.tenant_id', $1, true)` | 542 | 1.76 | 2.86 | 4.28 |
| 3 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 271 | 2.15 | 3.16 | 3.97 |
| 4 | `commit` | 542 | 4.72 | 6.73 | 10.71 |
| 5 | `begin` | 542 | 1.75 | 2.65 | 3.22 |
| 6 | `SELECT set_config('app.tenant_id', $1, true)` | 542 | 1.76 | 2.86 | 4.28 |
| 7 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 542 | 2.42 | 3.69 | 4.96 |
| 8 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 9 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 271 | 1.97 | 2.96 | 4.00 |
| 10 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 542 | 2.42 | 3.69 | 4.96 |
| 11 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 12 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 13 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 813 | 2.30 | 3.53 | 4.62 |
| 14 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 15 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 16 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 813 | 2.30 | 3.53 | 4.62 |
| 17 | `SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NUL…` | 271 | 2.22 | 3.28 | 4.45 |
| 18 | `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON …` | 271 | 1.99 | 3.17 | 5.95 |
| 19 | `INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number,…` | 271 | 2.71 | 4.27 | 8.23 |
| 20 | `SELECT ((data - $4::text[]) \|\| CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_b…` | 271 | 2.33 | 3.64 | 5.39 |
| 21 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 22 | `INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_de…` | 271 | 2.71 | 4.18 | 5.70 |
| 23 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 271 | 2.96 | 5.04 | 9.15 |
| 24 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 25 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 26 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 813 | 2.30 | 3.53 | 4.62 |
| 27 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 2439 | 1.76 | 2.92 | 4.16 |
| 28 | `UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5,…` | 271 | 2.26 | 3.60 | 4.35 |
| 29 | `commit` | 542 | 4.72 | 6.73 | 10.71 |

### create, agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `begin` | 1260 | 1.74 | 2.90 | 3.69 |
| 2 | `SELECT set_config('app.tenant_id', $1, true)` | 1260 | 1.74 | 2.99 | 4.12 |
| 3 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 630 | 2.29 | 3.64 | 5.62 |
| 4 | `commit` | 1260 | 5.10 | 8.17 | 10.88 |
| 5 | `begin` | 1260 | 1.74 | 2.90 | 3.69 |
| 6 | `SELECT set_config('app.tenant_id', $1, true)` | 1260 | 1.74 | 2.99 | 4.12 |
| 7 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1260 | 2.40 | 3.80 | 5.19 |
| 8 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 9 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 630 | 1.88 | 3.42 | 5.01 |
| 10 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1260 | 2.40 | 3.80 | 5.19 |
| 11 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 12 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 13 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1890 | 2.34 | 3.95 | 5.76 |
| 14 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 15 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 16 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1890 | 2.34 | 3.95 | 5.76 |
| 17 | `SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NUL…` | 630 | 1.90 | 3.48 | 4.81 |
| 18 | `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON …` | 630 | 108.60 | 580.79 | 1105.71 |
| 19 | `INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number,…` | 630 | 2.83 | 4.35 | 6.96 |
| 20 | `SELECT ((data - $4::text[]) \|\| CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_b…` | 630 | 2.29 | 3.57 | 6.19 |
| 21 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 22 | `INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_de…` | 630 | 2.87 | 4.56 | 6.41 |
| 23 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 630 | 2.92 | 4.41 | 5.81 |
| 24 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 25 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 26 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1890 | 2.34 | 3.95 | 5.76 |
| 27 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 5670 | 1.78 | 3.13 | 4.55 |
| 28 | `UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5,…` | 630 | 2.35 | 3.99 | 5.42 |
| 29 | `commit` | 1260 | 5.10 | 8.17 | 10.88 |

### create, admin

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `begin` | 1016 | 1.92 | 4.36 | 9.83 |
| 2 | `SELECT set_config('app.tenant_id', $1, true)` | 1016 | 1.94 | 4.34 | 9.40 |
| 3 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 508 | 2.43 | 5.11 | 11.23 |
| 4 | `commit` | 1016 | 5.82 | 11.10 | 17.14 |
| 5 | `begin` | 1016 | 1.92 | 4.36 | 9.83 |
| 6 | `SELECT set_config('app.tenant_id', $1, true)` | 1016 | 1.94 | 4.34 | 9.40 |
| 7 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1016 | 2.95 | 5.62 | 11.19 |
| 8 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 9 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 508 | 2.37 | 4.81 | 9.86 |
| 10 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1016 | 2.95 | 5.62 | 11.19 |
| 11 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 12 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 13 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1524 | 2.72 | 5.41 | 10.75 |
| 14 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 15 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 16 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1524 | 2.72 | 5.41 | 10.75 |
| 17 | `SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NUL…` | 508 | 2.36 | 5.09 | 10.02 |
| 18 | `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON …` | 508 | 127.32 | 771.72 | 1314.57 |
| 19 | `INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number,…` | 508 | 3.07 | 7.47 | 13.02 |
| 20 | `SELECT ((data - $4::text[]) \|\| CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_b…` | 508 | 2.56 | 5.90 | 9.24 |
| 21 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 22 | `INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_de…` | 508 | 3.47 | 6.42 | 10.38 |
| 23 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 508 | 3.50 | 6.67 | 10.67 |
| 24 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 25 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 26 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1524 | 2.72 | 5.41 | 10.75 |
| 27 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 4572 | 2.22 | 4.61 | 9.99 |
| 28 | `UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5,…` | 508 | 2.88 | 5.93 | 9.35 |
| 29 | `commit` | 1016 | 5.82 | 11.10 | 17.14 |

### create, small tenant agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `begin` | 1502 | 4.21 | 12.31 | 19.78 |
| 2 | `SELECT set_config('app.tenant_id', $1, true)` | 1502 | 4.27 | 11.70 | 20.05 |
| 3 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 751 | 4.90 | 13.36 | 19.22 |
| 4 | `commit` | 1502 | 11.81 | 47.50 | 191.42 |
| 5 | `begin` | 1502 | 4.21 | 12.31 | 19.78 |
| 6 | `SELECT set_config('app.tenant_id', $1, true)` | 1502 | 4.27 | 11.70 | 20.05 |
| 7 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1502 | 5.67 | 13.77 | 20.27 |
| 8 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 9 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 751 | 4.93 | 14.05 | 18.11 |
| 10 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1502 | 5.67 | 13.77 | 20.27 |
| 11 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 12 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 13 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2253 | 5.01 | 14.92 | 23.67 |
| 14 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 15 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 16 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2253 | 5.01 | 14.92 | 23.67 |
| 17 | `SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NUL…` | 751 | 4.57 | 13.97 | 24.10 |
| 18 | `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON …` | 751 | 4.62 | 12.75 | 21.22 |
| 19 | `INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number,…` | 751 | 6.05 | 16.57 | 25.64 |
| 20 | `SELECT ((data - $4::text[]) \|\| CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_b…` | 751 | 5.16 | 14.34 | 24.94 |
| 21 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 22 | `INSERT INTO audit_log ( tenant_id, id, domain, action, subject_type, subject_id, entity_de…` | 751 | 6.11 | 15.32 | 20.74 |
| 23 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 751 | 6.59 | 17.20 | 23.64 |
| 24 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 25 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 26 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2253 | 5.01 | 14.92 | 23.67 |
| 27 | `SELECT coalesce(current_setting('app.tenant_id', true), '')` | 6759 | 4.31 | 12.89 | 20.91 |
| 28 | `UPDATE idempotency_keys SET state = 'completed', response_status = $4, response_body = $5,…` | 751 | 5.10 | 14.75 | 20.37 |
| 29 | `commit` | 1502 | 11.81 | 47.50 | 191.42 |

## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)

One request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.

### single GET, agent

Request: `GET /api/v1/records/ticket/3f94a2ac-38ee-af8f-f434-143a689a293d` as `agent-1`

Statement 3:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.040..0.043 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.025..0.027 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.356 ms
Execution Time: 0.079 ms
```

Statement 4:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.id = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","3f94a2ac-38ee-af8f-f434-143a689a293d"]`

```
Result  (cost=1.00..25.08 rows=1 width=1261) (actual time=0.213..0.219 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=10
  ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.200..0.205 rows=1 loops=1)
        Buffers: shared hit=10
        ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.166..0.170 rows=1 loops=1)
              Join Filter: (r.entity_definition_id = d.id)
              Buffers: shared hit=7
              ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.135..0.137 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '3f94a2ac-38ee-af8f-f434-143a689a293d'::uuid))
                    Buffers: shared hit=4
              ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.026..0.026 rows=1 loops=1)
                    Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                    Buffers: shared hit=3
        ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.030..0.030 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=3
              ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.021..0.021 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = r.assignment_group_id))
                    Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.607 ms
Execution Time: 0.320 ms
```

Statement 5:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.005..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.019 ms
Execution Time: 0.017 ms
```

Statement 6:

```sql
SELECT coalesce(array_agg(principal_id ORDER BY principal_id), '{}') FROM record_shares
WHERE tenant_id = $1 AND record_id = $2
  AND ((principal_kind = 'user' AND principal_id = $3::uuid)
    OR (principal_kind = 'group' AND principal_id = ANY($4::uuid[])))
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","3f94a2ac-38ee-af8f-f434-143a689a293d","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=4.34..4.35 rows=1 width=32) (actual time=0.039..0.041 rows=1 loops=1)
  Buffers: shared hit=2
  ->  Sort  (cost=4.33..4.33 rows=1 width=16) (actual time=0.036..0.036 rows=0 loops=1)
        Sort Key: principal_id
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=2
        ->  Result  (cost=0.29..4.32 rows=1 width=16) (actual time=0.026..0.027 rows=0 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=2
              ->  Index Only Scan using record_shares_pkey on record_shares  (cost=0.29..4.32 rows=1 width=16) (actual time=0.020..0.020 rows=0 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (record_id = '3f94a2ac-38ee-af8f-f434-143a689a293d'::uuid))
                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    Heap Fetches: 0
                    Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.188 ms
Execution Time: 0.106 ms
```

Statement 7:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.019 ms
Execution Time: 0.016 ms
```

Statement 8:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.005..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.018 ms
Execution Time: 0.015 ms
```

Statement 9:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.053..0.055 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.052..0.053 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.042..0.045 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.034..0.037 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.023..0.024 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.275 ms
Execution Time: 0.087 ms
```

Statement 10:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.005..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.020 ms
Execution Time: 0.017 ms
```

Statement 11:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.004..0.004 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.016 ms
Execution Time: 0.013 ms
```

Statement 12:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.056..0.057 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.054..0.056 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.044..0.048 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.036..0.039 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.025..0.026 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.248 ms
Execution Time: 0.092 ms
```

### single GET, admin

Request: `GET /api/v1/records/ticket/7af47606-284d-7369-c4d9-ac496c0f82c8` as `admin-large`

Statement 3:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.041..0.044 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.026..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.327 ms
Execution Time: 0.078 ms
```

Statement 4:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.id = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","7af47606-284d-7369-c4d9-ac496c0f82c8"]`

```
Result  (cost=1.00..25.08 rows=1 width=1261) (actual time=0.085..0.089 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=10
  ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.073..0.077 rows=1 loops=1)
        Buffers: shared hit=10
        ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.052..0.055 rows=1 loops=1)
              Join Filter: (r.entity_definition_id = d.id)
              Buffers: shared hit=7
              ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.036..0.038 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '7af47606-284d-7369-c4d9-ac496c0f82c8'::uuid))
                    Buffers: shared hit=4
              ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.013..0.013 rows=1 loops=1)
                    Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                    Buffers: shared hit=3
        ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.019..0.020 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=3
              ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.016..0.016 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = r.assignment_group_id))
                    Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.507 ms
Execution Time: 0.232 ms
```

Statement 5:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.005..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.020 ms
Execution Time: 0.019 ms
```

Statement 6:

```sql
SELECT coalesce(array_agg(principal_id ORDER BY principal_id), '{}') FROM record_shares
WHERE tenant_id = $1 AND record_id = $2
  AND ((principal_kind = 'user' AND principal_id = $3::uuid)
    OR (principal_kind = 'group' AND principal_id = ANY($4::uuid[])))
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","7af47606-284d-7369-c4d9-ac496c0f82c8","08a17a31-85a1-47aa-8cbc-2b2737b7729e",["a7cb0945-9b15-4403-91ef-c5cd8e2ec31d"]]`

```
Aggregate  (cost=4.33..4.34 rows=1 width=32) (actual time=0.032..0.033 rows=1 loops=1)
  Buffers: shared hit=2
  ->  Sort  (cost=4.32..4.33 rows=1 width=16) (actual time=0.029..0.029 rows=0 loops=1)
        Sort Key: principal_id
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=2
        ->  Result  (cost=0.29..4.31 rows=1 width=16) (actual time=0.023..0.024 rows=0 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=2
              ->  Index Only Scan using record_shares_pkey on record_shares  (cost=0.29..4.31 rows=1 width=16) (actual time=0.017..0.017 rows=0 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (record_id = '7af47606-284d-7369-c4d9-ac496c0f82c8'::uuid))
                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '08a17a31-85a1-47aa-8cbc-2b2737b7729e'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{a7cb0945-9b15-4403-91ef-c5cd8e2ec31d}'::uuid[]))))
                    Heap Fetches: 0
                    Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.186 ms
Execution Time: 0.069 ms
```

Statement 7:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.004..0.005 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.017 ms
Execution Time: 0.014 ms
```

Statement 8:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.031 ms
Execution Time: 0.027 ms
```

Statement 9:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.049..0.051 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.048..0.049 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.039..0.043 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.034..0.037 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.023..0.024 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.254 ms
Execution Time: 0.081 ms
```

Statement 10:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.007 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.028 ms
Execution Time: 0.024 ms
```

Statement 11:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.022 ms
Execution Time: 0.019 ms
```

Statement 12:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.043..0.045 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.042..0.043 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.035..0.037 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.029..0.031 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.018..0.019 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.008 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.222 ms
Execution Time: 0.073 ms
```

### single GET, small tenant agent

Request: `GET /api/v1/records/ticket/c7df223c-cbb4-f467-776f-6fea3db8bf06` as `agent-s1`

Statement 3:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.122..0.127 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.100..0.103 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 13.595 ms
Execution Time: 0.208 ms
```

Statement 4:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.id = $2
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","c7df223c-cbb4-f467-776f-6fea3db8bf06"]`

```
Result  (cost=1.00..25.08 rows=1 width=1261) (actual time=1.404..1.414 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
  Buffers: shared hit=10
  ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=1.380..1.388 rows=1 loops=1)
        Buffers: shared hit=10
        ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=1.322..1.328 rows=1 loops=1)
              Join Filter: (r.entity_definition_id = d.id)
              Buffers: shared hit=7
              ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=1.273..1.277 rows=1 loops=1)
                    Index Cond: ((tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid) AND (id = 'c7df223c-cbb4-f467-776f-6fea3db8bf06'::uuid))
                    Buffers: shared hit=4
              ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.035..0.035 rows=1 loops=1)
                    Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                    Buffers: shared hit=3
        ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.050..0.050 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=3
              ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.026..0.026 rows=1 loops=1)
                    Index Cond: ((tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid) AND (id = r.assignment_group_id))
                    Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.004 ms
Execution Time: 1.546 ms
```

Statement 5:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.009..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.042 ms
Execution Time: 0.055 ms
```

Statement 6:

```sql
SELECT coalesce(array_agg(principal_id ORDER BY principal_id), '{}') FROM record_shares
WHERE tenant_id = $1 AND record_id = $2
  AND ((principal_kind = 'user' AND principal_id = $3::uuid)
    OR (principal_kind = 'group' AND principal_id = ANY($4::uuid[])))
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","c7df223c-cbb4-f467-776f-6fea3db8bf06","4e83828a-ef25-9e19-bf97-84874ff6db98",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"]]`

```
Aggregate  (cost=4.34..4.35 rows=1 width=32) (actual time=0.071..0.073 rows=1 loops=1)
  Buffers: shared hit=2
  ->  Sort  (cost=4.33..4.33 rows=1 width=16) (actual time=0.064..0.065 rows=0 loops=1)
        Sort Key: principal_id
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=2
        ->  Result  (cost=0.29..4.32 rows=1 width=16) (actual time=0.051..0.052 rows=0 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=2
              ->  Index Only Scan using record_shares_pkey on record_shares  (cost=0.29..4.32 rows=1 width=16) (actual time=0.036..0.037 rows=0 loops=1)
                    Index Cond: ((tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid) AND (record_id = 'c7df223c-cbb4-f467-776f-6fea3db8bf06'::uuid))
                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                    Heap Fetches: 0
                    Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.359 ms
Execution Time: 0.155 ms
```

Statement 7:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.011..0.012 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.042 ms
Execution Time: 0.035 ms
```

Statement 8:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.015..0.016 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.068 ms
Execution Time: 2.783 ms
```

Statement 9:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.095..0.098 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.093..0.095 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.077..0.083 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.062..0.067 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.044..0.046 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.013..0.015 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.487 ms
Execution Time: 0.160 ms
```

Statement 10:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.011..0.012 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.044 ms
Execution Time: 0.036 ms
```

Statement 11:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.009..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.043 ms
Execution Time: 0.033 ms
```

Statement 12:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.089..0.092 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.087..0.088 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.069..0.076 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.054..0.060 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.037..0.039 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.012..0.015 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.486 ms
Execution Time: 0.160 ms
```

### create, agent (1 client)

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
begin
```

Statement 3 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 4 (write, not EXPLAINed):

```sql
commit
```

Statement 5 (write, not EXPLAINed):

```sql
begin
```

Statement 7:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.042..0.045 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.027..0.029 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.340 ms
Execution Time: 0.081 ms
```

Statement 8:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.007..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.028 ms
Execution Time: 0.023 ms
```

Statement 9:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.049..0.052 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.036..0.038 rows=1 loops=1)
        Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.197 ms
Execution Time: 0.089 ms
```

Statement 10:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.052..0.055 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.033..0.036 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.503 ms
Execution Time: 0.106 ms
```

Statement 11:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.034 ms
Execution Time: 0.028 ms
```

Statement 12:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.026 ms
Execution Time: 0.023 ms
```

Statement 13:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.078..0.080 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.076..0.078 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.063..0.068 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.052..0.057 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.036..0.037 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.012..0.014 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.385 ms
Execution Time: 0.128 ms
```

Statement 14:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.011..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.039 ms
Execution Time: 0.105 ms
```

Statement 15:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.022 ms
Execution Time: 0.018 ms
```

Statement 16:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.061..0.062 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.059..0.060 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.049..0.053 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.040..0.044 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.009..0.011 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.290 ms
Execution Time: 0.099 ms
```

Statement 17:

```sql
SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL)
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","969c31f2-d8fe-635a-5a2d-95d39e4e7489"]`

```
Result  (cost=8.31..8.32 rows=1 width=1) (actual time=0.130..0.133 rows=1 loops=1)
  Buffers: shared hit=3
  InitPlan 1 (returns $0)
    ->  Result  (cost=0.29..8.31 rows=1 width=0) (actual time=0.126..0.127 rows=1 loops=1)
          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
          Buffers: shared hit=3
          ->  Index Scan using groups_live_key on groups  (cost=0.29..8.31 rows=1 width=0) (actual time=0.111..0.112 rows=1 loops=1)
                Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '969c31f2-d8fe-635a-5a2d-95d39e4e7489'::uuid))
                Filter: (deleted_at IS NULL)
                Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.401 ms
Execution Time: 0.183 ms
```

Statement 18 (write, not EXPLAINed):

```sql
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1
```

Statement 19 (write, not EXPLAINed):

```sql
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data,
    causation_origin_id, causation_depth)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15)
```

Statement 20:

```sql
SELECT ((data - $4::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($5::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($6::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, requester_id) END || jsonb_build_object($8::text, state))::text FROM records WHERE tenant_id = $1 AND id = $2 AND version = $3
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","01a1c02c-4ece-799b-9eae-f6aaf9735b38",1,["assignee","group","requester","state"],"assignee","group","requester","state"]`

```
Result  (cost=0.42..8.47 rows=1 width=32) (actual time=0.103..0.107 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=4
  ->  Index Scan using records_pkey on records  (cost=0.42..8.45 rows=1 width=1130) (actual time=0.040..0.042 rows=1 loops=1)
        Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '01a1c02c-4ece-799b-9eae-f6aaf9735b38'::uuid))
        Filter: (version = '1'::bigint)
        Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.263 ms
Execution Time: 0.143 ms
```

Statement 21:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.007..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.028 ms
Execution Time: 0.019 ms
```

Statement 22 (write, not EXPLAINed):

```sql
INSERT INTO audit_log (
	tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
	version_before, version_after, actor_principal_id, actor_system, source, source_ref,
	change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256,
	diff_truncated, diff, diff_salt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 23:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.id = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","01a1c02c-4ece-799b-9eae-f6aaf9735b38"]`

```
Result  (cost=1.00..25.08 rows=1 width=1261) (actual time=0.100..0.104 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=10
  ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.087..0.090 rows=1 loops=1)
        Buffers: shared hit=10
        ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.067..0.069 rows=1 loops=1)
              Join Filter: (r.entity_definition_id = d.id)
              Buffers: shared hit=7
              ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.050..0.052 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '01a1c02c-4ece-799b-9eae-f6aaf9735b38'::uuid))
                    Buffers: shared hit=4
              ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.013..0.013 rows=1 loops=1)
                    Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                    Buffers: shared hit=3
        ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.018..0.018 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=3
              ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = r.assignment_group_id))
                    Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.569 ms
Execution Time: 0.183 ms
```

Statement 24:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.021 ms
Execution Time: 0.017 ms
```

Statement 25:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.023 ms
Execution Time: 0.018 ms
```

Statement 26:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.081..0.083 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.079..0.081 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.069..0.072 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.039..0.042 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.028..0.029 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.265 ms
Execution Time: 0.119 ms
```

Statement 27:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.044 ms
Execution Time: 0.061 ms
```

Statement 28 (write, not EXPLAINed):

```sql
UPDATE idempotency_keys
   SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'
```

Statement 29 (write, not EXPLAINed):

```sql
commit
```

### create, agent

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
begin
```

Statement 3 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 4 (write, not EXPLAINed):

```sql
commit
```

Statement 5 (write, not EXPLAINed):

```sql
begin
```

Statement 7:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.055..0.059 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.038..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.009..0.009 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.422 ms
Execution Time: 0.111 ms
```

Statement 8:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.026 ms
Execution Time: 0.022 ms
```

Statement 9:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.038..0.041 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.028..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.166 ms
Execution Time: 0.068 ms
```

Statement 10:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.052..0.055 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.033..0.036 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.404 ms
Execution Time: 0.098 ms
```

Statement 11:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.030 ms
Execution Time: 0.023 ms
```

Statement 12:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.007 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.026 ms
Execution Time: 0.019 ms
```

Statement 13:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.056..0.058 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.054..0.056 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.044..0.047 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.035..0.038 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.024..0.025 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.262 ms
Execution Time: 0.092 ms
```

Statement 14:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.007 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.024 ms
Execution Time: 0.020 ms
```

Statement 15:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.020 ms
Execution Time: 0.017 ms
```

Statement 16:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.053..0.055 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.052..0.053 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.043..0.047 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.036..0.039 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.007..0.009 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.251 ms
Execution Time: 0.086 ms
```

Statement 17:

```sql
SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL)
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","969c31f2-d8fe-635a-5a2d-95d39e4e7489"]`

```
Result  (cost=8.31..8.32 rows=1 width=1) (actual time=0.038..0.040 rows=1 loops=1)
  Buffers: shared hit=3
  InitPlan 1 (returns $0)
    ->  Result  (cost=0.29..8.31 rows=1 width=0) (actual time=0.035..0.037 rows=1 loops=1)
          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
          Buffers: shared hit=3
          ->  Index Scan using groups_live_key on groups  (cost=0.29..8.31 rows=1 width=0) (actual time=0.027..0.028 rows=1 loops=1)
                Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '969c31f2-d8fe-635a-5a2d-95d39e4e7489'::uuid))
                Filter: (deleted_at IS NULL)
                Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.153 ms
Execution Time: 0.064 ms
```

Statement 18 (write, not EXPLAINed):

```sql
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1
```

Statement 19 (write, not EXPLAINed):

```sql
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data,
    causation_origin_id, causation_depth)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15)
```

Statement 20:

```sql
SELECT ((data - $4::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($5::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($6::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, requester_id) END || jsonb_build_object($8::text, state))::text FROM records WHERE tenant_id = $1 AND id = $2 AND version = $3
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","01a1c02c-af7e-72c0-950a-4cca82cc80c8",1,["assignee","group","requester","state"],"assignee","group","requester","state"]`

```
Result  (cost=0.42..8.47 rows=1 width=32) (actual time=0.093..0.096 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=4
  ->  Index Scan using records_pkey on records  (cost=0.42..8.45 rows=1 width=1130) (actual time=0.035..0.037 rows=1 loops=1)
        Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '01a1c02c-af7e-72c0-950a-4cca82cc80c8'::uuid))
        Filter: (version = '1'::bigint)
        Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.179 ms
Execution Time: 0.125 ms
```

Statement 21:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.006..0.007 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.024 ms
Execution Time: 0.019 ms
```

Statement 22 (write, not EXPLAINed):

```sql
INSERT INTO audit_log (
	tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
	version_before, version_after, actor_principal_id, actor_system, source, source_ref,
	change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256,
	diff_truncated, diff, diff_salt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 23:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.id = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","01a1c02c-af7e-72c0-950a-4cca82cc80c8"]`

```
Result  (cost=1.00..25.08 rows=1 width=1261) (actual time=0.117..0.122 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=10
  ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.102..0.106 rows=1 loops=1)
        Buffers: shared hit=10
        ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.077..0.080 rows=1 loops=1)
              Join Filter: (r.entity_definition_id = d.id)
              Buffers: shared hit=7
              ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.057..0.059 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '01a1c02c-af7e-72c0-950a-4cca82cc80c8'::uuid))
                    Buffers: shared hit=4
              ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.016..0.016 rows=1 loops=1)
                    Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                    Buffers: shared hit=3
        ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.022..0.022 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=3
              ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.017..0.017 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = r.assignment_group_id))
                    Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.833 ms
Execution Time: 0.253 ms
```

Statement 24:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.028 ms
Execution Time: 0.024 ms
```

Statement 25:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.007..0.008 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.027 ms
Execution Time: 0.021 ms
```

Statement 26:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.055..0.057 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.054..0.055 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.044..0.047 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.037..0.040 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.025..0.026 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.265 ms
Execution Time: 0.090 ms
```

Statement 27:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.027 ms
Execution Time: 0.021 ms
```

Statement 28 (write, not EXPLAINed):

```sql
UPDATE idempotency_keys
   SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'
```

Statement 29 (write, not EXPLAINed):

```sql
commit
```

### create, admin

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
begin
```

Statement 3 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 4 (write, not EXPLAINed):

```sql
commit
```

Statement 5 (write, not EXPLAINed):

```sql
begin
```

Statement 7:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.062..0.066 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.041..0.044 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.576 ms
Execution Time: 0.131 ms
```

Statement 8:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.044 ms
Execution Time: 0.034 ms
```

Statement 9:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.056..0.060 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.040..0.043 rows=1 loops=1)
        Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.198 ms
Execution Time: 0.104 ms
```

Statement 10:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.073..0.078 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.051..0.054 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.012..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.614 ms
Execution Time: 0.156 ms
```

Statement 11:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.011..0.012 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.047 ms
Execution Time: 0.039 ms
```

Statement 12:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.034 ms
Execution Time: 0.030 ms
```

Statement 13:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.086..0.090 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.084..0.086 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.070..0.075 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.054..0.058 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.038..0.039 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.011..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.437 ms
Execution Time: 0.155 ms
```

Statement 14:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.044 ms
Execution Time: 0.033 ms
```

Statement 15:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.035 ms
Execution Time: 0.030 ms
```

Statement 16:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.074..0.077 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.072..0.075 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.061..0.066 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.047..0.052 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.032..0.034 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.010..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.819 ms
Execution Time: 0.135 ms
```

Statement 17:

```sql
SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL)
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","d046a327-d56e-7709-6d29-0e329b19239c"]`

```
Result  (cost=8.31..8.32 rows=1 width=1) (actual time=0.080..0.083 rows=1 loops=1)
  Buffers: shared hit=3
  InitPlan 1 (returns $0)
    ->  Result  (cost=0.29..8.31 rows=1 width=0) (actual time=0.074..0.075 rows=1 loops=1)
          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
          Buffers: shared hit=3
          ->  Index Scan using groups_live_key on groups  (cost=0.29..8.31 rows=1 width=0) (actual time=0.055..0.056 rows=1 loops=1)
                Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = 'd046a327-d56e-7709-6d29-0e329b19239c'::uuid))
                Filter: (deleted_at IS NULL)
                Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.340 ms
Execution Time: 0.152 ms
```

Statement 18 (write, not EXPLAINed):

```sql
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1
```

Statement 19 (write, not EXPLAINed):

```sql
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data,
    causation_origin_id, causation_depth)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15)
```

Statement 20:

```sql
SELECT ((data - $4::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($5::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($6::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, requester_id) END || jsonb_build_object($8::text, state))::text FROM records WHERE tenant_id = $1 AND id = $2 AND version = $3
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","01a1c02d-0073-726b-a47b-9c2fc6bc120c",1,["assignee","group","requester","state"],"assignee","group","requester","state"]`

```
Result  (cost=0.42..8.47 rows=1 width=32) (actual time=0.116..0.120 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=4
  ->  Index Scan using records_pkey on records  (cost=0.42..8.45 rows=1 width=1130) (actual time=0.044..0.047 rows=1 loops=1)
        Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '01a1c02d-0073-726b-a47b-9c2fc6bc120c'::uuid))
        Filter: (version = '1'::bigint)
        Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.222 ms
Execution Time: 0.159 ms
```

Statement 21:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.009..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.031 ms
Execution Time: 0.025 ms
```

Statement 22 (write, not EXPLAINed):

```sql
INSERT INTO audit_log (
	tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
	version_before, version_after, actor_principal_id, actor_system, source, source_ref,
	change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256,
	diff_truncated, diff, diff_salt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 23:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.id = $2
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","01a1c02d-0073-726b-a47b-9c2fc6bc120c"]`

```
Result  (cost=1.00..25.08 rows=1 width=1261) (actual time=0.143..0.148 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
  Buffers: shared hit=10
  ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.122..0.126 rows=1 loops=1)
        Buffers: shared hit=10
        ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.090..0.093 rows=1 loops=1)
              Join Filter: (r.entity_definition_id = d.id)
              Buffers: shared hit=7
              ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.069..0.072 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = '01a1c02d-0073-726b-a47b-9c2fc6bc120c'::uuid))
                    Buffers: shared hit=4
              ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.017..0.017 rows=1 loops=1)
                    Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                    Buffers: shared hit=3
        ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.029..0.030 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=3
              ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.026..0.026 rows=1 loops=1)
                    Index Cond: ((tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid) AND (id = r.assignment_group_id))
                    Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.693 ms
Execution Time: 0.259 ms
```

Statement 24:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.031 ms
Execution Time: 0.026 ms
```

Statement 25:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.015..0.016 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.053 ms
Execution Time: 0.040 ms
```

Statement 26:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["2ac456d7-ac6f-4219-924b-836c8ba1cf46","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.125..0.128 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.122..0.124 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.107..0.113 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.092..0.097 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.040..0.042 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.047..0.049 rows=1 loops=1)
                          Index Cond: (tenant_id = '2ac456d7-ac6f-4219-924b-836c8ba1cf46'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.438 ms
Execution Time: 0.194 ms
```

Statement 27:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.042 ms
Execution Time: 0.034 ms
```

Statement 28 (write, not EXPLAINed):

```sql
UPDATE idempotency_keys
   SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'
```

Statement 29 (write, not EXPLAINed):

```sql
commit
```

### create, small tenant agent

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
begin
```

Statement 3 (write, not EXPLAINed):

```sql
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING
```

Statement 4 (write, not EXPLAINed):

```sql
commit
```

Statement 5 (write, not EXPLAINed):

```sql
begin
```

Statement 7:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.068..0.072 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.048..0.050 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.011..0.012 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.539 ms
Execution Time: 0.140 ms
```

Statement 8:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.009..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.035 ms
Execution Time: 0.028 ms
```

Statement 9:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.051..0.054 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.038..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.183 ms
Execution Time: 0.091 ms
```

Statement 10:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.058..0.062 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.038..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.559 ms
Execution Time: 0.114 ms
```

Statement 11:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.030 ms
Execution Time: 0.026 ms
```

Statement 12:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.033 ms
Execution Time: 0.030 ms
```

Statement 13:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.084..0.087 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.082..0.084 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.068..0.073 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.053..0.057 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.036..0.037 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.013..0.014 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.494 ms
Execution Time: 0.151 ms
```

Statement 14:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.044 ms
Execution Time: 0.032 ms
```

Statement 15:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.009..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.030 ms
Execution Time: 0.026 ms
```

Statement 16:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.070..0.072 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.068..0.070 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.053..0.059 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.040..0.045 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.010..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.408 ms
Execution Time: 0.125 ms
```

Statement 17:

```sql
SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL)
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","3164b0ca-6799-9372-d157-4acf299e68e8"]`

```
Result  (cost=8.31..8.32 rows=1 width=1) (actual time=0.059..0.061 rows=1 loops=1)
  Buffers: shared hit=3
  InitPlan 1 (returns $0)
    ->  Result  (cost=0.29..8.31 rows=1 width=0) (actual time=0.055..0.056 rows=1 loops=1)
          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
          Buffers: shared hit=3
          ->  Index Scan using groups_live_key on groups  (cost=0.29..8.31 rows=1 width=0) (actual time=0.041..0.042 rows=1 loops=1)
                Index Cond: ((tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid) AND (id = '3164b0ca-6799-9372-d157-4acf299e68e8'::uuid))
                Filter: (deleted_at IS NULL)
                Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.370 ms
Execution Time: 0.113 ms
```

Statement 18 (write, not EXPLAINed):

```sql
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1
```

Statement 19 (write, not EXPLAINed):

```sql
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data,
    causation_origin_id, causation_depth)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15)
```

Statement 20:

```sql
SELECT ((data - $4::text[]) || CASE WHEN assignee_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($5::text, assignee_id) END || CASE WHEN assignment_group_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($6::text, assignment_group_id) END || CASE WHEN requester_id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($7::text, requester_id) END || jsonb_build_object($8::text, state))::text FROM records WHERE tenant_id = $1 AND id = $2 AND version = $3
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","01a1c02d-68a5-7b19-95f9-27b3af8bf382",1,["assignee","group","requester","state"],"assignee","group","requester","state"]`

```
Result  (cost=0.42..8.47 rows=1 width=32) (actual time=0.148..0.153 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
  Buffers: shared hit=4
  ->  Index Scan using records_pkey on records  (cost=0.42..8.45 rows=1 width=1130) (actual time=0.054..0.057 rows=1 loops=1)
        Index Cond: ((tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid) AND (id = '01a1c02d-68a5-7b19-95f9-27b3af8bf382'::uuid))
        Filter: (version = '1'::bigint)
        Buffers: shared hit=4
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.366 ms
Execution Time: 0.210 ms
```

Statement 21:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.008..0.009 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.029 ms
Execution Time: 0.025 ms
```

Statement 22 (write, not EXPLAINed):

```sql
INSERT INTO audit_log (
	tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
	version_before, version_after, actor_principal_id, actor_system, source, source_ref,
	change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256,
	diff_truncated, diff, diff_salt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)
```

Statement 23:

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.id = $2
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","01a1c02d-68a5-7b19-95f9-27b3af8bf382"]`

```
Result  (cost=1.00..25.08 rows=1 width=1261) (actual time=0.132..0.137 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
  Buffers: shared hit=10
  ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.113..0.117 rows=1 loops=1)
        Buffers: shared hit=10
        ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.084..0.087 rows=1 loops=1)
              Join Filter: (r.entity_definition_id = d.id)
              Buffers: shared hit=7
              ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.061..0.064 rows=1 loops=1)
                    Index Cond: ((tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid) AND (id = '01a1c02d-68a5-7b19-95f9-27b3af8bf382'::uuid))
                    Buffers: shared hit=4
              ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.018..0.018 rows=1 loops=1)
                    Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                    Buffers: shared hit=3
        ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.026..0.026 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=3
              ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.021..0.021 rows=1 loops=1)
                    Index Cond: ((tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid) AND (id = r.assignment_group_id))
                    Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.994 ms
Execution Time: 0.259 ms
```

Statement 24:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.009..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.039 ms
Execution Time: 0.033 ms
```

Statement 25:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.044 ms
Execution Time: 0.032 ms
```

Statement 26:

```sql
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1
```

Arguments: `["046916af-2267-4cc2-aa9d-c561847b1ecf","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.086..0.089 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.084..0.085 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.069..0.073 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.054..0.058 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.038..0.039 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.012..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = '046916af-2267-4cc2-aa9d-c561847b1ecf'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.373 ms
Execution Time: 0.146 ms
```

Statement 27:

```sql
SELECT coalesce(current_setting('app.tenant_id', true), '')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.011..0.012 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.045 ms
Execution Time: 0.043 ms
```

Statement 28 (write, not EXPLAINed):

```sql
UPDATE idempotency_keys
   SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'
```

Statement 29 (write, not EXPLAINed):

```sql
commit
```

