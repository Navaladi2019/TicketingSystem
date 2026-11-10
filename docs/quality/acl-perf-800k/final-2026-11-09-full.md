### 2026-11-09T18:15:17Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 12545507328 bytes, kernel 5.15.167.4-microsoft-standard-WSL2

# P3-36 raw report (TICK-621)

Generated 2026-11-09T18:29:20Z by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): 8 clients, 20 s per scenario, through the real handlers.

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
records 808534, open 128812, priority 1 40021, avg data bytes 1075
tenants 400, largest 400000, smallest 1002
site s7, large tenant 1973 of 405283
record_shares 15815
heap 1053 MB, indexes 658 MB
```

## Transport floor

`SELECT 1` on the handler's pool, 8 clients x 200: p50 1.81 / p95 3.10 / p99 4.17 ms. Each statement in the tables below pays about this much before the database does any work.

## Results (ms, per request through the handler)

| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- |
| agent list by group, default sort, page 1 | large | 2837 | 54.11 | 72.54 | 90.03 | 150 | meets |
| agent list by group, default sort, page 20 | large | 1345 | 60.37 | 465.71 | 726.09 | 150 | **MISS** |
| requester list (heavy requesters), page 1 | large | 910 | 140.82 | 404.43 | 628.66 | 150 | **MISS** |
| requester list (heavy requesters), page 20 | large | 965 | 118.59 | 456.05 | 696.35 | 150 | **MISS** |
| admin list, default sort, page 1 | large | 1115 | 109.07 | 351.19 | 504.68 | 150 | **MISS** |
| admin list, default sort, page 20 | large | 439 | 341.48 | 680.37 | 932.30 | 150 | **MISS** |
| agent list, filter[state]=new | large | 288 | 525.57 | 884.92 | 1071.01 | 150 | **MISS** |
| agent list, filter[priority]=1 (data) | large | 266 | 577.75 | 931.21 | 1244.63 | 150 | **MISS** |
| agent list, filter[site]=s7 (rare data, < 1% of visible rows) | large | 429 | 357.28 | 511.33 | 634.74 | 150 | **MISS** |
| agent list, sort=-updated_at | large | 1853 | 75.81 | 146.19 | 208.99 | 150 | meets |
| agent list, sort=assignee | large | 2053 | 72.14 | 117.53 | 147.29 | 150 | meets |
| admin list, filter[state][in]=new,in_progress, sort=-updated_at | large | 2057 | 73.49 | 118.55 | 140.44 | 150 | meets |
| small tenant: agent list, page 1 | small | 1921 | 80.14 | 111.47 | 144.29 | 150 | meets |
| small tenant: admin list, page 1 | small | 3181 | 49.41 | 70.44 | 85.92 | 150 | meets |
| single GET, agent | large | 8821 | 17.27 | 24.97 | 31.17 | 50 | meets |
| single GET, admin | large | 10220 | 14.90 | 20.95 | 25.00 | 50 | meets |
| single GET, small tenant agent | small | 12404 | 12.44 | 16.12 | 18.46 | 50 | meets |
| create, agent (1 client) | large | 792 | 24.65 | 28.68 | 31.11 | 50 | meets |
| create, agent | large | 2214 | 58.07 | 162.80 | 246.44 | 50 | **MISS** |
| create, admin | large | 2173 | 57.58 | 166.50 | 233.74 | 50 | **MISS** |
| create, small tenant agent | small | 3200 | 44.74 | 82.91 | 111.53 | 50 | **MISS** |

## Counter hold (creates, D-3-9)

From the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (1.81 ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.

| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create, agent (1 client) | 792 | 3.43 / 4.54 | 5.22 / 6.87 | 1 | 1.81 / 1.81 | 3.28 / 4.89 | 0.00 / 0.60 |
| create, agent | 2214 | 28.55 / 133.48 | 6.81 / 9.01 | 1 | 1.81 / 1.81 | 4.62 / 6.84 | 0.00 / 0.63 |
| create, admin | 2173 | 28.73 / 135.90 | 6.81 / 9.76 | 1 | 1.81 / 1.81 | 4.62 / 7.69 | 0.00 / 0.62 |
| create, small tenant agent | 3200 | 6.35 / 10.94 | 9.82 / 18.87 | 1 | 1.81 / 1.81 | 7.69 / 16.70 | 0.53 / 0.71 |

## Per statement (ms, as pgx traced it, all requests of the scenario)

### agent list by group, default sort, page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 2837 | 3.11 | 7.16 | 11.40 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 2837 | 4.30 | 8.02 | 12.66 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2837 | 3.44 | 7.30 | 11.86 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2837 | 3.82 | 7.41 | 11.37 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2837 | 7.60 | 13.16 | 17.89 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2837 | 19.78 | 30.48 | 38.88 |
| 7 | `commit` | 2837 | 3.52 | 8.69 | 15.54 |

### agent list by group, default sort, page 20

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 1345 | 3.58 | 47.58 | 114.13 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1345 | 4.79 | 47.18 | 118.48 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1345 | 3.87 | 38.91 | 93.76 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1345 | 4.22 | 44.41 | 98.32 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1345 | 8.42 | 52.30 | 116.67 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1345 | 23.87 | 130.84 | 281.57 |
| 7 | `commit` | 1345 | 4.42 | 75.06 | 168.12 |

### requester list (heavy requesters), page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 910 | 10.30 | 50.86 | 103.07 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 910 | 11.51 | 50.36 | 111.12 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 910 | 10.41 | 49.22 | 105.60 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 910 | 11.26 | 55.91 | 109.05 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 910 | 14.40 | 65.94 | 114.42 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 910 | 37.91 | 136.70 | 211.74 |
| 7 | `commit` | 910 | 13.03 | 68.84 | 112.05 |

### requester list (heavy requesters), page 20

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 965 | 9.40 | 50.48 | 127.07 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 965 | 10.06 | 45.25 | 150.62 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 965 | 9.36 | 48.19 | 109.82 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 965 | 9.84 | 45.14 | 110.95 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 965 | 13.93 | 51.88 | 129.07 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 965 | 35.86 | 135.56 | 255.90 |
| 7 | `commit` | 965 | 10.99 | 64.87 | 177.65 |

### admin list, default sort, page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 1115 | 9.95 | 54.22 | 109.36 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1115 | 10.03 | 39.99 | 91.12 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1115 | 10.01 | 51.49 | 89.15 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1115 | 9.73 | 39.90 | 83.87 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 1115 | 10.72 | 53.85 | 97.19 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 1115 | 27.12 | 108.62 | 185.54 |
| 7 | `commit` | 1115 | 9.86 | 61.00 | 116.67 |

### admin list, default sort, page 20

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 439 | 24.94 | 123.06 | 254.49 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 439 | 24.17 | 95.07 | 186.43 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 439 | 21.57 | 108.08 | 309.62 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 439 | 23.54 | 96.47 | 166.45 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 439 | 27.20 | 117.38 | 165.01 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 439 | 103.27 | 298.96 | 430.58 |
| 7 | `commit` | 439 | 33.37 | 152.83 | 231.43 |

### agent list, filter[state]=new

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 288 | 16.72 | 77.97 | 113.76 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 288 | 16.35 | 73.25 | 151.51 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 288 | 15.66 | 55.08 | 80.81 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 288 | 15.78 | 57.17 | 102.47 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 288 | 148.69 | 294.71 | 447.16 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 288 | 214.36 | 434.59 | 499.16 |
| 7 | `commit` | 288 | 19.11 | 125.21 | 184.75 |

### agent list, filter[priority]=1 (data)

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 266 | 14.90 | 46.31 | 78.46 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 266 | 16.09 | 42.82 | 81.96 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 266 | 17.12 | 40.23 | 74.57 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 266 | 16.09 | 49.85 | 68.06 |
| 5 | `SELECT current_setting('statement_timeout')` | 266 | 15.35 | 60.18 | 107.77 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 266 | 14.96 | 57.34 | 150.93 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 266 | 157.99 | 288.82 | 357.76 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 266 | 226.38 | 414.13 | 560.72 |
| 9 | `SET LOCAL statement_timeout = '0'` | 266 | 19.30 | 109.25 | 221.64 |
| 10 | `commit` | 266 | 16.48 | 56.93 | 112.81 |

### agent list, filter[site]=s7 (rare data, < 1% of visible rows)

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 429 | 5.84 | 15.26 | 35.21 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 429 | 6.93 | 17.74 | 32.63 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 429 | 5.95 | 18.44 | 29.82 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 429 | 6.37 | 17.28 | 30.45 |
| 5 | `SELECT current_setting('statement_timeout')` | 429 | 5.71 | 15.85 | 24.34 |
| 6 | `SET LOCAL statement_timeout = '2000'` | 429 | 5.18 | 16.68 | 24.07 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 429 | 136.88 | 206.37 | 258.69 |
| 8 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 429 | 153.86 | 222.78 | 258.14 |
| 9 | `SET LOCAL statement_timeout = '0'` | 429 | 5.95 | 16.73 | 29.13 |
| 10 | `commit` | 429 | 5.37 | 14.23 | 26.50 |

### agent list, sort=-updated_at

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 1853 | 4.65 | 16.89 | 36.84 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1853 | 5.90 | 17.19 | 35.42 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1853 | 4.80 | 16.42 | 31.94 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1853 | 5.29 | 16.91 | 30.83 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1853 | 9.05 | 23.21 | 41.27 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1853 | 28.99 | 55.52 | 79.10 |
| 7 | `commit` | 1853 | 6.13 | 21.44 | 40.05 |

### agent list, sort=assignee

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 2053 | 4.41 | 11.82 | 22.35 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 2053 | 5.52 | 11.97 | 19.04 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2053 | 4.39 | 11.77 | 20.23 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2053 | 4.86 | 11.53 | 19.42 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 2053 | 8.58 | 19.14 | 27.51 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 2053 | 28.36 | 51.31 | 69.74 |
| 7 | `commit` | 2053 | 5.32 | 17.14 | 30.10 |

### admin list, filter[state][in]=new,in_progress, sort=-updated_at

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 2057 | 6.02 | 17.45 | 27.89 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 2057 | 7.03 | 15.93 | 23.91 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 2057 | 6.16 | 15.96 | 26.43 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 2057 | 6.67 | 16.54 | 27.24 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 2057 | 8.03 | 19.81 | 31.24 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 2057 | 20.98 | 40.05 | 56.92 |
| 7 | `commit` | 2057 | 6.29 | 17.55 | 28.75 |

### small tenant: agent list, page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '09ace1be-919d-4ea1-80e3-e6b6cdf53da9', true)` | 5 | 3.91 | 8.34 | 8.34 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 1921 | 5.43 | 10.67 | 18.70 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1921 | 4.39 | 9.79 | 15.56 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1921 | 4.86 | 10.08 | 15.09 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1921 | 19.34 | 30.47 | 39.30 |
| 6 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1921 | 28.05 | 43.67 | 55.30 |
| 7 | `commit` | 1921 | 4.80 | 12.64 | 20.34 |

### small tenant: admin list, page 1

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '09ace1be-919d-4ea1-80e3-e6b6cdf53da9', true)` | 8 | 3.37 | 10.42 | 10.42 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.sch…` | 3181 | 5.06 | 10.53 | 15.86 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 3181 | 4.25 | 8.92 | 13.19 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 3181 | 4.69 | 9.59 | 14.11 |
| 5 | `SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definiti…` | 3181 | 5.28 | 10.58 | 16.42 |
| 6 | `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.nu…` | 3181 | 12.61 | 22.60 | 28.99 |
| 7 | `commit` | 3181 | 4.01 | 9.72 | 17.14 |

### single GET, agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 8821 | 3.12 | 5.88 | 8.05 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text…` | 8821 | 9.54 | 14.35 | 17.55 |
| 3 | `commit` | 8821 | 3.11 | 5.94 | 8.47 |

### single GET, admin

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 10220 | 2.65 | 4.63 | 6.35 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text…` | 10220 | 8.41 | 12.10 | 14.82 |
| 3 | `commit` | 10220 | 2.66 | 4.71 | 6.37 |

### single GET, small tenant agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '09ace1be-919d-4ea1-80e3-e6b6cdf53da9', true)` | 32 | 2.32 | 4.95 | 7.89 |
| 2 | `BATCH: SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text…` | 12404 | 6.98 | 9.61 | 11.19 |
| 3 | `commit` | 12404 | 2.31 | 3.57 | 4.58 |

### create, agent (1 client)

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 1584 | 1.20 | 1.98 | 2.52 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 792 | 1.69 | 2.32 | 2.80 |
| 3 | `commit` | 792 | 3.88 | 5.10 | 5.58 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 1584 | 1.20 | 1.98 | 2.52 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 792 | 3.40 | 4.34 | 4.92 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 792 | 1.76 | 2.84 | 3.26 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 792 | 3.43 | 4.54 | 5.36 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 792 | 5.09 | 6.70 | 8.02 |

### create, agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 4428 | 1.76 | 2.92 | 3.50 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 2214 | 2.23 | 2.99 | 3.58 |
| 3 | `commit` | 2214 | 6.39 | 8.65 | 9.78 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 4428 | 1.76 | 2.92 | 3.50 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 2214 | 4.51 | 5.87 | 6.99 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2214 | 2.35 | 3.48 | 4.05 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 2214 | 28.55 | 133.48 | 217.75 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2214 | 6.43 | 8.65 | 10.15 |

### create, admin

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 4346 | 1.76 | 2.92 | 3.56 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 2173 | 2.24 | 3.16 | 3.97 |
| 3 | `commit` | 2173 | 6.41 | 8.96 | 13.92 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)` | 4346 | 1.76 | 2.92 | 3.56 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 2173 | 4.40 | 5.94 | 7.17 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 2173 | 2.34 | 3.52 | 4.21 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 2173 | 28.73 | 135.90 | 205.46 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 2173 | 6.42 | 9.50 | 14.09 |

### create, small tenant agent

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '09ace1be-919d-4ea1-80e3-e6b6cdf53da9', true)` | 18 | 2.25 | 4.93 | 4.93 |
| 2 | `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, exp…` | 3200 | 2.94 | 5.54 | 8.37 |
| 3 | `commit` | 3200 | 7.55 | 15.39 | 43.54 |
| 4 | `BEGIN; SELECT set_config('app.tenant_id', '09ace1be-919d-4ea1-80e3-e6b6cdf53da9', true)` | 18 | 2.25 | 4.93 | 4.93 |
| 5 | `BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key =…` | 3200 | 6.07 | 9.97 | 13.06 |
| 6 | `BATCH: SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at…` | 3200 | 3.48 | 6.35 | 8.94 |
| 7 | `BATCH: INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2,…` | 3200 | 6.35 | 10.94 | 16.30 |
| 8 | `BATCH: WITH done AS ( UPDATE idempotency_keys SET state = 'completed', response_status = $…` | 3200 | 9.50 | 18.51 | 30.96 |

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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.049..0.052 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.028..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.397 ms
Execution Time: 0.096 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.032 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.021..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.123 ms
Execution Time: 0.055 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.064..0.066 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.062..0.063 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.049..0.053 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.040..0.044 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.025..0.026 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.011..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.293 ms
Execution Time: 0.114 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=549.53..549.54 rows=1 width=8) (actual time=1.743..1.750 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..537.02 rows=1001 width=4) (actual time=0.062..1.571 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..599.18 rows=1117 width=4) (actual time=0.060..1.407 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..588.01 rows=1117 width=0) (actual time=0.060..1.190 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.14 rows=1001 width=0) (actual time=0.058..1.038 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.13 rows=1001 width=4) (actual time=0.057..0.858 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.99 rows=5751 width=4) (actual time=0.057..0.714 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.99 rows=5751 width=0) (actual time=0.048..0.482 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..389.24 rows=103 width=0) (never executed)
                          ->  Limit  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                ->  Result  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.21 rows=103 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.56 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.44 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.209 ms
Execution Time: 1.945 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1823.93 rows=1 width=1301) (actual time=0.429..2.242 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1815.60 rows=1 width=1266) (actual time=0.405..2.013 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1807.29 rows=1 width=1259) (actual time=0.383..1.833 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1806.14 rows=51 width=1259) (actual time=0.379..1.773 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..4015.19 rows=115 width=1259) (actual time=0.377..1.761 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3725.06 rows=51 width=1259) (actual time=0.131..1.486 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..420007.89 rows=5751 width=1259) (actual time=0.130..1.477 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..420007.89 rows=5751 width=1251) (actual time=0.125..1.452 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..192.44 rows=51 width=1259) (actual time=0.027..0.033 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..388.21 rows=103 width=1259) (actual time=0.027..0.032 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.21 rows=103 width=1251) (actual time=0.023..0.027 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..51.44 rows=12 width=1259) (actual time=0.026..0.028 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..51.44 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.44 rows=12 width=1251) (actual time=0.023..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.189..0.194 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.189..0.193 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.092..0.177 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.089..0.128 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.060..0.065 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.045..0.054 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.035..0.036 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.018..0.018 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.128 ms
Execution Time: 2.566 ms
```

### agent list by group, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjI1OjIwLjk0MDM3MiswMCIsImkiOiJhOTIzNTYzYi1kMTcyLTczYTYtNzU2YS05YjU1ZDg0NGEzMGQifQ.PzDy5yjCb8RrVCCC0USrGSK4dy0Iwgw25StGj00l9ac` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.079..0.084 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.056..0.059 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.013..0.014 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.932 ms
Execution Time: 0.169 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.046..0.049 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.035..0.037 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.183 ms
Execution Time: 0.085 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.078..0.081 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.076..0.078 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.063..0.068 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.050..0.054 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.033..0.034 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.012..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.438 ms
Execution Time: 0.143 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=549.53..549.54 rows=1 width=8) (actual time=2.375..2.386 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..537.02 rows=1001 width=4) (actual time=0.090..2.146 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..599.18 rows=1117 width=4) (actual time=0.088..1.933 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..588.01 rows=1117 width=0) (actual time=0.087..1.634 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.14 rows=1001 width=0) (actual time=0.086..1.442 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.13 rows=1001 width=4) (actual time=0.085..1.216 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.99 rows=5751 width=4) (actual time=0.084..1.031 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.99 rows=5751 width=0) (actual time=0.069..0.777 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..389.24 rows=103 width=0) (never executed)
                          ->  Limit  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                ->  Result  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.21 rows=103 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.56 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.44 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.801 ms
Execution Time: 2.773 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"2026-09-30 23:25:20.940372+00","a923563b-d172-73a6-756a-9b55d844a30d",51]`

```
Nested Loop Left Join  (cost=46.38..1851.47 rows=1 width=1301) (actual time=0.416..3.178 rows=51 loops=1)
  Buffers: shared hit=1118
  ->  Nested Loop  (cost=46.09..1843.14 rows=1 width=1266) (actual time=0.383..2.752 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=965
        ->  Subquery Scan on x  (cost=45.80..1834.83 rows=1 width=1259) (actual time=0.358..2.469 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=812
              ->  Limit  (cost=45.80..1833.68 rows=51 width=1259) (actual time=0.354..2.368 rows=51 loops=1)
                    Buffers: shared hit=812
                    ->  Merge Append  (cost=45.80..4077.28 rows=115 width=1259) (actual time=0.352..2.350 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=812
                          ->  Limit  (cost=0.42..3790.19 rows=51 width=1259) (actual time=0.084..2.034 rows=49 loops=1)
                                Buffers: shared hit=777
                                ->  Result  (cost=0.42..410558.16 rows=5525 width=1259) (actual time=0.083..2.021 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=777
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..410558.16 rows=5525 width=1251) (actual time=0.070..1.982 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 716
                                            Buffers: shared hit=777
                          ->  Limit  (cost=0.42..192.93 rows=51 width=1259) (actual time=0.038..0.047 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..374.11 rows=99 width=1259) (actual time=0.037..0.044 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..374.11 rows=99 width=1251) (actual time=0.033..0.038 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..47.91 rows=12 width=1259) (actual time=0.040..0.049 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..47.91 rows=12 width=1259) (actual time=0.039..0.048 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..47.91 rows=12 width=1251) (actual time=0.036..0.041 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.185..0.193 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.184..0.191 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.127..0.171 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.123..0.165 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.083..0.089 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.063..0.074 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.052..0.054 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.025..0.026 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.022..0.022 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:20.940372+00'::timestamp with time zone, 'a923563b-d172-73a6-756a-9b55d844a30d'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.943 ms
Execution Time: 3.755 ms
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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.044..0.047 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.026..0.028 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.414 ms
Execution Time: 0.090 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.048..0.051 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.036 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.177 ms
Execution Time: 0.096 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.085..0.087 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.082..0.084 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.069..0.074 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.055..0.059 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.037..0.038 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.014..0.016 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.343 ms
Execution Time: 0.137 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=197.02..197.03 rows=1 width=8) (actual time=1.541..1.548 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..184.51 rows=1001 width=4) (actual time=0.050..1.413 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..187.08 rows=1015 width=4) (actual time=0.049..1.269 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..176.93 rows=1015 width=0) (actual time=0.048..1.078 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..99.25 rows=1001 width=0) (actual time=0.048..0.907 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..89.24 rows=1001 width=4) (actual time=0.047..0.752 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..122.95 rows=1381 width=4) (actual time=0.046..0.628 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..122.95 rows=1381 width=0) (actual time=0.037..0.416 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..51.50 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.37 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..51.37 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..51.37 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.842 ms
Execution Time: 1.696 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],51]`

```
Nested Loop Left Join  (cost=22.56..225.29 rows=1 width=1301) (actual time=0.170..1.063 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..216.95 rows=1 width=1266) (actual time=0.141..0.715 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.98..208.64 rows=1 width=1259) (actual time=0.124..0.489 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.98..207.49 rows=51 width=1259) (actual time=0.121..0.363 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.98..258.42 rows=65 width=1259) (actual time=0.119..0.349 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..185.06 rows=51 width=1259) (actual time=0.056..0.263 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..5000.16 rows=1381 width=1259) (actual time=0.055..0.253 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..5000.16 rows=1381 width=1251) (actual time=0.049..0.224 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..51.37 rows=13 width=1259) (actual time=0.030..0.032 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..51.37 rows=13 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..51.37 rows=13 width=1251) (actual time=0.026..0.026 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.031..0.037 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.11..21.12 rows=1 width=1259) (actual time=0.031..0.036 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.10 rows=1 width=1259) (actual time=0.025..0.030 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.10 rows=1 width=1251) (actual time=0.023..0.027 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.023..0.026 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.020..0.022 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.092 ms
Execution Time: 1.311 ms
```

### requester list (heavy requesters), page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTI5IDAzOjIwOjI1LjYwNTY3NSswMCIsImkiOiJkOWZlM2ZiMi02ZWU5LTdhMTktNjE1Ny0wODUxOWU2M2U3OTcifQ.625wN2FsCLyPynNGQVrkrlnqPbD1bcf8JNN9Zt8tPvo` as `req-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.061..0.065 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.038..0.040 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.013..0.014 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.538 ms
Execution Time: 0.126 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.050..0.054 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.037..0.039 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.291 ms
Execution Time: 0.100 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.083..0.085 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.081..0.083 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.067..0.072 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.054..0.059 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.035..0.036 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.014..0.016 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.421 ms
Execution Time: 0.144 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid)) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[]]`

```
Aggregate  (cost=197.02..197.03 rows=1 width=8) (actual time=3.021..3.035 rows=1 loops=1)
  Buffers: shared hit=565
  ->  Limit  (cost=0.42..184.51 rows=1001 width=4) (actual time=0.453..2.850 rows=1001 loops=1)
        Buffers: shared hit=565
        ->  Result  (cost=0.42..187.08 rows=1015 width=4) (actual time=0.445..2.645 rows=1001 loops=1)
              Buffers: shared hit=565
              ->  Append  (cost=0.42..176.93 rows=1015 width=0) (actual time=0.443..2.402 rows=1001 loops=1)
                    Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.42..99.25 rows=1001 width=0) (actual time=0.439..2.189 rows=1001 loops=1)
                          Buffers: shared hit=565
                          ->  Limit  (cost=0.42..89.24 rows=1001 width=4) (actual time=0.434..1.823 rows=1001 loops=1)
                                Buffers: shared hit=565
                                ->  Result  (cost=0.42..122.95 rows=1381 width=4) (actual time=0.431..1.523 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=565
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.42..122.95 rows=1381 width=0) (actual time=0.392..1.235 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=565
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..51.50 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.37 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..51.37 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..51.37 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=13.05..21.11 rows=1 width=0) (never executed)
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.897 ms
Execution Time: 3.426 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0","fccbb3fd-048d-a9ac-ec41-5c1548c480a0",[],"2026-09-29 03:20:25.605675+00","d9fe3fb2-6ee9-7a19-6157-08519e63e797",51]`

```
Nested Loop Left Join  (cost=22.57..228.62 rows=1 width=1301) (actual time=0.715..3.679 rows=51 loops=1)
  Buffers: shared hit=367
  ->  Nested Loop  (cost=22.27..220.28 rows=1 width=1266) (actual time=0.637..2.711 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214
        ->  Subquery Scan on x  (cost=21.99..211.97 rows=1 width=1259) (actual time=0.565..2.267 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=61
              ->  Limit  (cost=21.99..210.82 rows=51 width=1259) (actual time=0.557..2.093 rows=51 loops=1)
                    Buffers: shared hit=61
                    ->  Merge Append  (cost=21.99..251.55 rows=62 width=1259) (actual time=0.551..2.067 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=61
                          ->  Limit  (cost=0.42..185.44 rows=51 width=1259) (actual time=0.336..1.806 rows=51 loops=1)
                                Buffers: shared hit=55
                                ->  Result  (cost=0.42..4146.91 rows=1143 width=1259) (actual time=0.335..1.793 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=55
                                      ->  Index Scan using records_list_requester on records r  (cost=0.42..4146.91 rows=1143 width=1251) (actual time=0.186..1.606 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Buffers: shared hit=55
                          ->  Limit  (cost=0.42..44.17 rows=10 width=1259) (actual time=0.111..0.113 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..44.17 rows=10 width=1259) (actual time=0.110..0.111 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..44.17 rows=10 width=1251) (actual time=0.100..0.101 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.096..0.104 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=21.12..21.12 rows=1 width=1259) (actual time=0.095..0.102 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=13.05..21.11 rows=1 width=1259) (actual time=0.061..0.067 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=13.05..21.11 rows=1 width=1251) (actual time=0.053..0.058 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.052..0.055 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.042..0.044 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.037..0.039 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.034..0.034 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=1251) (never executed)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:20:25.605675+00'::timestamp with time zone, 'd9fe3fb2-6ee9-7a19-6157-08519e63e797'::uuid)))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.005..0.005 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.013..0.013 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.243 ms
Execution Time: 4.678 ms
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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.109..0.113 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.081..0.084 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.014..0.014 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.695 ms
Execution Time: 0.188 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.048..0.051 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.034..0.037 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.159 ms
Execution Time: 0.092 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.154..0.156 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.150..0.151 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.125..0.130 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.103..0.108 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.075..0.076 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.023..0.025 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.544 ms
Execution Time: 0.251 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND true LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc",["6851538a-9e01-4034-bcc1-7b2a36167d92"]]`

```
Aggregate  (cost=78.80..78.81 rows=1 width=8) (actual time=0.890..0.892 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.29 rows=1001 width=4) (actual time=0.062..0.766 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..13045.84 rows=198271 width=4) (actual time=0.060..0.587 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..13045.84 rows=198271 width=0) (actual time=0.047..0.361 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.480 ms
Execution Time: 0.956 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc",["6851538a-9e01-4034-bcc1-7b2a36167d92"],51]`

```
Limit  (cost=1.00..126.49 rows=51 width=1293) (actual time=0.136..1.086 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..487856.96 rows=198271 width=1293) (actual time=0.133..1.067 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..486369.93 rows=198271 width=1266) (actual time=0.108..0.952 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..421503.22 rows=198271 width=1258) (actual time=0.081..0.373 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..419016.54 rows=198271 width=1251) (actual time=0.054..0.273 rows=51 loops=1)
                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.017..0.020 rows=1 loops=1)
                                Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                Filter: (id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.871 ms
Execution Time: 1.302 ms
```

### admin list, default sort, page 20

Request: `GET /api/v1/records/ticket?cursor=k1.eyJzIjoidXBkYXRlZF9hdCIsImQiOnRydWUsInYiOiIyMDI2LTA5LTMwIDIzOjU5OjU5LjU2MzExOCswMCIsImkiOiJkNmRhMTEwZi1jOGM3LTEzZmItZjRmNy1jNWUzYjFkMmMzMjQifQ.hfuz3qAK17wPtbkPJO--wEHHd1RECmscyNViIqFwKvw` as `admin-large`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.079..0.083 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.051..0.053 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.017..0.017 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.561 ms
Execution Time: 0.154 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.056..0.060 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.040..0.043 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.291 ms
Execution Time: 0.114 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.103..0.106 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.100..0.103 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.081..0.088 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.067..0.073 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.043..0.044 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.017..0.021 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.549 ms
Execution Time: 0.185 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND true LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc",["6851538a-9e01-4034-bcc1-7b2a36167d92"]]`

```
Aggregate  (cost=78.80..78.81 rows=1 width=8) (actual time=1.497..1.500 rows=1 loops=1)
  Buffers: shared hit=13
  ->  Limit  (cost=0.42..66.29 rows=1001 width=4) (actual time=0.183..1.237 rows=1001 loops=1)
        Buffers: shared hit=13
        ->  Result  (cost=0.42..13045.84 rows=198271 width=4) (actual time=0.180..0.978 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=13
              ->  Index Only Scan using records_number_key on records r  (cost=0.42..13045.84 rows=198271 width=0) (actual time=0.162..0.626 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=13
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.454 ms
Execution Time: 1.587 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc",["6851538a-9e01-4034-bcc1-7b2a36167d92"],"2026-09-30 23:59:59.563118+00","d6da110f-c8c7-13fb-f4f7-c5e3b1d2c324",51]`

```
Limit  (cost=1.00..126.62 rows=51 width=1293) (actual time=0.128..1.427 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..488310.93 rows=198245 width=1293) (actual time=0.126..1.360 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..486824.09 rows=198245 width=1266) (actual time=0.103..1.247 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..421965.63 rows=198245 width=1258) (actual time=0.076..0.560 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..419479.28 rows=198245 width=1251) (actual time=0.052..0.460 rows=51 loops=1)
                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:59:59.563118+00'::timestamp with time zone, 'd6da110f-c8c7-13fb-f4f7-c5e3b1d2c324'::uuid)))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.016..0.018 rows=1 loops=1)
                                Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                Filter: (id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.867 ms
Execution Time: 1.599 ms
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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.095..0.101 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.063..0.066 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.018..0.018 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.717 ms
Execution Time: 0.188 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.067..0.072 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.050..0.053 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.308 ms
Execution Time: 0.129 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.173..0.176 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.170..0.173 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.147..0.153 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.130..0.135 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.085..0.087 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.036..0.038 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.605 ms
Execution Time: 0.261 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = ANY($8::text[]) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"]]`

```
Aggregate  (cost=20107.55..20107.56 rows=1 width=8) (actual time=209.881..209.894 rows=1 loops=1)
  Buffers: shared hit=18568
  ->  Limit  (cost=440.57..20103.83 rows=298 width=4) (actual time=53.377..206.022 rows=1001 loops=1)
        Buffers: shared hit=18568
        ->  Result  (cost=440.57..20103.83 rows=298 width=4) (actual time=53.373..205.580 rows=1001 loops=1)
              Buffers: shared hit=18568
              ->  Append  (cost=440.57..20100.85 rows=298 width=0) (actual time=53.370..205.142 rows=1001 loops=1)
                    Buffers: shared hit=18568
                    ->  Subquery Scan on "*SELECT* 1"  (cost=440.57..19615.01 rows=291 width=0) (actual time=53.366..204.764 rows=1001 loops=1)
                          Buffers: shared hit=18568
                          ->  Limit  (cost=440.57..19612.10 rows=291 width=4) (actual time=53.361..204.295 rows=1001 loops=1)
                                Buffers: shared hit=18568
                                ->  Result  (cost=440.57..19612.10 rows=291 width=4) (actual time=53.355..203.908 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=18568
                                      ->  Bitmap Heap Scan on records r  (cost=440.57..19612.10 rows=291 width=0) (actual time=53.334..203.134 rows=1001 loops=1)
                                            Recheck Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (state = ANY ('{new}'::text[]))
                                            Rows Removed by Filter: 20147
                                            Heap Blocks: exact=18193
                                            Buffers: shared hit=18568
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..440.48 rows=5751 width=0) (actual time=34.006..34.007 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..388.40 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.42..388.35 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.42..388.35 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.35 rows=5 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.46 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.45 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..51.45 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.45 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.580 ms
Execution Time: 210.175 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],["new"],51]`

```
Nested Loop Left Join  (cost=19667.77..20071.84 rows=1 width=1301) (actual time=248.404..249.462 rows=51 loops=1)
  Buffers: shared hit=21516
  ->  Nested Loop  (cost=19667.47..20063.50 rows=1 width=1266) (actual time=248.372..249.047 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21363
        ->  Subquery Scan on x  (cost=19667.19..20055.19 rows=1 width=1259) (actual time=248.352..248.765 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21210
              ->  Limit  (cost=19667.19..20054.04 rows=51 width=1259) (actual time=248.346..248.610 rows=51 loops=1)
                    Buffers: shared hit=21210
                    ->  Merge Append  (cost=19667.19..20107.14 rows=58 width=1259) (actual time=248.343..248.593 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21210
                          ->  Limit  (cost=19621.81..19621.94 rows=51 width=1259) (actual time=247.989..248.027 rows=50 loops=1)
                                Buffers: shared hit=21119
                                ->  Sort  (cost=19621.81..19622.54 rows=291 width=1259) (actual time=247.987..248.012 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=21119
                                      ->  Result  (cost=440.57..19612.10 rows=291 width=1259) (actual time=76.369..245.119 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=21119
                                            ->  Bitmap Heap Scan on records r  (cost=440.57..19612.10 rows=291 width=1251) (actual time=76.354..244.137 rows=1168 loops=1)
                                                  Recheck Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (state = ANY ('{new}'::text[]))
                                                  Rows Removed by Filter: 22950
                                                  Heap Blocks: exact=20744
                                                  Buffers: shared hit=21119
                                                  ->  Bitmap Index Scan on records_list_group  (cost=0.00..440.48 rows=5751 width=0) (actual time=43.010..43.010 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=375
                          ->  Limit  (cost=0.42..388.35 rows=5 width=1259) (actual time=0.095..0.228 rows=2 loops=1)
                                Buffers: shared hit=46
                                ->  Result  (cost=0.42..388.35 rows=5 width=1259) (actual time=0.092..0.223 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=46
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.35 rows=5 width=1251) (actual time=0.072..0.201 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=46
                          ->  Limit  (cost=0.42..51.45 rows=1 width=1259) (actual time=0.085..0.087 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..51.45 rows=1 width=1259) (actual time=0.083..0.084 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.45 rows=1 width=1251) (actual time=0.077..0.077 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.164..0.172 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.163..0.170 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.114..0.154 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.109..0.147 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.077..0.083 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.055..0.065 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.043..0.045 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.018..0.018 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.017..0.017 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.885 ms
Execution Time: 249.868 ms
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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.074..0.078 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.048..0.050 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.015..0.015 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.547 ms
Execution Time: 0.144 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.071..0.075 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.054..0.056 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.320 ms
Execution Time: 0.137 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.111..0.114 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.109..0.111 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.090..0.096 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.073..0.078 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.045..0.047 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.021..0.024 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.723 ms
Execution Time: 0.200 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.014..0.016 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.037 ms
Execution Time: 0.088 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}"]`

```
Aggregate  (cost=20105.95..20105.96 rows=1 width=8) (actual time=171.264..171.294 rows=1 loops=1)
  Buffers: shared hit=17769
  ->  Limit  (cost=440.51..20105.19 rows=61 width=4) (actual time=27.721..170.676 rows=1001 loops=1)
        Buffers: shared hit=17769
        ->  Result  (cost=440.51..20105.19 rows=61 width=4) (actual time=27.718..170.350 rows=1001 loops=1)
              Buffers: shared hit=17769
              ->  Append  (cost=440.51..20104.58 rows=61 width=0) (actual time=27.717..169.766 rows=1001 loops=1)
                    Buffers: shared hit=17769
                    ->  Subquery Scan on "*SELECT* 1"  (cost=440.51..19619.81 rows=58 width=0) (actual time=27.714..169.419 rows=1001 loops=1)
                          Buffers: shared hit=17769
                          ->  Limit  (cost=440.51..19619.23 rows=58 width=4) (actual time=27.712..168.833 rows=1001 loops=1)
                                Buffers: shared hit=17769
                                ->  Result  (cost=440.51..19619.23 rows=58 width=4) (actual time=27.710..168.530 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=17769
                                      ->  Bitmap Heap Scan on records r  (cost=440.51..19619.23 rows=58 width=0) (actual time=27.692..167.822 rows=1001 loops=1)
                                            Recheck Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"priority": "1"}'::jsonb)
                                            Rows Removed by Filter: 19221
                                            Heap Blocks: exact=17394
                                            Buffers: shared hit=17769
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..440.48 rows=5751 width=0) (actual time=17.582..17.583 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..388.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..388.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..388.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.48 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..51.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.47 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.411 ms
Execution Time: 171.637 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"priority\":\"1\"}",51]`

```
Nested Loop Left Join  (cost=19666.89..20099.70 rows=1 width=1301) (actual time=174.678..175.462 rows=51 loops=1)
  Buffers: shared hit=21490
  ->  Nested Loop  (cost=19666.59..20091.36 rows=1 width=1266) (actual time=174.647..175.136 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21337
        ->  Subquery Scan on x  (cost=19666.31..20083.05 rows=1 width=1259) (actual time=174.627..174.893 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21184
              ->  Limit  (cost=19666.31..20081.90 rows=51 width=1259) (actual time=174.622..174.773 rows=51 loops=1)
                    Buffers: shared hit=21184
                    ->  Merge Append  (cost=19666.31..20106.35 rows=54 width=1259) (actual time=174.619..174.759 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21184
                          ->  Limit  (cost=19620.93..19621.06 rows=51 width=1259) (actual time=174.268..174.302 rows=50 loops=1)
                                Buffers: shared hit=21119
                                ->  Sort  (cost=19620.93..19621.08 rows=58 width=1259) (actual time=174.265..174.290 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=21119
                                      ->  Result  (cost=440.51..19619.23 rows=58 width=1259) (actual time=40.092..171.036 rows=1182 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=21119
                                            ->  Bitmap Heap Scan on records r  (cost=440.51..19619.23 rows=58 width=1251) (actual time=40.074..170.136 rows=1182 loops=1)
                                                  Recheck Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"priority": "1"}'::jsonb)
                                                  Rows Removed by Filter: 22936
                                                  Heap Blocks: exact=20744
                                                  Buffers: shared hit=21119
                                                  ->  Bitmap Index Scan on records_list_group  (cost=0.00..440.48 rows=5751 width=0) (actual time=16.999..16.999 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=375
                          ->  Limit  (cost=0.42..388.48 rows=1 width=1259) (actual time=0.150..0.152 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.42..388.48 rows=1 width=1259) (actual time=0.145..0.146 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=23
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.48 rows=1 width=1251) (actual time=0.124..0.124 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 19
                                            Buffers: shared hit=23
                          ->  Limit  (cost=0.42..51.47 rows=1 width=1259) (actual time=0.040..0.099 rows=2 loops=1)
                                Buffers: shared hit=17
                                ->  Result  (cost=0.42..51.47 rows=1 width=1259) (actual time=0.039..0.097 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=17
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.47 rows=1 width=1251) (actual time=0.033..0.088 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                            Rows Removed by Filter: 12
                                            Buffers: shared hit=17
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.152..0.159 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.151..0.157 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.138..0.144 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.134..0.139 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.076..0.081 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.054..0.064 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.041..0.043 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.016..0.017 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"priority": "1"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.896 ms
Execution Time: 175.899 ms
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
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.058..0.062 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.037..0.039 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.535 ms
Execution Time: 0.119 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.041..0.044 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.031 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.186 ms
Execution Time: 0.086 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.070..0.073 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.068..0.070 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.054..0.059 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.044..0.048 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.027..0.028 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.012..0.014 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.391 ms
Execution Time: 0.129 ms
```

Statement 5:

```sql
SELECT current_setting('statement_timeout')
```

Arguments: `null`

```
Result  (cost=0.00..0.01 rows=1 width=32) (actual time=0.009..0.010 rows=1 loops=1)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.034 ms
Execution Time: 0.026 ms
```

Statement 7:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (r.data @> $8::jsonb) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}"]`

```
Aggregate  (cost=20105.95..20105.96 rows=1 width=8) (actual time=120.702..120.717 rows=1 loops=1)
  Buffers: shared hit=21734
  ->  Limit  (cost=440.51..20105.19 rows=61 width=4) (actual time=22.015..120.489 rows=121 loops=1)
        Buffers: shared hit=21734
        ->  Result  (cost=440.51..20105.19 rows=61 width=4) (actual time=22.011..120.364 rows=121 loops=1)
              Buffers: shared hit=21734
              ->  Append  (cost=440.51..20104.58 rows=61 width=0) (actual time=22.009..120.227 rows=121 loops=1)
                    Buffers: shared hit=21734
                    ->  Subquery Scan on "*SELECT* 1"  (cost=440.51..19619.81 rows=58 width=0) (actual time=22.007..117.858 rows=120 loops=1)
                          Buffers: shared hit=21119
                          ->  Limit  (cost=440.51..19619.23 rows=58 width=4) (actual time=22.006..117.726 rows=120 loops=1)
                                Buffers: shared hit=21119
                                ->  Result  (cost=440.51..19619.23 rows=58 width=4) (actual time=22.003..117.625 rows=120 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=21119
                                      ->  Bitmap Heap Scan on records r  (cost=440.51..19619.23 rows=58 width=0) (actual time=21.985..117.284 rows=120 loops=1)
                                            Recheck Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (data @> '{"site": "s7"}'::jsonb)
                                            Rows Removed by Filter: 23998
                                            Heap Blocks: exact=20744
                                            Buffers: shared hit=21119
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..440.48 rows=5751 width=0) (actual time=10.577..10.578 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..388.49 rows=1 width=0) (actual time=0.685..1.733 rows=1 loops=1)
                          Buffers: shared hit=523
                          ->  Limit  (cost=0.42..388.48 rows=1 width=4) (actual time=0.683..1.729 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..388.48 rows=1 width=4) (actual time=0.679..1.724 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.48 rows=1 width=0) (actual time=0.660..1.703 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.48 rows=1 width=0) (actual time=0.339..0.341 rows=0 loops=1)
                          Buffers: shared hit=67
                          ->  Limit  (cost=0.42..51.47 rows=1 width=4) (actual time=0.338..0.339 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..51.47 rows=1 width=4) (actual time=0.335..0.336 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.47 rows=1 width=0) (actual time=0.320..0.320 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (actual time=0.170..0.173 rows=0 loops=1)
                          Buffers: shared hit=25
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (actual time=0.169..0.172 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (actual time=0.167..0.169 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=25
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (actual time=0.155..0.158 rows=0 loops=1)
                                            Buffers: shared hit=25
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.090..0.094 rows=3 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=13
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.066..0.078 rows=3 loops=1)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Heap Blocks: exact=3
                                                        Buffers: shared hit=13
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.050..0.051 rows=0 loops=1)
                                                              Buffers: shared hit=10
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.021..0.022 rows=3 loops=1)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.026..0.026 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=8
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.084 ms
Execution Time: 121.008 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"{\"site\":\"s7\"}",51]`

```
Nested Loop Left Join  (cost=19666.89..20099.70 rows=1 width=1301) (actual time=132.516..138.106 rows=51 loops=1)
  Buffers: shared hit=22040
  ->  Nested Loop  (cost=19666.59..20091.36 rows=1 width=1266) (actual time=132.477..137.579 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21887
        ->  Subquery Scan on x  (cost=19666.31..20083.05 rows=1 width=1259) (actual time=132.451..135.706 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21734
              ->  Limit  (cost=19666.31..20081.90 rows=51 width=1259) (actual time=132.446..135.585 rows=51 loops=1)
                    Buffers: shared hit=21734
                    ->  Merge Append  (cost=19666.31..20106.35 rows=54 width=1259) (actual time=132.442..135.479 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21734
                          ->  Limit  (cost=19620.93..19621.06 rows=51 width=1259) (actual time=131.244..131.281 rows=50 loops=1)
                                Buffers: shared hit=21119
                                ->  Sort  (cost=19620.93..19621.08 rows=58 width=1259) (actual time=131.242..131.268 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=21119
                                      ->  Result  (cost=440.51..19619.23 rows=58 width=1259) (actual time=24.479..130.461 rows=120 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=21119
                                            ->  Bitmap Heap Scan on records r  (cost=440.51..19619.23 rows=58 width=1251) (actual time=24.463..129.986 rows=120 loops=1)
                                                  Recheck Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (data @> '{"site": "s7"}'::jsonb)
                                                  Rows Removed by Filter: 23998
                                                  Heap Blocks: exact=20744
                                                  Buffers: shared hit=21119
                                                  ->  Bitmap Index Scan on records_list_group  (cost=0.00..440.48 rows=5751 width=0) (actual time=13.006..13.007 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=375
                          ->  Limit  (cost=0.42..388.48 rows=1 width=1259) (actual time=0.745..3.168 rows=1 loops=1)
                                Buffers: shared hit=523
                                ->  Result  (cost=0.42..388.48 rows=1 width=1259) (actual time=0.742..3.162 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=523
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.48 rows=1 width=1251) (actual time=0.724..3.141 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 512
                                            Buffers: shared hit=523
                          ->  Limit  (cost=0.42..51.47 rows=1 width=1259) (actual time=0.270..0.272 rows=0 loops=1)
                                Buffers: shared hit=67
                                ->  Result  (cost=0.42..51.47 rows=1 width=1259) (actual time=0.267..0.269 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=67
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.47 rows=1 width=1251) (actual time=0.252..0.253 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                            Rows Removed by Filter: 64
                                            Buffers: shared hit=67
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.174..0.705 rows=0 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1259) (actual time=0.172..0.702 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1259) (actual time=0.156..0.684 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.148..0.674 rows=0 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.084..0.092 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.061..0.073 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.046..0.048 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.020 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.018..0.018 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (data @> '{"site": "s7"}'::jsonb))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.033..0.033 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 5.707 ms
Execution Time: 138.548 ms
```

### agent list, sort=-updated_at

Request: `GET /api/v1/records/ticket?sort=-updated_at` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.074..0.078 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.048..0.050 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.015..0.015 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.754 ms
Execution Time: 0.148 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.043..0.047 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.183 ms
Execution Time: 0.088 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.087..0.089 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.084..0.086 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.068..0.073 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.054..0.059 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.034..0.035 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.015..0.017 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.470 ms
Execution Time: 0.159 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=549.53..549.54 rows=1 width=8) (actual time=2.271..2.324 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..537.02 rows=1001 width=4) (actual time=0.074..2.073 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..599.18 rows=1117 width=4) (actual time=0.072..1.873 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..588.01 rows=1117 width=0) (actual time=0.072..1.611 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.14 rows=1001 width=0) (actual time=0.070..1.425 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.13 rows=1001 width=4) (actual time=0.069..1.206 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.99 rows=5751 width=4) (actual time=0.068..1.035 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.99 rows=5751 width=0) (actual time=0.054..0.746 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..389.24 rows=103 width=0) (never executed)
                          ->  Limit  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                ->  Result  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.21 rows=103 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.56 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.44 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.733 ms
Execution Time: 2.527 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=46.38..1823.93 rows=1 width=1301) (actual time=0.491..3.381 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.08..1815.60 rows=1 width=1266) (actual time=0.462..3.018 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=45.80..1807.29 rows=1 width=1259) (actual time=0.444..2.710 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=45.80..1806.14 rows=51 width=1259) (actual time=0.440..2.603 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=45.80..4015.19 rows=115 width=1259) (actual time=0.438..2.585 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..3725.06 rows=51 width=1259) (actual time=0.208..2.309 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..420007.89 rows=5751 width=1259) (actual time=0.207..2.295 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..420007.89 rows=5751 width=1251) (actual time=0.196..2.255 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.42..192.44 rows=51 width=1259) (actual time=0.036..0.047 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..388.21 rows=103 width=1259) (actual time=0.035..0.045 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.21 rows=103 width=1251) (actual time=0.028..0.035 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.42..51.44 rows=12 width=1259) (actual time=0.030..0.032 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..51.44 rows=12 width=1259) (actual time=0.030..0.031 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.44 rows=12 width=1251) (actual time=0.026..0.026 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.157..0.167 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.156..0.165 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.104..0.146 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.099..0.139 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.074..0.081 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.054..0.066 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.042..0.044 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.025..0.025 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.016..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.811 ms
Execution Time: 3.746 ms
```

### agent list, sort=assignee

Request: `GET /api/v1/records/ticket?sort=assignee` as `agent-1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.157..0.162 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.052..0.055 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.011..0.011 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.567 ms
Execution Time: 0.233 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.056..0.061 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.044..0.047 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.274 ms
Execution Time: 0.109 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.093..0.095 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.090..0.092 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.074..0.079 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.061..0.065 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.041..0.042 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.015..0.017 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.640 ms
Execution Time: 0.171 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"]]`

```
Aggregate  (cost=549.53..549.54 rows=1 width=8) (actual time=2.465..2.477 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..537.02 rows=1001 width=4) (actual time=0.076..2.327 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..599.18 rows=1117 width=4) (actual time=0.073..2.130 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..588.01 rows=1117 width=0) (actual time=0.072..1.842 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..97.14 rows=1001 width=0) (actual time=0.071..1.629 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..87.13 rows=1001 width=4) (actual time=0.070..1.399 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..497.99 rows=5751 width=4) (actual time=0.069..1.204 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..497.99 rows=5751 width=0) (actual time=0.055..0.891 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..389.24 rows=103 width=0) (never executed)
                          ->  Limit  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                ->  Result  (cost=0.42..388.21 rows=103 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.21 rows=103 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..51.56 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..51.44 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..51.44 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.893 ms
Execution Time: 2.693 ms
```

Statement 6:

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.assignee_id AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.assignee_id ASC NULLS LAST, r.updated_at ASC, r.id ASC LIMIT $8)
ORDER BY sort_key ASC NULLS LAST, updated_at ASC, id ASC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key ASC NULLS LAST, x.updated_at ASC, x.id ASC
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],"59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344","59fb22d0-8db4-ebf3-0411-bc5e59f13344",["9e8a3462-a1b0-3ab1-93b3-a00e99528d30","969c31f2-d8fe-635a-5a2d-95d39e4e7489","2f067c2d-b39a-c447-6bd8-6f26d53e8105","e43d6642-9bbe-05be-b6a6-8d3a87bf4141"],51]`

```
Nested Loop Left Join  (cost=488.83..2167.68 rows=1 width=1309) (actual time=3.345..7.131 rows=51 loops=1)
  Buffers: shared hit=1718
  ->  Nested Loop  (cost=488.54..2159.34 rows=1 width=1274) (actual time=3.319..6.716 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1565
        ->  Subquery Scan on x  (cost=488.25..2151.03 rows=1 width=1267) (actual time=3.300..6.406 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1412
              ->  Limit  (cost=488.25..2149.88 rows=51 width=1267) (actual time=3.294..6.276 rows=51 loops=1)
                    Buffers: shared hit=1412
                    ->  Merge Append  (cost=488.25..4235.07 rows=115 width=1267) (actual time=3.290..6.254 rows=51 loops=1)
                          Sort Key: r.assignee_id, r.updated_at, r.id
                          Buffers: shared hit=1412
                          ->  Limit  (cost=0.42..3745.35 rows=51 width=1267) (actual time=0.099..3.012 rows=50 loops=1)
                                Buffers: shared hit=808
                                ->  Result  (cost=0.42..422295.89 rows=5751 width=1267) (actual time=0.099..2.996 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                      Buffers: shared hit=808
                                      ->  Index Scan Backward using records_list_assignee on records r  (cost=0.42..422295.89 rows=5751 width=1251) (actual time=0.086..2.943 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 727
                                            Buffers: shared hit=808
                          ->  Limit  (cost=391.65..391.78 rows=51 width=1267) (actual time=2.648..2.653 rows=2 loops=1)
                                Buffers: shared hit=523
                                ->  Sort  (cost=391.65..391.91 rows=103 width=1267) (actual time=2.645..2.649 rows=2 loops=1)
                                      Sort Key: r_1.assignee_id, r_1.updated_at, r_1.id
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=523
                                      ->  Result  (cost=0.42..388.21 rows=103 width=1267) (actual time=0.034..1.984 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=523
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..388.21 rows=103 width=1251) (actual time=0.029..1.714 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                          ->  Sort  (cost=51.65..51.68 rows=12 width=1267) (actual time=0.369..0.372 rows=1 loops=1)
                                Sort Key: r_2.assignee_id, r_2.updated_at, r_2.id
                                Sort Method: quicksort  Memory: 127kB
                                Buffers: shared hit=56
                                ->  Limit  (cost=0.42..51.44 rows=12 width=1267) (actual time=0.060..0.256 rows=51 loops=1)
                                      Buffers: shared hit=56
                                      ->  Result  (cost=0.42..51.44 rows=12 width=1267) (actual time=0.056..0.241 rows=51 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=56
                                            ->  Index Scan Backward using records_list_assignee on records r_2  (cost=0.42..51.44 rows=12 width=1251) (actual time=0.040..0.203 rows=51 loops=1)
                                                  Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=56
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1267) (actual time=0.164..0.173 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1267) (actual time=0.163..0.172 rows=1 loops=1)
                                      Sort Key: r_3.assignee_id, r_3.updated_at, r_3.id
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1267) (actual time=0.116..0.158 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.109..0.149 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.080..0.087 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.055..0.070 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.042..0.045 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.019 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.017..0.017 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.674 ms
Execution Time: 7.783 ms
```

### admin list, filter[state][in]=new,in_progress, sort=-updated_at

Request: `GET /api/v1/records/ticket?filter%5Bstate%5D%5Bin%5D=new,in_progress&sort=-updated_at` as `admin-large`

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
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.032..0.034 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.010..0.010 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.447 ms
Execution Time: 0.110 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.044..0.048 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.032..0.035 rows=1 loops=1)
        Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.207 ms
Execution Time: 0.086 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.067..0.069 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.065..0.066 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.052..0.056 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.042..0.045 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.026..0.027 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.011..0.013 rows=1 loops=1)
                          Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.382 ms
Execution Time: 0.120 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND r.state = ANY($7::text[]) LIMIT 1001) c
```

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc",["6851538a-9e01-4034-bcc1-7b2a36167d92"],["new","in_progress"]]`

```
Aggregate  (cost=91.15..91.16 rows=1 width=8) (actual time=1.396..1.399 rows=1 loops=1)
  Buffers: shared hit=532
  ->  Limit  (cost=0.42..78.64 rows=1001 width=4) (actual time=0.069..1.257 rows=1001 loops=1)
        Buffers: shared hit=532
        ->  Result  (cost=0.42..1534.65 rows=19635 width=4) (actual time=0.068..1.082 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
              Buffers: shared hit=532
              ->  Index Only Scan using records_list_state on records r  (cost=0.42..1534.65 rows=19635 width=0) (actual time=0.057..0.836 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid) AND (state = ANY ('{new,in_progress}'::text[])))
                    Heap Fetches: 0
                    Buffers: shared hit=532
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.345 ms
Execution Time: 1.461 ms
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

Arguments: `["4606c1e2-190f-45b2-8488-2382187661d2","7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc","0cc61b3d-7e0a-43fb-b951-dafb9f0dc0cc",["6851538a-9e01-4034-bcc1-7b2a36167d92"],["new","in_progress"],51]`

```
Limit  (cost=1.00..1112.85 rows=51 width=1293) (actual time=0.336..3.847 rows=51 loops=1)
  Buffers: shared hit=635
  ->  Result  (cost=1.00..428063.21 rows=19635 width=1293) (actual time=0.334..3.822 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
        Buffers: shared hit=635
        ->  Nested Loop Left Join  (cost=1.00..427915.95 rows=19635 width=1266) (actual time=0.310..3.574 rows=51 loops=1)
              Buffers: shared hit=635
              ->  Nested Loop  (cost=0.70..419765.95 rows=19635 width=1258) (actual time=0.273..2.826 rows=51 loops=1)
                    Buffers: shared hit=482
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..419512.22 rows=19635 width=1251) (actual time=0.236..2.676 rows=51 loops=1)
                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (entity_definition_id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid))
                          Filter: (state = ANY ('{new,in_progress}'::text[]))
                          Rows Removed by Filter: 419
                          Buffers: shared hit=479
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.023..0.026 rows=1 loops=1)
                                Index Cond: (tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                                Filter: (id = '7ab64858-d6a5-4b1f-bcee-4ed0cdd9bca2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.41 rows=1 width=40) (actual time=0.013..0.013 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.41 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                          Index Cond: ((tenant_id = '4606c1e2-190f-45b2-8488-2382187661d2'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.013 ms
Execution Time: 3.985 ms
```

### small tenant: agent list, page 1

Request: `GET /api/v1/records/ticket` as `agent-s1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.117..0.121 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.081..0.084 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.023..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.648 ms
Execution Time: 0.254 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.029..0.031 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.021..0.023 rows=1 loops=1)
        Index Cond: (tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.119 ms
Execution Time: 0.056 ms
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

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.063..0.065 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.062..0.063 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.048..0.052 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.039..0.042 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.023..0.024 rows=1 loops=1)
                          Index Cond: (tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.011..0.012 rows=1 loops=1)
                          Index Cond: (tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.301 ms
Execution Time: 0.112 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","4a7f62c5-288e-46e8-a4ac-d2a366bac508",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],"4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"]]`

```
Aggregate  (cost=34.43..34.44 rows=1 width=8) (actual time=8.815..8.827 rows=1 loops=1)
  Buffers: shared hit=4712
  ->  Limit  (cost=0.55..34.38 rows=4 width=4) (actual time=0.100..8.768 rows=179 loops=1)
        Buffers: shared hit=4712
        ->  Result  (cost=0.55..34.38 rows=4 width=4) (actual time=0.098..8.720 rows=179 loops=1)
              Buffers: shared hit=4712
              ->  Append  (cost=0.55..34.34 rows=4 width=0) (actual time=0.097..8.664 rows=179 loops=1)
                    Buffers: shared hit=4712
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.59 rows=1 width=0) (actual time=0.096..0.433 rows=50 loops=1)
                          Buffers: shared hit=20
                          ->  Limit  (cost=0.55..4.58 rows=1 width=4) (actual time=0.095..0.420 rows=50 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.55..4.58 rows=1 width=4) (actual time=0.095..0.410 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..4.58 rows=1 width=0) (actual time=0.085..0.387 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid) AND (assignment_group_id IS NOT NULL))
                                            Filter: (assignment_group_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))
                                            Rows Removed by Filter: 953
                                            Heap Fetches: 0
                                            Buffers: shared hit=20
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.42..8.46 rows=1 width=0) (actual time=0.044..1.268 rows=97 loops=1)
                          Buffers: shared hit=1013
                          ->  Limit  (cost=0.42..8.45 rows=1 width=4) (actual time=0.043..1.244 rows=97 loops=1)
                                Buffers: shared hit=1013
                                ->  Result  (cost=0.42..8.45 rows=1 width=4) (actual time=0.042..1.224 rows=97 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=1013
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.42..8.45 rows=1 width=0) (actual time=0.033..1.192 rows=97 loops=1)
                                            Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                                            Filter: ((requester_id IS NOT NULL) AND (requester_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 906
                                            Buffers: shared hit=1013
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..8.46 rows=1 width=0) (actual time=0.059..2.262 rows=32 loops=1)
                          Buffers: shared hit=1017
                          ->  Limit  (cost=0.42..8.46 rows=1 width=4) (actual time=0.058..2.202 rows=32 loops=1)
                                Buffers: shared hit=1017
                                ->  Result  (cost=0.42..8.46 rows=1 width=4) (actual time=0.056..2.189 rows=32 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=1017
                                      ->  Index Scan using records_list_requester on records r_2  (cost=0.42..8.46 rows=1 width=0) (actual time=0.042..2.160 rows=32 loops=1)
                                            Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                                            Filter: ((assignee_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND (assignee_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 971
                                            Buffers: shared hit=1017
                    ->  Subquery Scan on "*SELECT* 4"  (cost=0.72..12.80 rows=1 width=0) (actual time=4.644..4.646 rows=0 loops=1)
                          Buffers: shared hit=2662
                          ->  Limit  (cost=0.72..12.79 rows=1 width=4) (actual time=4.642..4.644 rows=0 loops=1)
                                Buffers: shared hit=2662
                                ->  Result  (cost=0.72..12.79 rows=1 width=4) (actual time=4.640..4.641 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=2662
                                      ->  Nested Loop Semi Join  (cost=0.72..12.79 rows=1 width=0) (actual time=4.624..4.625 rows=0 loops=1)
                                            Buffers: shared hit=2662
                                            ->  Index Scan using records_list_assignee on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.038..1.922 rows=824 loops=1)
                                                  Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 179
                                                  Buffers: shared hit=1013
                                            ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.32 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=824)
                                                  Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (record_id = r_3.id))
                                                  Filter: (((principal_kind = 'user'::text) AND (principal_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 0
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1649
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.518 ms
Execution Time: 8.968 ms
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

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","4a7f62c5-288e-46e8-a4ac-d2a366bac508",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],"4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98","4e83828a-ef25-9e19-bf97-84874ff6db98",["31353196-ebd2-353a-8c19-27c9aa090ef7","3164b0ca-6799-9372-d157-4acf299e68e8","beb7d466-3b9e-a13a-6683-dc4098b85e07","e0fd45c7-93cf-4ed2-bc3b-9f34dd669069"],51]`

```
Nested Loop Left Join  (cost=2.62..54.99 rows=1 width=1301) (actual time=4.646..9.378 rows=51 loops=1)
  Buffers: shared hit=3859
  ->  Nested Loop  (cost=2.32..46.66 rows=1 width=1266) (actual time=4.619..8.305 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3706
        ->  Subquery Scan on x  (cost=2.04..38.34 rows=1 width=1259) (actual time=4.598..7.590 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=3553
              ->  Limit  (cost=2.04..38.25 rows=4 width=1259) (actual time=4.586..7.365 rows=51 loops=1)
                    Buffers: shared hit=3553
                    ->  Merge Append  (cost=2.04..38.25 rows=4 width=1259) (actual time=4.584..7.339 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=3553
                          ->  Limit  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.099..0.803 rows=14 loops=1)
                                Buffers: shared hit=297
                                ->  Result  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.099..0.797 rows=14 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=297
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.091..0.773 rows=14 loops=1)
                                            Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[])))
                                            Rows Removed by Filter: 278
                                            Buffers: shared hit=297
                          ->  Limit  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.020..1.545 rows=30 loops=1)
                                Buffers: shared hit=294
                                ->  Result  (cost=0.42..8.45 rows=1 width=1259) (actual time=0.019..1.533 rows=30 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=294
                                      ->  Index Scan using records_list_updated on records r_1  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.015..1.502 rows=30 loops=1)
                                            Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                                            Filter: ((requester_id IS NOT NULL) AND (requester_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 259
                                            Buffers: shared hit=294
                          ->  Limit  (cost=0.42..8.46 rows=1 width=1259) (actual time=0.048..0.521 rows=9 loops=1)
                                Buffers: shared hit=299
                                ->  Result  (cost=0.42..8.46 rows=1 width=1259) (actual time=0.048..0.517 rows=9 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=299
                                      ->  Index Scan using records_list_updated on records r_2  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.045..0.503 rows=9 loops=1)
                                            Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                                            Filter: ((assignee_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND (assignee_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                            Rows Removed by Filter: 285
                                            Buffers: shared hit=299
                          ->  Limit  (cost=0.72..12.79 rows=1 width=1259) (actual time=4.412..4.416 rows=0 loops=1)
                                Buffers: shared hit=2663
                                ->  Result  (cost=0.72..12.79 rows=1 width=1259) (actual time=4.410..4.413 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                                      Buffers: shared hit=2663
                                      ->  Nested Loop Semi Join  (cost=0.72..12.79 rows=1 width=1251) (actual time=4.408..4.410 rows=0 loops=1)
                                            Buffers: shared hit=2663
                                            ->  Index Scan using records_list_updated on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..2.075 rows=824 loops=1)
                                                  Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 179
                                                  Buffers: shared hit=1014
                                            ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.32 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=824)
                                                  Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (record_id = r_3.id))
                                                  Filter: (((principal_kind = 'user'::text) AND (principal_id = '4e83828a-ef25-9e19-bf97-84874ff6db98'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{31353196-ebd2-353a-8c19-27c9aa090ef7,3164b0ca-6799-9372-d157-4acf299e68e8,beb7d466-3b9e-a13a-6683-dc4098b85e07,e0fd45c7-93cf-4ed2-bc3b-9f34dd669069}'::uuid[]))))
                                                  Rows Removed by Filter: 0
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1649
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.007..0.007 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.145 ms
Execution Time: 9.587 ms
```

### small tenant: admin list, page 1

Request: `GET /api/v1/records/ticket` as `admin-s1`

Statement 2:

```sql
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version
```

Arguments: `["ticket"]`

```
Nested Loop  (cost=0.56..16.62 rows=1 width=601) (actual time=0.043..0.046 rows=1 loops=1)
  Join Filter: ((d.id = v.definition_id) AND (v.version = d.current_version))
  Buffers: shared hit=6
  ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=54) (actual time=0.029..0.030 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
  ->  Index Scan using entity_definition_versions_pkey on entity_definition_versions v  (cost=0.28..8.30 rows=1 width=1436) (actual time=0.008..0.008 rows=1 loops=1)
        Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.366 ms
Execution Time: 0.083 ms
```

Statement 3:

```sql
SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2
```

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","ticket"]`

```
Result  (cost=0.27..8.29 rows=1 width=16) (actual time=0.030..0.033 rows=1 loops=1)
  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
  Buffers: shared hit=3
  ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions  (cost=0.27..8.29 rows=1 width=16) (actual time=0.022..0.024 rows=1 loops=1)
        Index Cond: (tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
        Filter: (key = 'ticket'::text)
        Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.121 ms
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

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","ticket"]`

```
Limit  (cost=16.62..16.62 rows=1 width=505) (actual time=0.056..0.057 rows=1 loops=1)
  Buffers: shared hit=6
  ->  Sort  (cost=16.62..16.62 rows=1 width=505) (actual time=0.055..0.056 rows=1 loops=1)
        Sort Key: a.version DESC
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=6
        ->  Result  (cost=0.56..16.61 rows=1 width=505) (actual time=0.045..0.049 rows=1 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
              Buffers: shared hit=6
              ->  Nested Loop  (cost=0.56..16.61 rows=1 width=505) (actual time=0.038..0.041 rows=1 loops=1)
                    Join Filter: (d.id = a.entity_definition_id)
                    Buffers: shared hit=6
                    ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.27..8.29 rows=1 width=32) (actual time=0.025..0.026 rows=1 loops=1)
                          Index Cond: (tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                          Filter: (key = 'ticket'::text)
                          Buffers: shared hit=3
                    ->  Index Scan using acl_versions_pkey on acl_versions a  (cost=0.27..8.29 rows=1 width=521) (actual time=0.008..0.010 rows=1 loops=1)
                          Index Cond: (tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                          Buffers: shared hit=3
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.281 ms
Execution Time: 0.092 ms
```

Statement 5:

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND true LIMIT 1001) c
```

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","4a7f62c5-288e-46e8-a4ac-d2a366bac508","c8786566-0db2-45b2-be92-45e1be713386","c8786566-0db2-45b2-be92-45e1be713386","c8786566-0db2-45b2-be92-45e1be713386",["51ba2359-4b7e-4ffc-9558-eadabed5fe1f"]]`

```
Aggregate  (cost=4.46..4.47 rows=1 width=8) (actual time=0.667..0.668 rows=1 loops=1)
  Buffers: shared hit=21
  ->  Limit  (cost=0.42..4.44 rows=1 width=4) (actual time=0.043..0.566 rows=1001 loops=1)
        Buffers: shared hit=21
        ->  Result  (cost=0.42..4.44 rows=1 width=4) (actual time=0.041..0.436 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
              Buffers: shared hit=21
              ->  Index Only Scan using records_list_assignee on records r  (cost=0.42..4.44 rows=1 width=0) (actual time=0.033..0.252 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                    Heap Fetches: 0
                    Buffers: shared hit=21
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.226 ms
Execution Time: 0.703 ms
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

Arguments: `["09ace1be-919d-4ea1-80e3-e6b6cdf53da9","4a7f62c5-288e-46e8-a4ac-d2a366bac508","c8786566-0db2-45b2-be92-45e1be713386","c8786566-0db2-45b2-be92-45e1be713386","c8786566-0db2-45b2-be92-45e1be713386",["51ba2359-4b7e-4ffc-9558-eadabed5fe1f"],51]`

```
Limit  (cost=1.00..25.09 rows=1 width=1293) (actual time=0.082..0.715 rows=51 loops=1)
  Buffers: shared hit=361
  ->  Result  (cost=1.00..25.09 rows=1 width=1293) (actual time=0.080..0.658 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
        Buffers: shared hit=361
        ->  Nested Loop Left Join  (cost=1.00..25.08 rows=1 width=1266) (actual time=0.062..0.594 rows=51 loops=1)
              Buffers: shared hit=361
              ->  Nested Loop  (cost=0.70..16.75 rows=1 width=1258) (actual time=0.047..0.304 rows=51 loops=1)
                    Buffers: shared hit=208
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..8.45 rows=1 width=1251) (actual time=0.034..0.151 rows=51 loops=1)
                          Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (entity_definition_id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid))
                          Buffers: shared hit=55
                    ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: (tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                          Filter: (id = '4a7f62c5-288e-46e8-a4ac-d2a366bac508'::uuid)
                          Buffers: shared hit=153
              ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '09ace1be-919d-4ea1-80e3-e6b6cdf53da9'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.736 ms
Execution Time: 0.790 ms
```

### single GET, agent

Request: `GET /api/v1/records/ticket/3f94a2ac-38ee-af8f-f434-143a689a293d` as `agent-1`

### single GET, admin

Request: `GET /api/v1/records/ticket/7af47606-284d-7369-c4d9-ac496c0f82c8` as `admin-large`

### single GET, small tenant agent

Request: `GET /api/v1/records/ticket/c7df223c-cbb4-f467-776f-6fea3db8bf06` as `agent-s1`

### create, agent (1 client)

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)
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
BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
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

### create, agent

Request: `POST /api/v1/records/ticket` as `agent-1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)
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
BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
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

### create, admin

Request: `POST /api/v1/records/ticket` as `admin-large`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)
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
BEGIN; SELECT set_config('app.tenant_id', '4606c1e2-190f-45b2-8488-2382187661d2', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
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

### create, small tenant agent

Request: `POST /api/v1/records/ticket` as `agent-s1`

Statement 1 (write, not EXPLAINed):

```sql
BEGIN; SELECT set_config('app.tenant_id', '09ace1be-919d-4ea1-80e3-e6b6cdf53da9', true)
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
BEGIN; SELECT set_config('app.tenant_id', '09ace1be-919d-4ea1-80e3-e6b6cdf53da9', true)
```

Statement 5 (write, not EXPLAINed):

```sql
BATCH: SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight' FOR UPDATE ; SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 ; SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text, v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at FROM entity_definitions d JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id WHERE d.key = $1 AND v.version = d.current_version ; SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id WHERE d.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1
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

