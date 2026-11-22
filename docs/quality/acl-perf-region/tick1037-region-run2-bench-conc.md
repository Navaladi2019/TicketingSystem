# TICK-1037 region rows (A2-11)

## Sets (as each row ran)

| Set | Type | \|S\| | m | N̂_e | K* | Shape (A2-5) |
| --- | --- | --- | --- | --- | --- | --- |
| bench_conc, regions 0-8 (\|S\| 9,099) | bench_conc | 9099 | 0 | 20000 | 1000 | C3 (m <= K*) |
| bench_conc at root (\|S\| 10,111, info) | bench_conc | 10111 | 20000 | 20000 | 1000 | W0 (m > K*) |
| bench_conc, regions 0-8 (\|S\| 9,099), state=new (info) | bench_conc | 9099 | 0 | 20000 | 1000 | C3 (m <= K*) |
| bench_conc at root (\|S\| 10,111), state=new (info) | bench_conc | 10111 | 20000 | 20000 | 1000 | W0 (m > K*) |

## Rows: handler p95 against 150 ms, paired base beside it (1 client)

| Row | Attempt | n | p50 | **p95** | Base n | Base p50 | Base p95 | Δp95 vs base | Floor p50 before / after | Gate | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| region: bench_conc, regions 0-8 (\|S\| 9,099), agents | 1 | 794 | 56.44 | **70.08** | 793 | 16.59 | 22.51 | +47.57 | 0.54 / 0.55 |  | pass |
| region: bench_conc, regions 0-8 (\|S\| 9,099), count-only | 1 | 695 | 68.50 | **82.69** | 695 | 15.64 | 20.46 | +62.23 | 0.54 / 0.58 |  | pass |
| region: bench_conc at root (\|S\| 10,111, info), agents | 1 | 1366 | 25.97 | **33.14** | 1366 | 16.27 | 21.48 | +11.65 | 0.54 / 0.55 |  | info (not scored) |
| region: bench_conc at root (\|S\| 10,111, info), count-only | 1 | 1175 | 30.67 | **41.40** | 1175 | 18.01 | 25.12 | +16.28 | 0.81 / 0.64 |  | info (not scored) |
| region: bench_conc, regions 0-8 (\|S\| 9,099), state=new (info), agents | 1 | 623 | 76.19 | **101.09** | 623 | 14.93 | 23.09 | +78.00 | 0.59 / 0.57 |  | info (not scored) |
| region: bench_conc, regions 0-8 (\|S\| 9,099), state=new (info), count-only | 1 | 653 | 73.73 | **92.91** | 652 | 15.37 | 22.01 | +70.90 | 0.56 / 0.93 |  | info (not scored) |
| region: bench_conc at root (\|S\| 10,111), state=new (info), agents | 1 | 665 | 71.38 | **88.84** | 664 | 16.18 | 21.63 | +67.21 | 0.54 / 0.55 |  | info (not scored) |
| region: bench_conc at root (\|S\| 10,111), state=new (info), count-only | 1 | 661 | 71.63 | **86.74** | 660 | 16.79 | 22.19 | +64.55 | 0.55 / 0.57 |  | info (not scored) |

## Host

```
### 2026-11-22T12:46:15Z postgres:16.10
### host: MINGW64_NT-10.0-26300; docker 29.9.2
### docker VM: 16 CPUs, 26260877312 bytes, kernel 5.15.167.4-microsoft-standard-WSL2
### other containers at start: multica-backend-1 multica-frontend-1 multica-postgres-1 
```

Container sampler: 133 samples, 0 with a foreign container.

First start at 12:30:40Z: refused by the run-level floor gate (p50 4.15 ms > 3 ms; another task ran the full `go test -race ./...` suite on the host). No numbers were taken; rerun at 12:45:54Z when it had finished.

## Per-statement times (ms)

### region: bench_conc, regions 0-8 (|S| 9,099), agents, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 794 | 1.11 | 1.74 | 2.41 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 794 | 1.25 | 2.16 | 2.74 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 794 | 1.11 | 1.78 | 2.34 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 794 | 1.12 | 1.80 | 2.49 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 794 | 4.82 | 6.21 | 7.60 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 794 | 1.11 | 1.82 | 2.27 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 794 | 19.66 | 24.30 | 27.67 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 794 | 3.52 | 4.78 | 5.96 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 794 | 19.62 | 23.53 | 26.93 |
| 10 | `commit` | 794 | 1.14 | 1.82 | 2.28 |


### region: bench_conc, regions 0-8 (|S| 9,099), count-only, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 695 | 1.11 | 1.76 | 2.22 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 695 | 1.15 | 1.89 | 2.70 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 695 | 1.10 | 1.66 | 2.21 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 695 | 1.11 | 1.74 | 2.32 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 695 | 4.63 | 6.01 | 7.28 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 695 | 1.11 | 1.92 | 2.30 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 695 | 19.09 | 23.16 | 26.58 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 695 | 21.55 | 25.68 | 28.53 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 695 | 15.14 | 17.95 | 19.64 |
| 10 | `commit` | 695 | 1.23 | 2.17 | 2.69 |


### region: bench_conc at root (|S| 10,111, info), agents, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 1366 | 1.09 | 1.78 | 2.39 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1366 | 1.24 | 2.18 | 2.72 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1366 | 1.10 | 1.78 | 2.35 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1366 | 1.11 | 1.79 | 2.32 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 1366 | 4.82 | 6.08 | 7.36 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 1366 | 1.11 | 1.88 | 2.46 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 1366 | 1.76 | 2.72 | 3.76 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1366 | 3.00 | 4.11 | 4.95 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1366 | 7.30 | 9.69 | 11.39 |
| 10 | `commit` | 1366 | 1.11 | 1.73 | 2.36 |


### region: bench_conc at root (|S| 10,111, info), count-only, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 1175 | 1.13 | 1.81 | 2.53 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 1175 | 1.63 | 2.29 | 2.96 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 1175 | 1.12 | 1.92 | 2.51 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 1175 | 1.13 | 2.18 | 2.85 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 1175 | 5.13 | 7.01 | 8.01 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 1175 | 1.15 | 2.22 | 2.75 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 1175 | 2.17 | 3.27 | 3.99 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 1175 | 8.13 | 10.92 | 12.56 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 1175 | 5.41 | 7.99 | 9.60 |
| 10 | `commit` | 1175 | 1.14 | 1.88 | 2.52 |


### region: bench_conc, regions 0-8 (|S| 9,099), state=new (info), agents, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 623 | 1.12 | 1.89 | 2.81 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 623 | 1.43 | 2.30 | 3.59 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 623 | 1.11 | 2.00 | 2.91 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 623 | 1.11 | 2.17 | 3.01 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 623 | 4.95 | 7.03 | 9.22 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 623 | 1.12 | 2.24 | 3.47 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 623 | 19.96 | 26.88 | 34.86 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 623 | 23.39 | 30.91 | 42.42 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 623 | 18.58 | 24.45 | 30.24 |
| 10 | `commit` | 623 | 1.16 | 2.21 | 2.93 |


### region: bench_conc, regions 0-8 (|S| 9,099), state=new (info), count-only, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 653 | 1.12 | 2.09 | 2.92 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 653 | 1.51 | 2.35 | 3.24 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 653 | 1.12 | 2.13 | 2.79 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 653 | 1.12 | 2.20 | 3.39 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 653 | 5.02 | 6.85 | 8.06 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 653 | 1.14 | 2.24 | 2.97 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 653 | 20.22 | 25.75 | 30.65 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 653 | 23.16 | 28.47 | 34.51 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 653 | 16.25 | 19.77 | 22.04 |
| 10 | `commit` | 653 | 1.32 | 2.43 | 3.05 |


### region: bench_conc at root (|S| 10,111), state=new (info), agents, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 665 | 1.13 | 2.16 | 2.52 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 665 | 1.60 | 2.44 | 3.03 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 665 | 1.12 | 2.27 | 3.22 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 665 | 1.14 | 2.22 | 3.14 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 665 | 5.11 | 6.92 | 8.12 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 665 | 1.20 | 2.46 | 3.24 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 665 | 2.16 | 3.30 | 4.05 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 665 | 43.07 | 52.68 | 62.49 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 665 | 10.84 | 14.46 | 16.80 |
| 10 | `commit` | 665 | 1.22 | 2.45 | 3.19 |


### region: bench_conc at root (|S| 10,111), state=new (info), count-only, attempt 1

Round trips: 10, statements: 10, statements per batch: [] (one request, the first actor).

| # | Statement | n | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 1 | `BEGIN; SELECT set_config('app.tenant_id', '37c4ffa8-477d-454b-8e5f-3e048e23550e', true)` | 661 | 1.14 | 2.23 | 2.84 |
| 2 | `SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.s…` | 661 | 1.64 | 2.53 | 3.31 |
| 3 | `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2` | 661 | 1.14 | 2.29 | 2.92 |
| 4 | `SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity_definitions d JOIN …` | 661 | 1.18 | 2.34 | 2.94 |
| 5 | `WITH m AS ( SELECT org_unit_id FROM org_unit_members WHERE tenant_id = $1 AND principal_id…` | 661 | 5.25 | 6.74 | 8.32 |
| 6 | `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definitio…` | 661 | 1.36 | 2.44 | 3.12 |
| 7 | `SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM …` | 661 | 2.17 | 3.35 | 4.22 |
| 8 | `SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r…` | 661 | 42.98 | 52.35 | 64.41 |
| 9 | `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.nu…` | 661 | 10.41 | 13.61 | 15.83 |
| 10 | `commit` | 661 | 1.25 | 2.38 | 3.17 |

