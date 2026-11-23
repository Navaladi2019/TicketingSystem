
## 2026-11-21T11:37:25Z phase=setup secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### Closure build (live units only)

| Run | INSERT ms | rows |
| --- | --- | --- |
| 1 | 541.3 | 49321 |
| 2 | 508.4 | 49321 |
| 3 | 454.2 | 49321 |

## 2026-11-21T11:37:46Z phase=e-setup secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### SP-1e setup done: district mixed (900 tickets + 150,000 bench_other), uniform districts m = 1001 / 3000 / 10000, records_org_unit on the B1 key.

## 2026-11-21T11:38:14Z phase=x-setup secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### A2-12 (a) setup done: uniform m = 4500 (district pre 3338), bench_rare (500 over all leaves), bench_conc (20,000 in district pre 9101).

## 2026-11-21T11:38:54Z phase=x-check secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### A2-12 (a) equality vs reference (today's count + C1 page): page ids in order and capped count

| Set | agents: C3 / W0 / MC3 (of 20) | count-only: C3 W0 MC3 R1 |
| --- | --- | --- |
| area (|S| 10, m 220, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 220) |
| district (|S| 101, m 3379, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| district thin (|S| 101, m 600, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 600) |
| district hot (|S| 101, m 22996, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| region (|S| 1011, m 37930, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| root (|S| 10111, m 380048, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| district mixed (|S| 101, m 900, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 900) |
| uniform m=1001 (|S| 101, m 1001, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| uniform m=3000 (|S| 101, m 3000, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| uniform m=4500 (|S| 101, m 4500, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| uniform m=10000 (|S| 101, m 10000, N_e 400000, K* 4516) | 20 / 20 / 20 | true true true true (count 1001) |
| rare, district (|S| 101, m 6, N_e 500, K* 1000) | 0 / 0 / 0 | true true true true (count 6) |
| rare, region (|S| 1011, m 46, N_e 500, K* 1000) | 0 / 0 / 0 | true true true true (count 46) |
| rare, root (|S| 10111, m 500, N_e 500, K* 1000) | 0 / 0 / 0 | true true true true (count 500) |
| conc, district (|S| 101, m 0, N_e 20000, K* 1009) | 0 / 0 / 0 | true true true true (count 0) |
| conc, region (|S| 1011, m 0, N_e 20000, K* 1009) | 0 / 0 / 0 | true true true true (count 0) |
| conc, regions 0-8 (|S| 9099, m 0, N_e 20000, K* 1009) | 0 / 0 / 0 | true true true true (count 0) |

## 2026-11-21T11:40:16Z phase=x-plans secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### A2-12 (a) diagnostic EXPLAIN, count-only caller

| case | plan / exec ms | region branch ms | region branch (or statement) plan | records read / removed |
| --- | --- | --- | --- | --- |
| probe P3e K*, rare, district (|S| 101, m 6, N_e 500, K* 1000) | 2.92 / 1.65 | 1.45 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Only Scan records_org_unit | 0 / 0 |
| CC count, rare, district (|S| 101, m 6, N_e 500, K* 1000) | 3.68 / 2.01 | 1.22 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey | 0 / 0 |
| page C3, rare, district (|S| 101, m 6, N_e 500, K* 1000) | 3.46 / 1.26 | 0.42 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey > Sort > CTE Scan | 6 / 0 |
| R1 W0 count, rare, district (|S| 101, m 6, N_e 500, K* 1000) | 0.87 / 1.79 | 1.60 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_list_created > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 500 / 494 |
| R1 W0 page, rare, district (|S| 101, m 6, N_e 500, K* 1000) | 1.38 / 2.21 | 1.76 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 500 / 494 |
| probe P3e K*, rare, region (|S| 1011, m 46, N_e 500, K* 1000) | 0.38 / 4.35 | 4.30 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Only Scan records_org_unit | 0 / 0 |
| CC count, rare, region (|S| 1011, m 46, N_e 500, K* 1000) | 1.07 / 4.03 | 3.78 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey | 0 / 0 |
| page C3, rare, region (|S| 1011, m 46, N_e 500, K* 1000) | 1.29 / 6.27 | 5.17 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey > Sort > CTE Scan | 46 / 0 |
| R1 W0 count, rare, region (|S| 1011, m 46, N_e 500, K* 1000) | 2.12 / 3.94 | 3.53 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_list_created > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 500 / 454 |
| R1 W0 page, rare, region (|S| 1011, m 46, N_e 500, K* 1000) | 1.33 / 2.25 | 1.68 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 500 / 454 |
| probe P3e K*, rare, root (|S| 10111, m 500, N_e 500, K* 1000) | 0.23 / 23.86 | 23.82 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Only Scan records_org_unit | 0 / 0 |
| CC count, rare, root (|S| 10111, m 500, N_e 500, K* 1000) | 0.96 / 28.03 | 27.53 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey | 0 / 0 |
| page C3, rare, root (|S| 10111, m 500, N_e 500, K* 1000) | 1.58 / 17.70 | 16.93 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey > Sort > CTE Scan | 500 / 0 |
| R1 W0 count, rare, root (|S| 10111, m 500, N_e 500, K* 1000) | 0.94 / 2.65 | 2.24 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_list_created > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 500 / 0 |
| R1 W0 page, rare, root (|S| 10111, m 500, N_e 500, K* 1000) | 2.25 / 1.38 | 0.49 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 51 / 0 |
| probe P3e K*, conc, district (|S| 101, m 0, N_e 20000, K* 1009) | 0.25 / 0.42 | 0.39 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Only Scan records_org_unit | 0 / 0 |
| CC count, conc, district (|S| 101, m 0, N_e 20000, K* 1009) | 1.07 / 0.70 | 0.49 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_org_unit > Result [never executed] > Index Only Scan record_shares_pkey [never executed] | 0 / 0 |
| page C3, conc, district (|S| 101, m 0, N_e 20000, K* 1009) | 1.42 / 0.53 | 0.31 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Sort > CTE Scan | 0 / 0 |
| R1 W0 count, conc, district (|S| 101, m 0, N_e 20000, K* 1009) | 0.95 / 78.95 | 78.77 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_org_unit > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 20000 / 20000 |
| R1 W0 page, conc, district (|S| 101, m 0, N_e 20000, K* 1009) | 1.53 / 61.96 | 61.59 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 20000 / 20000 |
| probe P3e K*, conc, region (|S| 1011, m 0, N_e 20000, K* 1009) | 0.26 / 3.03 | 2.98 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Only Scan records_org_unit | 0 / 0 |
| CC count, conc, region (|S| 1011, m 0, N_e 20000, K* 1009) | 0.99 / 3.51 | 3.33 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_org_unit > Result [never executed] > Index Only Scan record_shares_pkey [never executed] | 0 / 0 |
| page C3, conc, region (|S| 1011, m 0, N_e 20000, K* 1009) | 1.37 / 2.13 | 1.91 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Sort > CTE Scan | 0 / 0 |
| R1 W0 count, conc, region (|S| 1011, m 0, N_e 20000, K* 1009) | 0.93 / 64.22 | 64.03 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_org_unit > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 20000 / 20000 |
| R1 W0 page, conc, region (|S| 1011, m 0, N_e 20000, K* 1009) | 1.41 / 67.31 | 67.03 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 20000 / 20000 |
| probe P3e K*, conc, regions 0-8 (|S| 9099, m 0, N_e 20000, K* 1009) | 0.28 / 27.01 | 26.96 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Only Scan records_org_unit | 0 / 0 |
| CC count, conc, regions 0-8 (|S| 9099, m 0, N_e 20000, K* 1009) | 1.58 / 27.97 | 27.75 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_org_unit > Result [never executed] > Index Only Scan record_shares_pkey [never executed] | 0 / 0 |
| page C3, conc, regions 0-8 (|S| 9099, m 0, N_e 20000, K* 1009) | 1.55 / 16.65 | 16.40 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Sort > CTE Scan | 0 / 0 |
| R1 W0 count, conc, regions 0-8 (|S| 9099, m 0, N_e 20000, K* 1009) | 1.06 / 288.20 | 288.00 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_org_unit > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result [never executed] > Index Only Scan record_shares_pkey [never executed] | 20000 / 20000 |
| R1 W0 page, conc, regions 0-8 (|S| 9099, m 0, N_e 20000, K* 1009) | 1.49 / 319.42 | 319.15 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result [never executed] > Index Only Scan record_shares_pkey [never executed] | 20000 / 20000 |

## 2026-11-21T11:40:32Z phase=x-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (A2-12 (a) start): `SELECT 1`, 8 clients x 200: p50 1.21, p95 2.23 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### A2-12 (a) paired wall-clock, 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, same type, no region leaf)

| Set | mode | caller | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** | foreign containers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| uniform m=1001 (|S| 101, m 1001, N_e 400000, K* 4516) | C3 | count-only | 12.78 | 22.36 | 21.90 | 34.59 | **+9.46** | **+12.23** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=1001 (|S| 101, m 1001, N_e 400000, K* 4516) | W0 | count-only | 14.85 | 36.45 | 76.40 | 128.97 | **+61.52** | **+92.52** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=3000 (|S| 101, m 3000, N_e 400000, K* 4516) | C3 | count-only | 27.45 | 48.88 | 55.73 | 81.61 | **+28.07** | **+32.73** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=3000 (|S| 101, m 3000, N_e 400000, K* 4516) | W0 | count-only | 21.04 | 53.33 | 61.90 | 99.00 | **+37.86** | **+45.67** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=4500 (|S| 101, m 4500, N_e 400000, K* 4516) | C3 | count-only | 17.04 | 28.21 | 49.94 | 68.25 | **+31.80** | **+40.04** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=4500 (|S| 101, m 4500, N_e 400000, K* 4516) | W0 | count-only | 14.69 | 25.90 | 33.92 | 52.48 | **+19.06** | **+26.58** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=10000 (|S| 101, m 10000, N_e 400000, K* 4516) | C3 | count-only | 13.97 | 20.81 | 52.23 | 67.32 | **+38.35** | **+46.51** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=10000 (|S| 101, m 10000, N_e 400000, K* 4516) | W0 | count-only | 12.00 | 21.22 | 25.44 | 37.23 | **+12.50** | **+16.00** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |

Floor gate (A2-12 (a) end): `SELECT 1`, 8 clients x 200: p50 1.11, p95 1.96 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T11:43:48Z phase=x-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (A2-12 (a) start): `SELECT 1`, 8 clients x 200: p50 1.09, p95 1.68 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### A2-12 (a) paired wall-clock, 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, same type, no region leaf)

| Set | mode | caller | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** | foreign containers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (|S| 10, m 220, N_e 400000, K* 4516) | MC3 | count-only | 11.44 | 19.23 | 17.61 | 28.13 | **+6.16** | **+8.90** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district (|S| 101, m 3379, N_e 400000, K* 4516) | MC3 | count-only | 13.16 | 19.52 | 31.96 | 40.79 | **+18.60** | **+21.27** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district thin (|S| 101, m 600, N_e 400000, K* 4516) | MC3 | count-only | 11.72 | 19.38 | 20.83 | 29.98 | **+8.00** | **+10.60** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district hot (|S| 101, m 22996, N_e 400000, K* 4516) | MC3 | count-only | 11.23 | 18.47 | 22.78 | 32.07 | **+10.79** | **+13.60** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| region (|S| 1011, m 37930, N_e 400000, K* 4516) | MC3 | count-only | 11.64 | 19.74 | 23.29 | 33.94 | **+10.56** | **+14.20** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| root (|S| 10111, m 380048, N_e 400000, K* 4516) | MC3 | count-only | 11.72 | 20.29 | 20.29 | 31.18 | **+8.31** | **+10.90** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district mixed (|S| 101, m 900, N_e 400000, K* 4516) | MC3 | count-only | 11.71 | 16.14 | 22.03 | 29.96 | **+10.33** | **+13.83** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=1001 (|S| 101, m 1001, N_e 400000, K* 4516) | MC3 | count-only | 11.87 | 17.19 | 22.61 | 31.41 | **+10.32** | **+14.22** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=3000 (|S| 101, m 3000, N_e 400000, K* 4516) | MC3 | count-only | 11.93 | 16.42 | 34.57 | 46.95 | **+22.50** | **+30.52** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=4500 (|S| 101, m 4500, N_e 400000, K* 4516) | MC3 | count-only | 12.57 | 19.90 | 43.13 | 55.75 | **+30.85** | **+35.85** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=10000 (|S| 101, m 10000, N_e 400000, K* 4516) | MC3 | count-only | 13.49 | 47.77 | 31.79 | 75.39 | **+18.42** | **+27.63** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |

Floor gate (A2-12 (a) end): `SELECT 1`, 8 clients x 200: p50 3.25, p95 7.76 ms -> **REFUSED: not scored**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T11:46:36Z phase=x-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (A2-12 (a) start): `SELECT 1`, 8 clients x 200: p50 3.17, p95 6.30 ms -> **REFUSED: not scored**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

**A2-12 (a): not scored** (floor gate refused). Stopped.

## 2026-11-21T11:46:59Z phase=x-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (A2-12 (a) start): `SELECT 1`, 8 clients x 200: p50 1.32, p95 2.44 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### A2-12 (a) paired wall-clock, 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, same type, no region leaf)

| Set | mode | caller | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** | foreign containers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (|S| 10, m 220, N_e 400000, K* 4516) | MC3 | count-only | 10.88 | 19.02 | 17.45 | 30.49 | **+6.19** | **+11.47** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district (|S| 101, m 3379, N_e 400000, K* 4516) | MC3 | count-only | 11.32 | 19.55 | 30.71 | 42.79 | **+19.22** | **+23.23** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district thin (|S| 101, m 600, N_e 400000, K* 4516) | MC3 | count-only | 11.74 | 21.74 | 21.32 | 35.20 | **+8.66** | **+13.46** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district hot (|S| 101, m 22996, N_e 400000, K* 4516) | MC3 | count-only | 10.96 | 18.68 | 21.98 | 31.80 | **+10.82** | **+13.12** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| region (|S| 1011, m 37930, N_e 400000, K* 4516) | MC3 | count-only | 11.53 | 19.56 | 21.69 | 34.04 | **+9.70** | **+14.48** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| root (|S| 10111, m 380048, N_e 400000, K* 4516) | MC3 | count-only | 11.90 | 20.44 | 20.63 | 33.57 | **+8.43** | **+13.14** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district mixed (|S| 101, m 900, N_e 400000, K* 4516) | MC3 | count-only | 11.97 | 20.34 | 22.52 | 32.85 | **+9.93** | **+12.52** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=1001 (|S| 101, m 1001, N_e 400000, K* 4516) | MC3 | count-only | 11.38 | 19.12 | 21.23 | 31.71 | **+9.73** | **+12.60** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=3000 (|S| 101, m 3000, N_e 400000, K* 4516) | MC3 | count-only | 12.67 | 19.16 | 31.21 | 41.84 | **+18.84** | **+22.67** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=4500 (|S| 101, m 4500, N_e 400000, K* 4516) | MC3 | count-only | 12.54 | 19.57 | 38.05 | 46.73 | **+25.45** | **+27.16** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=10000 (|S| 101, m 10000, N_e 400000, K* 4516) | MC3 | count-only | 11.62 | 20.01 | 27.17 | 39.96 | **+15.20** | **+19.95** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |

Floor gate (A2-12 (a) end): `SELECT 1`, 8 clients x 200: p50 1.08, p95 1.71 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T11:49:30Z phase=x-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (A2-12 (a) start): `SELECT 1`, 8 clients x 200: p50 1.14, p95 2.23 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### A2-12 (a) paired wall-clock, 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, same type, no region leaf)

| Set | mode | caller | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** | foreign containers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (|S| 10, m 220, N_e 400000, K* 4516) | MC3 | agents | 15.37 | 24.13 | 19.02 | 29.18 | **+3.61** | **+5.05** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district (|S| 101, m 3379, N_e 400000, K* 4516) | MC3 | agents | 20.33 | 69.82 | 40.34 | 92.28 | **+20.28** | **+22.46** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district thin (|S| 101, m 600, N_e 400000, K* 4516) | MC3 | agents | 21.20 | 34.16 | 30.28 | 46.66 | **+9.05** | **+12.50** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district hot (|S| 101, m 22996, N_e 400000, K* 4516) | MC3 | agents | 22.97 | 35.27 | 31.25 | 44.45 | **+8.33** | **+9.18** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| region (|S| 1011, m 37930, N_e 400000, K* 4516) | MC3 | agents | 20.09 | 33.49 | 26.68 | 44.09 | **+6.30** | **+10.60** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| root (|S| 10111, m 380048, N_e 400000, K* 4516) | MC3 | agents | 18.29 | 28.63 | 23.54 | 38.29 | **+4.85** | **+9.66** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district mixed (|S| 101, m 900, N_e 400000, K* 4516) | MC3 | agents | 20.97 | 34.44 | 31.49 | 47.17 | **+9.42** | **+12.74** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=1001 (|S| 101, m 1001, N_e 400000, K* 4516) | MC3 | agents | 19.66 | 33.79 | 28.15 | 47.28 | **+8.39** | **+13.50** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=3000 (|S| 101, m 3000, N_e 400000, K* 4516) | MC3 | agents | 17.42 | 28.91 | 35.16 | 53.09 | **+18.14** | **+24.18** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=4500 (|S| 101, m 4500, N_e 400000, K* 4516) | MC3 | agents | 16.57 | 23.40 | 39.05 | 48.97 | **+22.57** | **+25.57** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=10000 (|S| 101, m 10000, N_e 400000, K* 4516) | MC3 | agents | 16.12 | 32.13 | 22.05 | 39.53 | **+5.24** | **+7.40** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |

Floor gate (A2-12 (a) end): `SELECT 1`, 8 clients x 200: p50 1.17, p95 2.26 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T11:52:46Z phase=x-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (A2-12 (a) start): `SELECT 1`, 8 clients x 200: p50 1.14, p95 2.54 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### A2-12 (a) paired wall-clock, 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, same type, no region leaf)

| Set | mode | caller | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** | foreign containers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| rare, district (|S| 101, m 6, N_e 500, K* 1000) | MC3 | count-only | 27.83 | 152.18 | 37.80 | 180.12 | **+9.28** | **+27.94** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| rare, district (|S| 101, m 6, N_e 500, K* 1000) | R1 | count-only | 25.24 | 50.08 | 32.97 | 60.61 | **+6.79** | **+10.53** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| rare, region (|S| 1011, m 46, N_e 500, K* 1000) | MC3 | count-only | 13.07 | 24.91 | 29.38 | 49.77 | **+15.22** | **+24.86** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| rare, region (|S| 1011, m 46, N_e 500, K* 1000) | R1 | count-only | 12.32 | 24.43 | 20.41 | 33.17 | **+7.08** | **+8.75** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| rare, root (|S| 10111, m 500, N_e 500, K* 1000) | MC3 | count-only | 17.43 | 67.33 | 89.42 | 190.15 | **+71.06** | **+122.82** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| rare, root (|S| 10111, m 500, N_e 500, K* 1000) | R1 | count-only | 17.22 | 48.01 | 25.96 | 63.45 | **+7.65** | **+15.44** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| conc, district (|S| 101, m 0, N_e 20000, K* 1009) | MC3 | count-only | 11.25 | 17.87 | 16.40 | 26.32 | **+5.00** | **+8.45** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| conc, region (|S| 1011, m 0, N_e 20000, K* 1009) | MC3 | count-only | 11.91 | 20.63 | 23.14 | 33.00 | **+10.95** | **+12.37** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| conc, regions 0-8 (|S| 9099, m 0, N_e 20000, K* 1009) | MC3 | count-only | 34.36 | 92.58 | 136.03 | 244.20 | **+93.85** | **+151.62** | `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |

Floor gate (A2-12 (a) end): `SELECT 1`, 8 clients x 200: p50 1.68, p95 2.75 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T11:57:27Z phase=x-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (A2-12 (a) start): `SELECT 1`, 8 clients x 200: p50 1.66, p95 2.79 ms -> **PASS**. Foreign containers: `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### A2-12 (a) paired wall-clock, 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, same type, no region leaf)

| Set | mode | caller | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** | foreign containers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| district (|S| 101, m 3379, N_e 400000, K* 3658) | MC3 | count-only | 12.25 | 43.06 | 35.44 | 90.54 | **+23.32** | **+47.48** | `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| district thin (|S| 101, m 600, N_e 400000, K* 3658) | MC3 | count-only | 11.84 | 21.62 | 22.09 | 40.26 | **+10.02** | **+18.63** | `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=1001 (|S| 101, m 1001, N_e 400000, K* 3658) | MC3 | count-only | 11.07 | 19.43 | 21.75 | 33.37 | **+10.12** | **+13.93** | `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=3000 (|S| 101, m 3000, N_e 400000, K* 3658) | MC3 | count-only | 13.73 | 29.75 | 38.41 | 98.68 | **+24.79** | **+68.93** | `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=4500 (|S| 101, m 4500, N_e 400000, K* 3658) | MC3 | count-only | 12.47 | 20.00 | 34.24 | 50.53 | **+21.96** | **+30.54** | `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |
| uniform m=10000 (|S| 101, m 10000, N_e 400000, K* 3658) | MC3 | count-only | 14.81 | 27.47 | 35.87 | 67.21 | **+20.66** | **+39.74** | `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1` |

Floor gate (A2-12 (a) end): `SELECT 1`, 8 clients x 200: p50 1.22, p95 2.31 ms -> **PASS**. Foreign containers: `tick1046-pg nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.
