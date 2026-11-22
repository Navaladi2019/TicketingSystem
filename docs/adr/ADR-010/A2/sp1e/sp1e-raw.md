
## 2026-11-21T06:41:23Z phase=setup secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### Closure build (live units only)

| Run | INSERT ms | rows |
| --- | --- | --- |
| 1 | 255.2 | 49321 |
| 2 | 337.8 | 49321 |
| 3 | 341.3 | 49321 |

## 2026-11-21T06:41:38Z phase=e-setup secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

## 2026-11-21T06:42:30Z phase=e-setup secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `sec-swap-1032 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### SP-1e setup done: district mixed (900 tickets + 150,000 bench_other), uniform districts m = 1001 / 3000 / 10000, records_org_unit on the B1 key.

## 2026-11-21T06:43:41Z phase=e-check secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `sec-swap-1032 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### SP-1e equality check vs reference (today's count + C1 page): page ids in order and capped count, 40 agents + the count-only caller per set

| Set | MC3 | ML1 | L1 everywhere | count-only caller (MC3, ML1, L1) | probe P3e (K+1 capped) |
| --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 220) | 220 |
| district (101 units, 3379 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 1001) | 1001 |
| district thin (101 units, 600 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 600) | 600 |
| district hot (101 units, 22996 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 1001) | 1001 |
| region (1011 units, 37930 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 1001) | 1001 |
| root (10111 units, 380048 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 1001) | 1001 |
| district mixed (101 units, 900 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 900) | 900 |
| uniform m=1001 (101 units, 1001 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 1001) | 1001 |
| uniform m=3000 (101 units, 3000 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 1001) | 1001 |
| uniform m=10000 (101 units, 10000 tickets) | 40/40 | 40/40 | 40/40 | true true true (count 1001) | 1001 |

## 2026-11-21T06:45:23Z phase=e-plans secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-it-pgb-noprep-423403037 nava-it-pgb-pooled-423403037 nava-it-pg-423403037 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### (4) mixed entity types, both index keys (diagnostic EXPLAIN, agents[0])

| case | plan / exec ms | region branch ms | region branch (or statement) plan | records read / removed |
| --- | --- | --- | --- | --- |
| old key: probe P3e K=1000, district mixed (101 units, 900 tickets) | 2.44 / 220.30 | 220.12 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Scan records_org_unit | 150894 / 149985 |
| old key: page C2, district mixed (101 units, 900 tickets) | 3.67 / 189.84 | 183.64 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 150900 / 150065 |
| old key: page C3, district mixed (101 units, 900 tickets) | 1.64 / 192.86 | 190.41 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 150900 / 150065 |
| B1 key: probe P3e K=1000, district mixed (101 units, 900 tickets) | 0.75 / 1.88 | 1.82 | Aggregate > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Index Only Scan records_org_unit | 909 / 0 |
| B1 key: page C2, district mixed (101 units, 900 tickets) | 2.64 / 8.50 | 4.94 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 900 / 65 |
| B1 key: page C3, district mixed (101 units, 900 tickets) | 2.00 / 9.40 | 6.62 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 900 / 65 |

### (7) <= K shapes: plan stability over n_distinct(records.org_unit_id) (diagnostic EXPLAIN, agents[0], records_ou_list present)

| case | plan / exec ms | region branch ms | region branch (or statement) plan | records read / removed |
| --- | --- | --- | --- | --- |
| n_distinct sampled (3080): page C2, area (10 units, 220 tickets) | 3.82 / 5.46 | 1.56 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 220 / 10 |
| n_distinct sampled (3080): page C3, area (10 units, 220 tickets) | 2.86 / 4.43 | 1.27 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 220 / 10 |
| n_distinct sampled (3080): page L1, area (10 units, 220 tickets) | 2.77 / 4.33 | 1.24 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 150 / 0 |
| n_distinct sampled (3080): page C2, district thin (101 units, 600 tickets) | 2.95 / 10.83 | 6.38 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 600 / 37 |
| n_distinct sampled (3080): page C3, district thin (101 units, 600 tickets) | 2.28 / 7.72 | 4.13 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 600 / 37 |
| n_distinct sampled (3080): page L1, district thin (101 units, 600 tickets) | 2.65 / 6.29 | 3.67 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 606 / 0 |
| n_distinct sampled (3080): page C2, district mixed (101 units, 900 tickets) | 2.50 / 10.18 | 7.18 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 900 / 65 |
| n_distinct sampled (3080): page C3, district mixed (101 units, 900 tickets) | 2.17 / 8.70 | 5.97 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 900 / 65 |
| n_distinct sampled (3080): page L1, district mixed (101 units, 900 tickets) | 3.11 / 8.05 | 5.27 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 606 / 0 |
| n_distinct 1000 (1000): page C2, area (10 units, 220 tickets) | 2.76 / 3.43 | 0.77 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 220 / 10 |
| n_distinct 1000 (1000): page C3, area (10 units, 220 tickets) | 1.77 / 2.80 | 0.76 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 220 / 10 |
| n_distinct 1000 (1000): page L1, area (10 units, 220 tickets) | 2.07 / 3.26 | 0.79 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 150 / 0 |
| n_distinct 1000 (1000): page C2, district thin (101 units, 600 tickets) | 1.96 / 6.41 | 3.76 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 600 / 37 |
| n_distinct 1000 (1000): page C3, district thin (101 units, 600 tickets) | 2.43 / 5.70 | 3.00 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 600 / 37 |
| n_distinct 1000 (1000): page L1, district thin (101 units, 600 tickets) | 2.12 / 5.54 | 3.45 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 606 / 0 |
| n_distinct 1000 (1000): page C2, district mixed (101 units, 900 tickets) | 2.25 / 7.31 | 4.41 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 900 / 65 |
| n_distinct 1000 (1000): page C3, district mixed (101 units, 900 tickets) | 2.12 / 8.05 | 5.74 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 900 / 65 |
| n_distinct 1000 (1000): page L1, district mixed (101 units, 900 tickets) | 2.12 / 6.92 | 4.21 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 606 / 0 |
| n_distinct 9000 (9000): page C2, area (10 units, 220 tickets) | 2.58 / 4.04 | 0.99 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 220 / 10 |
| n_distinct 9000 (9000): page C3, area (10 units, 220 tickets) | 2.23 / 3.59 | 0.99 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 220 / 10 |
| n_distinct 9000 (9000): page L1, area (10 units, 220 tickets) | 3.12 / 5.36 | 2.51 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Result > Index Only Scan record_shares_pkey | 150 / 0 |
| n_distinct 9000 (9000): page C2, district thin (101 units, 600 tickets) | 2.10 / 5.81 | 3.09 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 600 / 37 |
| n_distinct 9000 (9000): page C3, district thin (101 units, 600 tickets) | 2.46 / 5.54 | 3.02 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 600 / 37 |
| n_distinct 9000 (9000): page L1, district thin (101 units, 600 tickets) | 2.67 / 8.16 | 5.32 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Result > Index Only Scan record_shares_pkey | 606 / 0 |
| n_distinct 9000 (9000): page C2, district mixed (101 units, 900 tickets) | 2.32 / 7.97 | 4.81 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 900 / 65 |
| n_distinct 9000 (9000): page C3, district mixed (101 units, 900 tickets) | 2.32 / 8.44 | 5.49 | Limit > InitPlan(Result) > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal > Sort > CTE Scan | 900 / 65 |
| n_distinct 9000 (9000): page L1, district mixed (101 units, 900 tickets) | 2.54 / 11.51 | 7.82 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Result > Index Only Scan record_shares_pkey | 606 / 0 |

### (5) page above K, W0 vs L1, and (6) the count-only caller's CC count (diagnostic EXPLAIN)

| case | plan / exec ms | region branch ms | region branch (or statement) plan | records read / removed |
| --- | --- | --- | --- | --- |
| count-only caller: CC count, area (10 units, 220 tickets) | 2.20 / 1.21 | 0.69 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 220 / 0 |
| page W0, district (101 units, 3379 tickets) | 3.48 / 7.90 | 5.67 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 949 / 942 |
| page L1, district (101 units, 3379 tickets) | 2.63 / 19.29 | 16.11 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 1919 / 101 |
| count-only caller: CC count, district (101 units, 3379 tickets) | 2.14 / 4.22 | 3.15 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 984 / 0 |
| count-only caller: CC count, district thin (101 units, 600 tickets) | 1.58 / 3.05 | 2.46 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 606 / 0 |
| page W0, district hot (101 units, 22996 tickets) | 2.55 / 3.87 | 2.30 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 448 / 422 |
| page L1, district hot (101 units, 22996 tickets) | 2.29 / 11.09 | 9.31 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 1515 / 101 |
| count-only caller: CC count, district hot (101 units, 22996 tickets) | 1.60 / 3.42 | 2.57 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 992 / 0 |
| page W0, region (1011 units, 37930 tickets) | 1.96 / 3.48 | 1.88 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 329 / 298 |
| page L1, region (1011 units, 37930 tickets) | 2.12 / 139.61 | 137.39 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 19209 / 1011 |
| count-only caller: CC count, region (1011 units, 37930 tickets) | 1.75 / 3.99 | 3.08 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 988 / 0 |
| page W0, root (10111 units, 380048 tickets) | 2.14 / 2.61 | 1.35 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 52 / 2 |
| page L1, root (10111 units, 380048 tickets) | 2.57 / 1141.90 | 1140.74 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 181998 / 10111 |
| count-only caller: CC count, root (10111 units, 380048 tickets) | 1.63 / 3.10 | 2.38 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 1008 / 0 |
| count-only caller: CC count, district mixed (101 units, 900 tickets) | 1.18 / 3.54 | 2.88 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 909 / 0 |
| page W0, uniform m=1001 (101 units, 1001 tickets) | 1.68 / 5.53 | 4.08 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 1016 / 1012 |
| page L1, uniform m=1001 (101 units, 1001 tickets) | 1.68 / 6.22 | 4.13 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 1010 / 101 |
| count-only caller: CC count, uniform m=1001 (101 units, 1001 tickets) | 1.33 / 3.29 | 2.60 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 1010 / 0 |
| page W0, uniform m=3000 (101 units, 3000 tickets) | 1.72 / 4.52 | 3.13 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 763 / 759 |
| page L1, uniform m=3000 (101 units, 3000 tickets) | 1.75 / 12.66 | 10.75 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 3030 / 202 |
| count-only caller: CC count, uniform m=3000 (101 units, 3000 tickets) | 1.31 / 3.02 | 2.25 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 986 / 0 |
| page W0, uniform m=10000 (101 units, 10000 tickets) | 1.69 / 3.64 | 2.36 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 542 / 526 |
| page L1, uniform m=10000 (101 units, 10000 tickets) | 1.83 / 20.01 | 18.36 | Limit > Sort > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 4848 / 303 |
| count-only caller: CC count, uniform m=10000 (101 units, 10000 tickets) | 1.74 / 3.36 | 2.60 | Subquery Scan > Limit > Result > Nested Loop > Nested Loop > Function Scan > Index Only Scan org_unit_closure_pkey > Limit > Result > Nested Loop > Index Scan records_ou_list > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 1001 / 0 |
| count-only caller: today's W0 count (SP-1c), district (101 units, 3379 tickets) | 1.16 / 643.33 | 642.39 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_org_unit > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 181374 / 180373 |
| count-only caller: today's W0 count (SP-1c), uniform m=1001 (101 units, 1001 tickets) | 1.03 / 1050.20 | 1049.18 | Subquery Scan > Limit > Result > Nested Loop > Index Scan records_org_unit > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 375712 / 374711 |

### SP-1e plans

#### old key: probe P3e K=1000, district mixed (101 units, 900 tickets)

```
Aggregate  (cost=2675.60..2675.61 rows=1 width=8) (actual time=170.871..170.874 rows=1 loops=1)
  Buffers: shared hit=146992
  ->  Limit  (cost=0.85..2673.59 rows=161 width=4) (actual time=0.381..170.713 rows=900 loops=1)
        Buffers: shared hit=146992
        ->  Result  (cost=0.85..2673.59 rows=161 width=4) (actual time=0.380..170.599 rows=900 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
              Buffers: shared hit=146992
              ->  Nested Loop  (cost=0.85..2673.59 rows=161 width=0) (actual time=0.370..170.414 rows=900 loops=1)
                    Buffers: shared hit=146992
                    ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.036..0.189 rows=101 loops=1)
                          Buffers: shared hit=5
                          ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.008 rows=1 loops=1)
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.028..0.122 rows=101 loops=1)
                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Limit  (cost=0.42..380.82 rows=23 width=4) (actual time=0.793..1.682 rows=9 loops=101)
                          Buffers: shared hit=146987
                          ->  Result  (cost=0.42..380.82 rows=23 width=4) (actual time=0.792..1.680 rows=9 loops=101)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                Buffers: shared hit=146987
                                ->  Index Scan using records_org_unit on records r  (cost=0.42..380.82 rows=23 width=0) (actual time=0.787..1.674 rows=9 loops=101)
                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                      Filter: (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid)
                                      Rows Removed by Filter: 1485
                                      Buffers: shared hit=146987
Planning Time: 0.661 ms
Execution Time: 170.964 ms
```

#### old key: page C2, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=3926.88..4989.37 rows=1 width=1301) (actual time=197.560..199.345 rows=51 loops=1)
  Buffers: shared hit=148130
  ->  Nested Loop  (cost=3926.58..4981.04 rows=1 width=1266) (actual time=197.534..199.104 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=147977
        ->  Subquery Scan on x  (cost=3926.30..4972.73 rows=1 width=1259) (actual time=197.492..198.903 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=147824
              ->  Limit  (cost=3926.30..4971.58 rows=51 width=1259) (actual time=197.480..198.824 rows=51 loops=1)
                    Buffers: shared hit=147824
                    ->  Merge Append  (cost=3926.30..7390.06 rows=169 width=1259) (actual time=197.478..198.814 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=147824
                          ->  Limit  (cost=0.42..3194.81 rows=51 width=1259) (actual time=0.097..1.394 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..530330.99 rows=8467 width=1259) (actual time=0.097..1.387 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..530330.99 rows=8467 width=1251) (actual time=0.092..1.356 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..205.65 rows=51 width=1259) (actual time=0.019..0.024 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..511.28 rows=127 width=1259) (actual time=0.019..0.023 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..511.28 rows=127 width=1251) (actual time=0.016..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.89 rows=15 width=1259) (actual time=0.013..0.014 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.89 rows=15 width=1259) (actual time=0.013..0.014 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.89 rows=15 width=1251) (actual time=0.012..0.012 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.094 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.094 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.061..0.084 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.081 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.037..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=3880.23..3880.36 rows=51 width=1259) (actual time=197.254..197.261 rows=3 loops=1)
                                Buffers: shared hit=147005
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.019..0.045 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.018..0.033 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=3869.58..3870.09 rows=205 width=1259) (actual time=197.253..197.258 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=147005
                                      ->  Result  (cost=21.93..3862.74 rows=205 width=1259) (actual time=0.568..196.362 rows=835 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=147005
                                            ->  Nested Loop Anti Join  (cost=21.93..3862.74 rows=205 width=1251) (actual time=0.565..196.082 rows=835 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 2505
                                                  Buffers: shared hit=147005
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..3823.66 rows=205 width=1251) (actual time=0.523..195.098 rows=835 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Rows Removed by Filter: 150065
                                                        Buffers: shared hit=146992
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.028..0.035 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.025..0.031 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.583 ms
Execution Time: 199.542 ms
```

#### old key: page C3, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=3930.98..4993.47 rows=1 width=1000) (actual time=188.521..190.032 rows=51 loops=1)
  Buffers: shared hit=148130
  ->  Nested Loop  (cost=3930.68..4985.14 rows=1 width=965) (actual time=188.502..189.842 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=147977
        ->  Subquery Scan on x  (cost=3930.40..4976.83 rows=1 width=958) (actual time=188.477..189.659 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=147824
              ->  Limit  (cost=3930.40..4975.68 rows=51 width=958) (actual time=188.464..189.594 rows=51 loops=1)
                    Buffers: shared hit=147824
                    ->  Merge Append  (cost=3930.40..7394.16 rows=169 width=958) (actual time=188.463..189.582 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=147824
                          ->  Limit  (cost=0.42..3194.81 rows=51 width=1259) (actual time=0.092..1.173 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..530330.99 rows=8467 width=1259) (actual time=0.091..1.166 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..530330.99 rows=8467 width=1251) (actual time=0.087..1.144 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..205.65 rows=51 width=1259) (actual time=0.017..0.021 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..511.28 rows=127 width=1259) (actual time=0.017..0.020 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..511.28 rows=127 width=1251) (actual time=0.015..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.89 rows=15 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.89 rows=15 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.89 rows=15 width=1251) (actual time=0.012..0.012 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.096..0.103 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.095..0.102 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.064..0.087 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.062..0.084 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.049..0.054 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=3884.33..3884.46 rows=51 width=262) (actual time=188.240..188.248 rows=3 loops=1)
                                Buffers: shared hit=147005
                                CTE acl_m
                                  ->  Result  (cost=32.58..3873.39 rows=205 width=1259) (actual time=0.300..185.886 rows=835 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=147005
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.015..0.038 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.013..0.027 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..3862.74 rows=205 width=1251) (actual time=0.298..185.623 rows=835 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 2505
                                              Buffers: shared hit=147005
                                              ->  Index Scan using records_org_unit on records r_4  (cost=0.42..3823.66 rows=205 width=1251) (actual time=0.267..184.635 rows=835 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-26… (549 chars)
                                                    Rows Removed by Filter: 150065
                                                    Buffers: shared hit=146992
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.021..0.026 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.018..0.022 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=10.94..11.45 rows=205 width=262) (actual time=188.239..188.242 rows=3 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=147005
                                      ->  CTE Scan on acl_m  (cost=0.00..4.10 rows=205 width=262) (actual time=0.303..187.844 rows=835 loops=1)
                                            Buffers: shared hit=147005
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.656 ms
Execution Time: 190.315 ms
```

#### B1 key: probe P3e K=1000, district mixed (101 units, 900 tickets)

```
Aggregate  (cost=47.32..47.33 rows=1 width=8) (actual time=1.695..1.698 rows=1 loops=1)
  Buffers: shared hit=332
  ->  Limit  (cost=0.85..45.30 rows=161 width=4) (actual time=0.082..1.609 rows=900 loops=1)
        Buffers: shared hit=332
        ->  Result  (cost=0.85..45.30 rows=161 width=4) (actual time=0.081..1.487 rows=900 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
              Buffers: shared hit=332
              ->  Nested Loop  (cost=0.85..45.30 rows=161 width=0) (actual time=0.068..1.333 rows=900 loops=1)
                    Buffers: shared hit=332
                    ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.042..0.090 rows=101 loops=1)
                          Buffers: shared hit=5
                          ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.033..0.059 rows=101 loops=1)
                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Limit  (cost=0.42..5.35 rows=23 width=4) (actual time=0.007..0.011 rows=9 loops=101)
                          Buffers: shared hit=327
                          ->  Result  (cost=0.42..5.35 rows=23 width=4) (actual time=0.007..0.009 rows=9 loops=101)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                Buffers: shared hit=327
                                ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.35 rows=23 width=0) (actual time=0.005..0.006 rows=9 loops=101)
                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                      Heap Fetches: 0
                                      Buffers: shared hit=327
Planning Time: 0.440 ms
Execution Time: 1.771 ms
```

#### B1 key: page C2, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=1711.61..2793.32 rows=1 width=1301) (actual time=4.660..6.719 rows=51 loops=1)
  Buffers: shared hit=2077
  ->  Nested Loop  (cost=1711.31..2784.98 rows=1 width=1266) (actual time=4.641..6.504 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1924
        ->  Subquery Scan on x  (cost=1711.03..2776.67 rows=1 width=1259) (actual time=4.625..6.304 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1771
              ->  Limit  (cost=1711.03..2775.52 rows=51 width=1259) (actual time=4.616..6.228 rows=51 loops=1)
                    Buffers: shared hit=1771
                    ->  Merge Append  (cost=1711.03..5217.59 rows=168 width=1259) (actual time=4.614..6.214 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1771
                          ->  Limit  (cost=0.42..3236.77 rows=51 width=1259) (actual time=0.126..1.682 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..527778.30 rows=8317 width=1259) (actual time=0.126..1.673 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..527778.30 rows=8317 width=1251) (actual time=0.116..1.638 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..206.29 rows=51 width=1259) (actual time=0.062..0.070 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..742.82 rows=184 width=1259) (actual time=0.062..0.069 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..742.82 rows=184 width=1251) (actual time=0.058..0.063 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..62.11 rows=14 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..62.11 rows=14 width=1259) (actual time=0.029..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..62.11 rows=14 width=1251) (actual time=0.022..0.023 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.119..0.125 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.118..0.124 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.077..0.105 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.074..0.101 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.049..0.054 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1664.97..1665.09 rows=51 width=1259) (actual time=4.270..4.279 rows=3 loops=1)
                                Buffers: shared hit=952
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.014..0.046 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.013..0.029 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=1654.31..1654.80 rows=197 width=1259) (actual time=4.269..4.275 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=952
                                      ->  Result  (cost=21.93..1647.74 rows=197 width=1259) (actual time=0.121..3.591 rows=835 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=952
                                            ->  Nested Loop Anti Join  (cost=21.93..1647.74 rows=197 width=1251) (actual time=0.118..3.326 rows=835 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 2505
                                                  Buffers: shared hit=952
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1608.78 rows=197 width=1251) (actual time=0.084..2.349 rows=835 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 65
                                                        Buffers: shared hit=939
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.022..0.030 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.020..0.026 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.016..0.018 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.309 ms
Execution Time: 7.012 ms
```

#### B1 key: page C3, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=1715.55..2797.26 rows=1 width=998) (actual time=5.641..7.428 rows=51 loops=1)
  Buffers: shared hit=2077
  ->  Nested Loop  (cost=1715.25..2788.92 rows=1 width=963) (actual time=5.540..7.117 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1924
        ->  Subquery Scan on x  (cost=1714.97..2780.61 rows=1 width=956) (actual time=5.469..6.896 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1771
              ->  Limit  (cost=1714.97..2779.46 rows=51 width=956) (actual time=5.456..6.826 rows=51 loops=1)
                    Buffers: shared hit=1771
                    ->  Merge Append  (cost=1714.97..5221.53 rows=168 width=956) (actual time=5.454..6.814 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1771
                          ->  Limit  (cost=0.42..3236.77 rows=51 width=1259) (actual time=0.096..1.415 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..527778.30 rows=8317 width=1259) (actual time=0.096..1.398 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..527778.30 rows=8317 width=1251) (actual time=0.090..1.368 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..206.29 rows=51 width=1259) (actual time=0.019..0.026 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..742.82 rows=184 width=1259) (actual time=0.018..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..742.82 rows=184 width=1251) (actual time=0.016..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..62.11 rows=14 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..62.11 rows=14 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..62.11 rows=14 width=1251) (actual time=0.013..0.013 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.082..0.088 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.082..0.087 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.055..0.078 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.053..0.075 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.042 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1668.91..1669.03 rows=51 width=262) (actual time=5.235..5.242 rows=3 loops=1)
                                Buffers: shared hit=952
                                CTE acl_m
                                  ->  Result  (cost=32.58..1658.39 rows=197 width=1259) (actual time=0.109..3.157 rows=835 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=952
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.021..0.049 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.019..0.035 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..1647.74 rows=197 width=1251) (actual time=0.106..2.930 rows=835 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 2505
                                              Buffers: shared hit=952
                                              ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1608.78 rows=197 width=1251) (actual time=0.079..2.075 rows=835 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 65
                                                    Buffers: shared hit=939
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.022 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.019 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=10.51..11.00 rows=197 width=262) (actual time=5.233..5.235 rows=3 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=952
                                      ->  CTE Scan on acl_m  (cost=0.00..3.94 rows=197 width=262) (actual time=0.111..4.879 rows=835 loops=1)
                                            Buffers: shared hit=952
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.832 ms
Execution Time: 7.738 ms
```

#### n_distinct sampled (3080): page C2, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=1741.71..2842.70 rows=1 width=1301) (actual time=1.306..3.669 rows=51 loops=1)
  Buffers: shared hit=1397
  ->  Nested Loop  (cost=1741.41..2834.37 rows=1 width=1266) (actual time=1.286..3.400 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1244
        ->  Subquery Scan on x  (cost=1741.13..2826.06 rows=1 width=1259) (actual time=1.263..3.159 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1091
              ->  Limit  (cost=1741.13..2824.91 rows=51 width=1259) (actual time=1.257..3.063 rows=51 loops=1)
                    Buffers: shared hit=1091
                    ->  Merge Append  (cost=1741.13..5332.48 rows=169 width=1259) (actual time=1.252..3.041 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1091
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.135..1.876 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.135..1.865 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.123..1.822 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.025..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.025..0.029 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.020..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.119..0.130 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.119..0.125 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.069..0.099 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.067..0.094 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.044 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1695.06..1695.19 rows=51 width=1259) (actual time=0.947..0.955 rows=1 loops=1)
                                Buffers: shared hit=258
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..4.48 rows=3 width=16) (actual time=0.020..0.024 rows=10 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=4
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.018..0.020 rows=10 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=4
                                ->  Sort  (cost=1690.59..1691.10 rows=206 width=1259) (actual time=0.946..0.952 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 169kB
                                      Buffers: shared hit=258
                                      ->  Result  (cost=21.93..1683.71 rows=206 width=1259) (actual time=0.083..0.755 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=258
                                            ->  Nested Loop Anti Join  (cost=21.93..1683.71 rows=206 width=1251) (actual time=0.080..0.687 rows=210 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 630
                                                  Buffers: shared hit=258
                                                  ->  Index Scan using records_ou_list on records r_4  (cost=0.42..1644.62 rows=206 width=1251) (actual time=0.049..0.380 rows=210 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 10
                                                        Buffers: shared hit=245
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.001 rows=3 loops=210)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.021..0.028 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.023 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.340 ms
Execution Time: 4.115 ms
```

#### n_distinct sampled (3080): page C3, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=1745.83..2846.82 rows=1 width=1000) (actual time=1.387..3.782 rows=51 loops=1)
  Buffers: shared hit=1397
  ->  Nested Loop  (cost=1745.53..2838.49 rows=1 width=965) (actual time=1.366..3.516 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1244
        ->  Subquery Scan on x  (cost=1745.25..2830.18 rows=1 width=958) (actual time=1.338..3.011 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1091
              ->  Limit  (cost=1745.25..2829.03 rows=51 width=958) (actual time=1.332..2.939 rows=51 loops=1)
                    Buffers: shared hit=1091
                    ->  Merge Append  (cost=1745.25..5336.60 rows=169 width=958) (actual time=1.330..2.922 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1091
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.115..1.663 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.114..1.655 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.108..1.620 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.024..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.023..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.020..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.141..0.148 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.141..0.146 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.108..0.136 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.106..0.130 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.085..0.089 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.070..0.077 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.061..0.062 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.044..0.045 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1699.18..1699.31 rows=51 width=262) (actual time=1.027..1.034 rows=1 loops=1)
                                Buffers: shared hit=258
                                CTE acl_m
                                  ->  Result  (cost=26.40..1688.19 rows=206 width=1259) (actual time=0.085..0.669 rows=210 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=258
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..4.48 rows=3 width=16) (actual time=0.021..0.025 rows=10 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=4
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.019..0.021 rows=10 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=4
                                        ->  Nested Loop Anti Join  (cost=21.93..1683.71 rows=206 width=1251) (actual time=0.082..0.592 rows=210 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 630
                                              Buffers: shared hit=258
                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..1644.62 rows=206 width=1251) (actual time=0.048..0.326 rows=210 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 10
                                                    Buffers: shared hit=245
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=210)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.025 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.016..0.021 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=10.99..11.51 rows=206 width=262) (actual time=1.027..1.027 rows=1 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 169kB
                                      Buffers: shared hit=258
                                      ->  CTE Scan on acl_m  (cost=0.00..4.12 rows=206 width=262) (actual time=0.088..0.944 rows=210 loops=1)
                                            Buffers: shared hit=258
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.008..0.008 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.216 ms
Execution Time: 4.079 ms
```

#### n_distinct sampled (3080): page L1, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=1464.54..2565.53 rows=1 width=1301) (actual time=1.139..3.645 rows=51 loops=1)
  Buffers: shared hit=1335
  ->  Nested Loop  (cost=1464.24..2557.20 rows=1 width=1266) (actual time=1.116..3.267 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1182
        ->  Subquery Scan on x  (cost=1463.96..2548.89 rows=1 width=1259) (actual time=1.098..3.034 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1029
              ->  Limit  (cost=1463.96..2547.74 rows=51 width=1259) (actual time=1.096..2.952 rows=51 loops=1)
                    Buffers: shared hit=1029
                    ->  Merge Append  (cost=1463.96..5055.31 rows=169 width=1259) (actual time=1.094..2.939 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1029
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.133..1.923 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.132..1.913 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.118..1.868 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.024..0.031 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.024..0.029 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.020..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.123..0.131 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.122..0.128 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.085..0.114 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.082..0.109 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.065..0.069 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1417.89..1418.02 rows=51 width=1259) (actual time=0.791..0.802 rows=1 loops=1)
                                Buffers: shared hit=196
                                ->  Sort  (cost=1417.89..1418.26 rows=147 width=1259) (actual time=0.790..0.801 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 181kB
                                      Buffers: shared hit=196
                                      ->  Result  (cost=22.36..1412.99 rows=147 width=1259) (actual time=0.072..0.661 rows=150 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=196
                                            ->  Nested Loop  (cost=22.36..1412.99 rows=147 width=1259) (actual time=0.069..0.609 rows=150 loops=1)
                                                  Buffers: shared hit=196
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.021..0.028 rows=10 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.013..0.016 rows=10 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=4
                                                  ->  Limit  (cost=21.93..200.98 rows=21 width=1259) (actual time=0.014..0.053 rows=15 loops=10)
                                                        Buffers: shared hit=192
                                                        ->  Result  (cost=21.93..200.98 rows=21 width=1259) (actual time=0.013..0.050 rows=15 loops=10)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=192
                                                              ->  Nested Loop Anti Join  (cost=21.93..200.98 rows=21 width=1251) (actual time=0.012..0.044 rows=15 loops=10)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 45
                                                                    Buffers: shared hit=192
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..164.66 rows=21 width=1251) (actual time=0.009..0.025 rows=15 loops=10)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=179
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=150)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.018..0.025 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.016..0.022 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.015 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.480 ms
Execution Time: 3.956 ms
```

#### n_distinct sampled (3080): page C2, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=1747.63..2848.62 rows=1 width=1301) (actual time=4.189..6.699 rows=51 loops=1)
  Buffers: shared hit=2041
  ->  Nested Loop  (cost=1747.33..2840.29 rows=1 width=1266) (actual time=4.168..6.392 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1888
        ->  Subquery Scan on x  (cost=1747.05..2831.98 rows=1 width=1259) (actual time=4.142..6.114 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1735
              ->  Limit  (cost=1747.05..2830.83 rows=51 width=1259) (actual time=4.129..5.977 rows=51 loops=1)
                    Buffers: shared hit=1735
                    ->  Merge Append  (cost=1747.05..5338.40 rows=169 width=1259) (actual time=4.126..5.961 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1735
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.154..1.936 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.153..1.922 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.145..1.878 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.027..0.033 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.026..0.031 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.022..0.025 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.019..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.207..0.217 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.206..0.215 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.116..0.182 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.113..0.157 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.087..0.093 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.037..0.048 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.031 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.017..0.018 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1700.98..1701.11 rows=51 width=1259) (actual time=3.709..3.717 rows=1 loops=1)
                                Buffers: shared hit=902
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.39 rows=99 width=16) (actual time=0.023..0.054 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.39 rows=99 width=16) (actual time=0.021..0.038 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=1690.59..1691.10 rows=206 width=1259) (actual time=3.708..3.713 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=902
                                      ->  Result  (cost=21.93..1683.71 rows=206 width=1259) (actual time=0.131..3.050 rows=563 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=902
                                            ->  Nested Loop Anti Join  (cost=21.93..1683.71 rows=206 width=1251) (actual time=0.126..2.851 rows=563 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 1689
                                                  Buffers: shared hit=902
                                                  ->  Index Scan using records_ou_list on records r_4  (cost=0.42..1644.62 rows=206 width=1251) (actual time=0.090..2.000 rows=563 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 37
                                                        Buffers: shared hit=889
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=563)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.023..0.031 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.026 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.014..0.016 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.706 ms
Execution Time: 7.002 ms
```

#### n_distinct sampled (3080): page C3, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=1751.75..2852.74 rows=1 width=1000) (actual time=3.552..5.546 rows=51 loops=1)
  Buffers: shared hit=2041
  ->  Nested Loop  (cost=1751.45..2844.41 rows=1 width=965) (actual time=3.532..5.311 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1888
        ->  Subquery Scan on x  (cost=1751.17..2836.10 rows=1 width=958) (actual time=3.513..5.127 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1735
              ->  Limit  (cost=1751.17..2834.95 rows=51 width=958) (actual time=3.504..5.056 rows=51 loops=1)
                    Buffers: shared hit=1735
                    ->  Merge Append  (cost=1751.17..5342.52 rows=169 width=958) (actual time=3.502..5.043 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1735
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.114..1.614 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.113..1.605 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.107..1.572 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.021..0.028 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.021..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.019..0.021 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.014..0.014 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.096..0.102 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.096..0.101 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.063..0.091 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.061..0.087 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.045..0.049 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.040 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1705.10..1705.23 rows=51 width=262) (actual time=3.247..3.255 rows=1 loops=1)
                                Buffers: shared hit=902
                                CTE acl_m
                                  ->  Result  (cost=32.32..1694.11 rows=206 width=1259) (actual time=0.110..2.332 rows=563 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=902
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.39 rows=99 width=16) (actual time=0.016..0.046 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.39 rows=99 width=16) (actual time=0.014..0.031 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..1683.71 rows=206 width=1251) (actual time=0.108..2.159 rows=563 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 1689
                                              Buffers: shared hit=902
                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..1644.62 rows=206 width=1251) (actual time=0.080..1.489 rows=563 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 37
                                                    Buffers: shared hit=889
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=563)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.024 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.016..0.022 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=10.99..11.51 rows=206 width=262) (actual time=3.246..3.247 rows=1 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=902
                                      ->  CTE Scan on acl_m  (cost=0.00..4.12 rows=206 width=262) (actual time=0.113..3.044 rows=563 loops=1)
                                            Buffers: shared hit=902
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.359 ms
Execution Time: 5.890 ms
```

#### n_distinct sampled (3080): page L1, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=1464.54..2565.53 rows=1 width=1301) (actual time=4.018..6.109 rows=51 loops=1)
  Buffers: shared hit=2037
  ->  Nested Loop  (cost=1464.24..2557.20 rows=1 width=1266) (actual time=3.996..5.881 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1884
        ->  Subquery Scan on x  (cost=1463.96..2548.89 rows=1 width=1259) (actual time=3.973..5.659 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1731
              ->  Limit  (cost=1463.96..2547.74 rows=51 width=1259) (actual time=3.969..5.582 rows=51 loops=1)
                    Buffers: shared hit=1731
                    ->  Merge Append  (cost=1463.96..5055.31 rows=169 width=1259) (actual time=3.967..5.566 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1731
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.134..1.684 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.134..1.671 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.124..1.627 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.024..0.031 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.024..0.029 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.020..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.023..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.022..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.132..0.139 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.132..0.137 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.083..0.114 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.079..0.109 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.058..0.063 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.046..0.054 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.037..0.038 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1417.89..1418.02 rows=51 width=1259) (actual time=3.648..3.658 rows=1 loops=1)
                                Buffers: shared hit=898
                                ->  Sort  (cost=1417.89..1418.26 rows=147 width=1259) (actual time=3.647..3.656 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=898
                                      ->  Result  (cost=22.36..1412.99 rows=147 width=1259) (actual time=0.086..3.120 rows=559 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=898
                                            ->  Nested Loop  (cost=22.36..1412.99 rows=147 width=1259) (actual time=0.083..2.926 rows=559 loops=1)
                                                  Buffers: shared hit=898
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.022..0.074 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.015..0.043 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..200.98 rows=21 width=1259) (actual time=0.009..0.026 rows=6 loops=101)
                                                        Buffers: shared hit=893
                                                        ->  Result  (cost=21.93..200.98 rows=21 width=1259) (actual time=0.009..0.025 rows=6 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=893
                                                              ->  Nested Loop Anti Join  (cost=21.93..200.98 rows=21 width=1251) (actual time=0.008..0.022 rows=6 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 17
                                                                    Buffers: shared hit=893
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..164.66 rows=21 width=1251) (actual time=0.007..0.015 rows=6 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=880
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=559)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.026..0.032 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.020..0.025 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.015..0.017 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.564 ms
Execution Time: 6.435 ms
```

#### n_distinct sampled (3080): page C2, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=1747.89..2848.88 rows=1 width=1301) (actual time=4.677..6.526 rows=51 loops=1)
  Buffers: shared hit=2332
  ->  Nested Loop  (cost=1747.59..2840.55 rows=1 width=1266) (actual time=4.658..6.301 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2179
        ->  Subquery Scan on x  (cost=1747.31..2832.24 rows=1 width=1259) (actual time=4.638..6.120 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2026
              ->  Limit  (cost=1747.31..2831.09 rows=51 width=1259) (actual time=4.628..6.048 rows=51 loops=1)
                    Buffers: shared hit=2026
                    ->  Merge Append  (cost=1747.31..5338.66 rows=169 width=1259) (actual time=4.626..6.033 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2026
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.117..1.485 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.117..1.476 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.108..1.445 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.020..0.025 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.020..0.024 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.018..0.020 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.099..0.104 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.098..0.103 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.065..0.093 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.064..0.089 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.052 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1701.24..1701.37 rows=51 width=1259) (actual time=4.367..4.375 rows=3 loops=1)
                                Buffers: shared hit=1207
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.015..0.046 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.013..0.031 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=1690.59..1691.10 rows=206 width=1259) (actual time=4.366..4.372 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=1207
                                      ->  Result  (cost=21.93..1683.71 rows=206 width=1259) (actual time=0.105..3.658 rows=835 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=1207
                                            ->  Nested Loop Anti Join  (cost=21.93..1683.71 rows=206 width=1251) (actual time=0.104..3.364 rows=835 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 2505
                                                  Buffers: shared hit=1207
                                                  ->  Index Scan using records_ou_list on records r_4  (cost=0.42..1644.62 rows=206 width=1251) (actual time=0.078..2.380 rows=835 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 65
                                                        Buffers: shared hit=1194
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.015..0.022 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.019 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.013 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.450 ms
Execution Time: 6.819 ms
```

#### n_distinct sampled (3080): page C3, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=1752.01..2853.00 rows=1 width=1000) (actual time=5.897..7.919 rows=51 loops=1)
  Buffers: shared hit=2332
  ->  Nested Loop  (cost=1751.71..2844.67 rows=1 width=965) (actual time=5.871..7.637 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2179
        ->  Subquery Scan on x  (cost=1751.43..2836.36 rows=1 width=958) (actual time=5.850..7.412 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2026
              ->  Limit  (cost=1751.43..2835.21 rows=51 width=958) (actual time=5.839..7.314 rows=51 loops=1)
                    Buffers: shared hit=2026
                    ->  Merge Append  (cost=1751.43..5342.78 rows=169 width=958) (actual time=5.838..7.299 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2026
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.136..1.551 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.135..1.541 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.125..1.500 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.027..0.032 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.026..0.030 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.023..0.025 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.018..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.105..0.110 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.105..0.109 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.073..0.099 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.071..0.095 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.050..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1705.36..1705.49 rows=51 width=262) (actual time=5.545..5.553 rows=3 loops=1)
                                Buffers: shared hit=1207
                                CTE acl_m
                                  ->  Result  (cost=32.58..1694.37 rows=206 width=1259) (actual time=0.134..4.091 rows=835 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=1207
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.019..0.053 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.016..0.036 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..1683.71 rows=206 width=1251) (actual time=0.131..3.796 rows=835 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 2505
                                              Buffers: shared hit=1207
                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..1644.62 rows=206 width=1251) (actual time=0.094..2.755 rows=835 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 65
                                                    Buffers: shared hit=1194
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.020..0.031 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.027 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=10.99..11.51 rows=206 width=262) (actual time=5.544..5.546 rows=3 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=1207
                                      ->  CTE Scan on acl_m  (cost=0.00..4.12 rows=206 width=262) (actual time=0.136..5.238 rows=835 loops=1)
                                            Buffers: shared hit=1207
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.082 ms
Execution Time: 8.351 ms
```

#### n_distinct sampled (3080): page L1, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=1464.54..2565.53 rows=1 width=1301) (actual time=5.289..7.431 rows=51 loops=1)
  Buffers: shared hit=2109
  ->  Nested Loop  (cost=1464.24..2557.20 rows=1 width=1266) (actual time=5.265..7.127 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1956
        ->  Subquery Scan on x  (cost=1463.96..2548.89 rows=1 width=1259) (actual time=5.244..6.917 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1803
              ->  Limit  (cost=1463.96..2547.74 rows=51 width=1259) (actual time=5.237..6.830 rows=51 loops=1)
                    Buffers: shared hit=1803
                    ->  Merge Append  (cost=1463.96..5055.31 rows=169 width=1259) (actual time=5.235..6.815 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1803
                          ->  Limit  (cost=0.42..3322.94 rows=51 width=1259) (actual time=0.179..1.707 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..529517.37 rows=8128 width=1259) (actual time=0.178..1.697 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529517.37 rows=8128 width=1251) (actual time=0.168..1.660 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..205.11 rows=51 width=1259) (actual time=0.039..0.045 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..538.01 rows=134 width=1259) (actual time=0.038..0.043 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..538.01 rows=134 width=1251) (actual time=0.033..0.036 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.035..0.036 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.90 rows=15 width=1259) (actual time=0.034..0.035 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.90 rows=15 width=1251) (actual time=0.029..0.029 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.158..0.165 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.157..0.163 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.120..0.146 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.115..0.140 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.082..0.087 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.065..0.074 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.053..0.054 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.021..0.021 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.030..0.030 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1417.89..1418.02 rows=51 width=1259) (actual time=4.818..4.831 rows=3 loops=1)
                                Buffers: shared hit=984
                                ->  Sort  (cost=1417.89..1418.26 rows=147 width=1259) (actual time=4.817..4.829 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 191kB
                                      Buffers: shared hit=984
                                      ->  Result  (cost=22.36..1412.99 rows=147 width=1259) (actual time=0.070..4.110 rows=637 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=984
                                            ->  Nested Loop  (cost=22.36..1412.99 rows=147 width=1259) (actual time=0.068..3.881 rows=637 loops=1)
                                                  Buffers: shared hit=984
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.021..0.077 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.014..0.048 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..200.98 rows=21 width=1259) (actual time=0.011..0.035 rows=6 loops=101)
                                                        Buffers: shared hit=979
                                                        ->  Result  (cost=21.93..200.98 rows=21 width=1259) (actual time=0.010..0.034 rows=6 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=979
                                                              ->  Nested Loop Anti Join  (cost=21.93..200.98 rows=21 width=1251) (actual time=0.009..0.030 rows=6 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 19
                                                                    Buffers: shared hit=979
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..164.66 rows=21 width=1251) (actual time=0.007..0.021 rows=6 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=966
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=637)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.024 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.021 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.889 ms
Execution Time: 7.808 ms
```

#### n_distinct 1000 (1000): page C2, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=5061.33..6134.81 rows=1 width=1301) (actual time=1.113..2.950 rows=51 loops=1)
  Buffers: shared hit=1225
  ->  Nested Loop  (cost=5061.03..6126.48 rows=1 width=1266) (actual time=1.096..2.747 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1072
        ->  Subquery Scan on x  (cost=5060.75..6118.17 rows=1 width=1259) (actual time=1.083..2.584 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=919
              ->  Limit  (cost=5060.75..6117.02 rows=51 width=1259) (actual time=1.078..2.521 rows=51 loops=1)
                    Buffers: shared hit=919
                    ->  Merge Append  (cost=5060.75..8560.95 rows=169 width=1259) (actual time=1.077..2.510 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=919
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.108..1.506 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.107..1.498 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.102..1.469 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.022..0.027 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.022..0.026 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.019..0.021 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.014..0.014 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.143..0.147 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.143..0.147 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.111..0.137 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.109..0.133 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.050..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.025..0.025 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=5014.68..5014.81 rows=51 width=1259) (actual time=0.782..0.788 rows=1 loops=1)
                                Buffers: shared hit=86
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..4.48 rows=3 width=16) (actual time=0.017..0.020 rows=10 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=4
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.014..0.016 rows=10 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=4
                                ->  Sort  (cost=5010.20..5011.78 rows=631 width=1259) (actual time=0.781..0.786 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=86
                                      ->  Result  (cost=21.93..4989.15 rows=631 width=1259) (actual time=0.079..0.589 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=86
                                            ->  Nested Loop Anti Join  (cost=21.93..4989.15 rows=631 width=1251) (actual time=0.076..0.532 rows=210 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 630
                                                  Buffers: shared hit=86
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..4943.68 rows=631 width=1251) (actual time=0.042..0.287 rows=210 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 10
                                                        Buffers: shared hit=73
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=210)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.025 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.021 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.908 ms
Execution Time: 3.176 ms
```

#### n_distinct 1000 (1000): page C3, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=5073.95..6147.43 rows=1 width=1000) (actual time=1.219..2.999 rows=51 loops=1)
  Buffers: shared hit=1225
  ->  Nested Loop  (cost=5073.65..6139.10 rows=1 width=965) (actual time=1.204..2.814 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1072
        ->  Subquery Scan on x  (cost=5073.37..6130.79 rows=1 width=958) (actual time=1.190..2.656 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=919
              ->  Limit  (cost=5073.37..6129.64 rows=51 width=958) (actual time=1.186..2.595 rows=51 loops=1)
                    Buffers: shared hit=919
                    ->  Merge Append  (cost=5073.37..8573.57 rows=169 width=958) (actual time=1.185..2.585 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=919
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.118..1.484 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.117..1.476 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.112..1.449 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.025..0.029 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.025..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.021..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.023..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.103..0.108 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.103..0.107 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.067..0.097 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.065..0.093 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.045..0.048 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=5027.30..5027.43 rows=51 width=262) (actual time=0.911..0.917 rows=1 loops=1)
                                Buffers: shared hit=86
                                CTE acl_m
                                  ->  Result  (cost=26.40..4993.63 rows=631 width=1259) (actual time=0.072..0.546 rows=210 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=86
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..4.48 rows=3 width=16) (actual time=0.018..0.022 rows=10 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=4
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.017..0.019 rows=10 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=4
                                        ->  Nested Loop Anti Join  (cost=21.93..4989.15 rows=631 width=1251) (actual time=0.070..0.482 rows=210 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 630
                                              Buffers: shared hit=86
                                              ->  Index Scan using records_org_unit on records r_4  (cost=0.42..4943.68 rows=631 width=1251) (actual time=0.039..0.241 rows=210 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 10
                                                    Buffers: shared hit=73
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=210)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.021..0.026 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.018..0.023 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=33.67..35.25 rows=631 width=262) (actual time=0.910..0.911 rows=1 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=86
                                      ->  CTE Scan on acl_m  (cost=0.00..12.62 rows=631 width=262) (actual time=0.074..0.825 rows=210 loops=1)
                                            Buffers: shared hit=86
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.990 ms
Execution Time: 3.240 ms
```

#### n_distinct 1000 (1000): page L1, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=3149.83..4223.32 rows=1 width=1301) (actual time=1.462..4.759 rows=51 loops=1)
  Buffers: shared hit=1335
  ->  Nested Loop  (cost=3149.53..4214.98 rows=1 width=1266) (actual time=1.436..4.342 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1182
        ->  Subquery Scan on x  (cost=3149.25..4206.67 rows=1 width=1259) (actual time=1.417..3.962 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1029
              ->  Limit  (cost=3149.25..4205.52 rows=51 width=1259) (actual time=1.413..3.827 rows=51 loops=1)
                    Buffers: shared hit=1029
                    ->  Merge Append  (cost=3149.25..6649.45 rows=169 width=1259) (actual time=1.411..3.769 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1029
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.115..2.411 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.115..2.394 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.109..2.272 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.023..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.022..0.029 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.019..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.020..0.022 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.020..0.021 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.018..0.018 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.201..0.207 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.200..0.206 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.075..0.193 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.072..0.188 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.054..0.059 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.037..0.049 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.040..0.040 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=3103.18..3103.31 rows=51 width=1259) (actual time=1.046..1.057 rows=1 loops=1)
                                Buffers: shared hit=196
                                ->  Sort  (cost=3103.18..3104.07 rows=357 width=1259) (actual time=1.045..1.055 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 181kB
                                      Buffers: shared hit=196
                                      ->  Result  (cost=22.36..3091.27 rows=357 width=1259) (actual time=0.110..0.861 rows=150 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=196
                                            ->  Nested Loop  (cost=22.36..3091.27 rows=357 width=1259) (actual time=0.105..0.809 rows=150 loops=1)
                                                  Buffers: shared hit=196
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.031..0.044 rows=10 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.011..0.012 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.017..0.025 rows=10 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=4
                                                  ->  Limit  (cost=21.93..440.44 rows=51 width=1259) (actual time=0.018..0.070 rows=15 loops=10)
                                                        Buffers: shared hit=192
                                                        ->  Result  (cost=21.93..538.91 rows=63 width=1259) (actual time=0.018..0.065 rows=15 loops=10)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=192
                                                              ->  Nested Loop Anti Join  (cost=21.93..538.91 rows=63 width=1251) (actual time=0.015..0.059 rows=15 loops=10)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 45
                                                                    Buffers: shared hit=192
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..501.96 rows=63 width=1251) (actual time=0.011..0.037 rows=15 loops=10)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=179
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.001 rows=3 loops=150)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.028..0.034 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.025..0.030 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.004..0.004 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.666 ms
Execution Time: 5.034 ms
```

#### n_distinct 1000 (1000): page C2, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=5067.25..6140.73 rows=1 width=1301) (actual time=2.834..4.876 rows=51 loops=1)
  Buffers: shared hit=1885
  ->  Nested Loop  (cost=5066.95..6132.40 rows=1 width=1266) (actual time=2.791..4.599 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1732
        ->  Subquery Scan on x  (cost=5066.67..6124.09 rows=1 width=1259) (actual time=2.761..4.366 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1579
              ->  Limit  (cost=5066.67..6122.94 rows=51 width=1259) (actual time=2.751..4.286 rows=51 loops=1)
                    Buffers: shared hit=1579
                    ->  Merge Append  (cost=5066.67..8566.87 rows=169 width=1259) (actual time=2.750..4.273 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1579
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.098..1.582 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.098..1.570 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.092..1.531 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.017..0.022 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.017..0.021 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.015..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.013..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.013..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.012..0.012 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.090..0.095 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.094 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.054..0.085 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.052..0.081 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.042 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=5020.60..5020.73 rows=51 width=1259) (actual time=2.522..2.528 rows=1 loops=1)
                                Buffers: shared hit=746
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.39 rows=99 width=16) (actual time=0.014..0.038 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.39 rows=99 width=16) (actual time=0.013..0.026 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=5010.20..5011.78 rows=631 width=1259) (actual time=2.521..2.526 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 197kB
                                      Buffers: shared hit=746
                                      ->  Result  (cost=21.93..4989.15 rows=631 width=1259) (actual time=0.163..2.074 rows=563 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=746
                                            ->  Nested Loop Anti Join  (cost=21.93..4989.15 rows=631 width=1251) (actual time=0.160..1.895 rows=563 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 1689
                                                  Buffers: shared hit=746
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..4943.68 rows=631 width=1251) (actual time=0.127..1.298 rows=563 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 37
                                                        Buffers: shared hit=733
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=563)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.022..0.029 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.024 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.142 ms
Execution Time: 5.156 ms
```

#### n_distinct 1000 (1000): page C3, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=5079.87..6153.35 rows=1 width=1000) (actual time=3.046..4.852 rows=51 loops=1)
  Buffers: shared hit=1885
  ->  Nested Loop  (cost=5079.57..6145.02 rows=1 width=965) (actual time=3.026..4.629 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1732
        ->  Subquery Scan on x  (cost=5079.29..6136.71 rows=1 width=958) (actual time=3.010..4.420 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1579
              ->  Limit  (cost=5079.29..6135.56 rows=51 width=958) (actual time=3.003..4.343 rows=51 loops=1)
                    Buffers: shared hit=1579
                    ->  Merge Append  (cost=5079.29..8579.49 rows=169 width=958) (actual time=3.002..4.330 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1579
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.099..1.384 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.099..1.376 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.094..1.343 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.025..0.029 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.024..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.020..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.093 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.088..0.092 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.059..0.083 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.057..0.080 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.042 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=5033.22..5033.35 rows=51 width=262) (actual time=2.766..2.775 rows=1 loops=1)
                                Buffers: shared hit=746
                                CTE acl_m
                                  ->  Result  (cost=32.32..4999.55 rows=631 width=1259) (actual time=0.092..1.913 rows=563 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=746
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.39 rows=99 width=16) (actual time=0.014..0.038 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.39 rows=99 width=16) (actual time=0.012..0.025 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..4989.15 rows=631 width=1251) (actual time=0.089..1.759 rows=563 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 1689
                                              Buffers: shared hit=746
                                              ->  Index Scan using records_org_unit on records r_4  (cost=0.42..4943.68 rows=631 width=1251) (actual time=0.066..1.147 rows=563 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 37
                                                    Buffers: shared hit=733
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=563)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.015..0.020 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.018 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=33.67..35.25 rows=631 width=262) (actual time=2.765..2.766 rows=1 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 197kB
                                      Buffers: shared hit=746
                                      ->  CTE Scan on acl_m  (cost=0.00..12.62 rows=631 width=262) (actual time=0.094..2.578 rows=563 loops=1)
                                            Buffers: shared hit=746
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.892 ms
Execution Time: 5.118 ms
```

#### n_distinct 1000 (1000): page L1, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=3149.83..4223.32 rows=1 width=1301) (actual time=4.036..5.716 rows=51 loops=1)
  Buffers: shared hit=2037
  ->  Nested Loop  (cost=3149.53..4214.98 rows=1 width=1266) (actual time=4.020..5.514 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1884
        ->  Subquery Scan on x  (cost=3149.25..4206.67 rows=1 width=1259) (actual time=4.006..5.340 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1731
              ->  Limit  (cost=3149.25..4205.52 rows=51 width=1259) (actual time=4.002..5.272 rows=51 loops=1)
                    Buffers: shared hit=1731
                    ->  Merge Append  (cost=3149.25..6649.45 rows=169 width=1259) (actual time=4.001..5.259 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1731
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.192..1.390 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.191..1.382 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.185..1.352 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.030..0.037 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.030..0.036 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.025..0.028 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.024..0.026 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.111..0.115 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.110..0.114 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.070..0.100 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.065..0.094 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.047..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.042 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=3103.18..3103.31 rows=51 width=1259) (actual time=3.638..3.646 rows=1 loops=1)
                                Buffers: shared hit=898
                                ->  Sort  (cost=3103.18..3104.07 rows=357 width=1259) (actual time=3.638..3.645 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=898
                                      ->  Result  (cost=22.36..3091.27 rows=357 width=1259) (actual time=0.099..3.121 rows=559 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=898
                                            ->  Nested Loop  (cost=22.36..3091.27 rows=357 width=1259) (actual time=0.096..2.928 rows=559 loops=1)
                                                  Buffers: shared hit=898
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.030..0.079 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.009..0.010 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.018..0.043 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..440.44 rows=51 width=1259) (actual time=0.010..0.026 rows=6 loops=101)
                                                        Buffers: shared hit=893
                                                        ->  Result  (cost=21.93..538.91 rows=63 width=1259) (actual time=0.009..0.025 rows=6 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=893
                                                              ->  Nested Loop Anti Join  (cost=21.93..538.91 rows=63 width=1251) (actual time=0.008..0.022 rows=6 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 17
                                                                    Buffers: shared hit=893
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..501.96 rows=63 width=1251) (actual time=0.007..0.015 rows=6 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=880
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=559)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.025..0.030 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.023..0.027 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.045 ms
Execution Time: 6.008 ms
```

#### n_distinct 1000 (1000): page C2, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=5067.51..6140.99 rows=1 width=1301) (actual time=4.235..6.100 rows=51 loops=1)
  Buffers: shared hit=2077
  ->  Nested Loop  (cost=5067.21..6132.66 rows=1 width=1266) (actual time=4.213..5.888 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1924
        ->  Subquery Scan on x  (cost=5066.93..6124.35 rows=1 width=1259) (actual time=4.195..5.701 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1771
              ->  Limit  (cost=5066.93..6123.20 rows=51 width=1259) (actual time=4.185..5.620 rows=51 loops=1)
                    Buffers: shared hit=1771
                    ->  Merge Append  (cost=5066.93..8567.13 rows=169 width=1259) (actual time=4.183..5.608 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1771
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.112..1.494 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.112..1.485 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.106..1.456 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.026..0.032 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.026..0.030 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.022..0.026 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.020..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.020..0.021 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.017..0.018 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.103..0.110 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.103..0.109 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.063..0.093 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.089 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.049 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.039 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=5020.86..5020.99 rows=51 width=1259) (actual time=3.914..3.923 rows=3 loops=1)
                                Buffers: shared hit=952
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.013..0.043 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.012..0.028 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=5010.20..5011.78 rows=631 width=1259) (actual time=3.913..3.920 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=952
                                      ->  Result  (cost=21.93..4989.15 rows=631 width=1259) (actual time=0.103..3.298 rows=835 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=952
                                            ->  Nested Loop Anti Join  (cost=21.93..4989.15 rows=631 width=1251) (actual time=0.101..3.051 rows=835 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 2505
                                                  Buffers: shared hit=952
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..4943.68 rows=631 width=1251) (actual time=0.075..2.126 rows=835 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 65
                                                        Buffers: shared hit=939
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.023 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.019 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.233 ms
Execution Time: 6.352 ms
```

#### n_distinct 1000 (1000): page C3, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=5080.13..6153.61 rows=1 width=1000) (actual time=4.795..6.699 rows=51 loops=1)
  Buffers: shared hit=2077
  ->  Nested Loop  (cost=5079.83..6145.28 rows=1 width=965) (actual time=4.778..6.464 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1924
        ->  Subquery Scan on x  (cost=5079.55..6136.97 rows=1 width=958) (actual time=4.763..6.259 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1771
              ->  Limit  (cost=5079.55..6135.82 rows=51 width=958) (actual time=4.756..6.175 rows=51 loops=1)
                    Buffers: shared hit=1771
                    ->  Merge Append  (cost=5079.55..8579.75 rows=169 width=958) (actual time=4.755..6.162 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1771
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.113..1.480 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.112..1.471 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.107..1.441 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.053..0.057 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.052..0.056 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.049..0.052 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.023..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.019..0.019 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.092..0.097 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.092..0.096 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.060..0.086 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.057..0.082 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.042 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.028..0.034 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=5033.48..5033.61 rows=51 width=262) (actual time=4.468..4.475 rows=3 loops=1)
                                Buffers: shared hit=952
                                CTE acl_m
                                  ->  Result  (cost=32.58..4999.81 rows=631 width=1259) (actual time=0.115..3.223 rows=835 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=952
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.018..0.050 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.016..0.036 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..4989.15 rows=631 width=1251) (actual time=0.112..2.986 rows=835 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 2505
                                              Buffers: shared hit=952
                                              ->  Index Scan using records_org_unit on records r_4  (cost=0.42..4943.68 rows=631 width=1251) (actual time=0.081..2.092 rows=835 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 65
                                                    Buffers: shared hit=939
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.022..0.026 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.023 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=33.67..35.25 rows=631 width=262) (actual time=4.467..4.469 rows=3 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=952
                                      ->  CTE Scan on acl_m  (cost=0.00..12.62 rows=631 width=262) (actual time=0.117..4.197 rows=835 loops=1)
                                            Buffers: shared hit=952
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.178 ms
Execution Time: 7.014 ms
```

#### n_distinct 1000 (1000): page L1, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=3149.83..4223.32 rows=1 width=1301) (actual time=4.015..5.554 rows=51 loops=1)
  Buffers: shared hit=2109
  ->  Nested Loop  (cost=3149.53..4214.98 rows=1 width=1266) (actual time=4.000..5.378 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1956
        ->  Subquery Scan on x  (cost=3149.25..4206.67 rows=1 width=1259) (actual time=3.986..5.229 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1803
              ->  Limit  (cost=3149.25..4205.52 rows=51 width=1259) (actual time=3.984..5.176 rows=51 loops=1)
                    Buffers: shared hit=1803
                    ->  Merge Append  (cost=3149.25..6649.45 rows=169 width=1259) (actual time=3.982..5.166 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1803
                          ->  Limit  (cost=0.42..3233.19 rows=51 width=1259) (actual time=0.103..1.249 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..529032.33 rows=8346 width=1259) (actual time=0.102..1.239 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..529032.33 rows=8346 width=1251) (actual time=0.097..1.216 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..203.92 rows=51 width=1259) (actual time=0.022..0.026 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..1105.13 rows=277 width=1259) (actual time=0.022..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..1105.13 rows=277 width=1251) (actual time=0.017..0.020 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.015..0.016 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.68 rows=15 width=1259) (actual time=0.015..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.68 rows=15 width=1251) (actual time=0.013..0.013 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.086 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.085 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.053..0.076 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.051..0.072 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.042 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.028..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=3103.18..3103.31 rows=51 width=1259) (actual time=3.756..3.764 rows=3 loops=1)
                                Buffers: shared hit=984
                                ->  Sort  (cost=3103.18..3104.07 rows=357 width=1259) (actual time=3.756..3.763 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 191kB
                                      Buffers: shared hit=984
                                      ->  Result  (cost=22.36..3091.27 rows=357 width=1259) (actual time=0.064..3.281 rows=637 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=984
                                            ->  Nested Loop  (cost=22.36..3091.27 rows=357 width=1259) (actual time=0.063..3.092 rows=637 loops=1)
                                                  Buffers: shared hit=984
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.021..0.064 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.005 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.015..0.040 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..440.44 rows=51 width=1259) (actual time=0.008..0.028 rows=6 loops=101)
                                                        Buffers: shared hit=979
                                                        ->  Result  (cost=21.93..538.91 rows=63 width=1259) (actual time=0.008..0.027 rows=6 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=979
                                                              ->  Nested Loop Anti Join  (cost=21.93..538.91 rows=63 width=1251) (actual time=0.007..0.024 rows=6 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 19
                                                                    Buffers: shared hit=979
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..501.96 rows=63 width=1251) (actual time=0.006..0.017 rows=6 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=966
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=637)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.016..0.020 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.018 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.137 ms
Execution Time: 5.782 ms
```

#### n_distinct 9000 (9000): page C2, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=673.41..1750.91 rows=1 width=1301) (actual time=1.164..3.178 rows=51 loops=1)
  Buffers: shared hit=1397
  ->  Nested Loop  (cost=673.11..1742.58 rows=1 width=1266) (actual time=1.143..2.938 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1244
        ->  Subquery Scan on x  (cost=672.83..1734.26 rows=1 width=1259) (actual time=1.128..2.731 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1091
              ->  Limit  (cost=672.83..1733.12 rows=51 width=1259) (actual time=1.121..2.655 rows=51 loops=1)
                    Buffers: shared hit=1091
                    ->  Merge Append  (cost=672.83..4186.32 rows=169 width=1259) (actual time=1.119..2.643 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1091
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.113..1.594 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.113..1.580 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.107..1.548 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.022..0.028 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.022..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.019..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.014..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.103..0.108 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.102..0.107 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.071..0.097 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.069..0.094 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.052..0.056 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.047 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=626.77..626.89 rows=51 width=1259) (actual time=0.859..0.865 rows=1 loops=1)
                                Buffers: shared hit=258
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..4.48 rows=3 width=16) (actual time=0.017..0.021 rows=10 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=4
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.015..0.017 rows=10 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=4
                                ->  Sort  (cost=622.29..622.46 rows=69 width=1259) (actual time=0.858..0.863 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 169kB
                                      Buffers: shared hit=258
                                      ->  Result  (cost=21.93..620.18 rows=69 width=1259) (actual time=0.078..0.690 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=258
                                            ->  Nested Loop Anti Join  (cost=21.93..620.18 rows=69 width=1251) (actual time=0.075..0.623 rows=210 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 630
                                                  Buffers: shared hit=258
                                                  ->  Index Scan using records_ou_list on records r_4  (cost=0.42..583.14 rows=69 width=1251) (actual time=0.043..0.337 rows=210 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 10
                                                        Buffers: shared hit=245
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=210)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.028 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.015..0.025 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.059 ms
Execution Time: 3.461 ms
```

#### n_distinct 9000 (9000): page C3, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=674.79..1752.29 rows=1 width=1000) (actual time=1.447..3.261 rows=51 loops=1)
  Buffers: shared hit=1397
  ->  Nested Loop  (cost=674.49..1743.96 rows=1 width=965) (actual time=1.426..3.055 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1244
        ->  Subquery Scan on x  (cost=674.21..1735.64 rows=1 width=958) (actual time=1.410..2.877 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1091
              ->  Limit  (cost=674.21..1734.50 rows=51 width=958) (actual time=1.403..2.812 rows=51 loops=1)
                    Buffers: shared hit=1091
                    ->  Merge Append  (cost=674.21..4187.70 rows=169 width=958) (actual time=1.401..2.800 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1091
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.140..1.502 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.139..1.494 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.133..1.465 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.023..0.028 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.023..0.026 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.019..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.022..0.024 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.022..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.123..0.128 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.122..0.127 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.091..0.117 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.088..0.113 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.071..0.075 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.059..0.067 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.048..0.049 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.031..0.031 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=628.15..628.27 rows=51 width=262) (actual time=1.088..1.096 rows=1 loops=1)
                                Buffers: shared hit=258
                                CTE acl_m
                                  ->  Result  (cost=26.40..624.66 rows=69 width=1259) (actual time=0.086..0.750 rows=210 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=258
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..4.48 rows=3 width=16) (actual time=0.022..0.026 rows=10 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=4
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.019..0.021 rows=10 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=4
                                        ->  Nested Loop Anti Join  (cost=21.93..620.18 rows=69 width=1251) (actual time=0.082..0.687 rows=210 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 630
                                              Buffers: shared hit=258
                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..583.14 rows=69 width=1251) (actual time=0.052..0.423 rows=210 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 10
                                                    Buffers: shared hit=245
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=210)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.020..0.027 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.022 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=3.49..3.66 rows=69 width=262) (actual time=1.087..1.088 rows=1 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 169kB
                                      Buffers: shared hit=258
                                      ->  CTE Scan on acl_m  (cost=0.00..1.38 rows=69 width=262) (actual time=0.089..0.998 rows=210 loops=1)
                                            Buffers: shared hit=258
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.452 ms
Execution Time: 3.586 ms
```

#### n_distinct 9000 (9000): page L1, area (10 units, 220 tickets)

```
Nested Loop Left Join  (cost=669.62..1759.80 rows=1 width=1301) (actual time=1.533..3.429 rows=51 loops=1)
  Buffers: shared hit=1623
  ->  Nested Loop  (cost=669.32..1751.47 rows=1 width=1266) (actual time=1.516..3.219 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1470
        ->  Subquery Scan on x  (cost=669.04..1743.15 rows=1 width=1259) (actual time=1.501..3.044 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1317
              ->  Limit  (cost=669.04..1742.01 rows=51 width=1259) (actual time=1.498..2.981 rows=51 loops=1)
                    Buffers: shared hit=1317
                    ->  Merge Append  (cost=669.04..4182.48 rows=167 width=1259) (actual time=1.497..2.970 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1317
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.114..1.554 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.113..1.545 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.106..1.513 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.028..0.032 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.027..0.031 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.024..0.026 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.019..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.108..0.112 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.107..0.111 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.075..0.101 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.072..0.097 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.054..0.058 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.041..0.049 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.030 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=622.97..623.09 rows=49 width=1259) (actual time=1.223..1.228 rows=1 loops=1)
                                Buffers: shared hit=484
                                ->  Sort  (cost=622.97..623.09 rows=49 width=1259) (actual time=1.222..1.227 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 181kB
                                      Buffers: shared hit=484
                                      ->  Result  (cost=1.15..621.59 rows=49 width=1259) (actual time=0.100..1.082 rows=150 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=484
                                            ->  Nested Loop  (cost=1.15..621.59 rows=49 width=1259) (actual time=0.097..1.031 rows=150 loops=1)
                                                  Buffers: shared hit=484
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.060..0.066 rows=10 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.053..0.056 rows=10 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=4
                                                  ->  Limit  (cost=0.72..88.06 rows=7 width=1259) (actual time=0.012..0.090 rows=15 loops=10)
                                                        Buffers: shared hit=480
                                                        ->  Result  (cost=0.72..88.06 rows=7 width=1259) (actual time=0.011..0.087 rows=15 loops=10)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=480
                                                              ->  Nested Loop Anti Join  (cost=0.72..88.06 rows=7 width=1251) (actual time=0.010..0.081 rows=15 loops=10)
                                                                    Buffers: shared hit=480
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..57.76 rows=7 width=1251) (actual time=0.008..0.024 rows=15 loops=10)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=179
                                                                    ->  Result  (cost=0.29..4.32 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=150)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=301
                                                                          ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..4.32 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=150)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d… (414 chars)
                                                                                Rows Removed by Filter: 0
                                                                                Heap Fetches: 0
                                                                                Buffers: shared hit=301
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.221 ms
Execution Time: 3.674 ms
```

#### n_distinct 9000 (9000): page C2, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=679.33..1756.83 rows=1 width=1301) (actual time=2.896..4.778 rows=51 loops=1)
  Buffers: shared hit=2041
  ->  Nested Loop  (cost=679.03..1748.50 rows=1 width=1266) (actual time=2.878..4.589 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1888
        ->  Subquery Scan on x  (cost=678.75..1740.18 rows=1 width=1259) (actual time=2.861..4.429 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1735
              ->  Limit  (cost=678.75..1739.04 rows=51 width=1259) (actual time=2.853..4.369 rows=51 loops=1)
                    Buffers: shared hit=1735
                    ->  Merge Append  (cost=678.75..4192.24 rows=169 width=1259) (actual time=2.852..4.358 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1735
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.117..1.587 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.116..1.579 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.111..1.552 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.023..0.028 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.023..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.019..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.016..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.014..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.098..0.104 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.097..0.103 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.066..0.092 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.064..0.089 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.037..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.031 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=632.69..632.81 rows=51 width=1259) (actual time=2.593..2.599 rows=1 loops=1)
                                Buffers: shared hit=902
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.39 rows=99 width=16) (actual time=0.016..0.044 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.39 rows=99 width=16) (actual time=0.015..0.030 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=622.29..622.46 rows=69 width=1259) (actual time=2.592..2.597 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=902
                                      ->  Result  (cost=21.93..620.18 rows=69 width=1259) (actual time=0.100..2.165 rows=563 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=902
                                            ->  Nested Loop Anti Join  (cost=21.93..620.18 rows=69 width=1251) (actual time=0.098..2.012 rows=563 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 1689
                                                  Buffers: shared hit=902
                                                  ->  Index Scan using records_ou_list on records r_4  (cost=0.42..583.14 rows=69 width=1251) (actual time=0.075..1.411 rows=563 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 37
                                                        Buffers: shared hit=889
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=563)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.020 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.012..0.017 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.005..0.006 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.931 ms
Execution Time: 5.023 ms
```

#### n_distinct 9000 (9000): page C3, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=680.71..1758.21 rows=1 width=1000) (actual time=4.000..6.234 rows=51 loops=1)
  Buffers: shared hit=2041
  ->  Nested Loop  (cost=680.41..1749.88 rows=1 width=965) (actual time=3.977..5.983 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1888
        ->  Subquery Scan on x  (cost=680.13..1741.56 rows=1 width=958) (actual time=3.956..5.748 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1735
              ->  Limit  (cost=680.13..1740.42 rows=51 width=958) (actual time=3.946..5.660 rows=51 loops=1)
                    Buffers: shared hit=1735
                    ->  Merge Append  (cost=680.13..4193.62 rows=169 width=958) (actual time=3.944..5.646 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1735
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.125..1.779 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.124..1.766 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.116..1.728 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.025..0.032 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.024..0.030 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.020..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.017..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.109..0.115 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.109..0.114 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.073..0.103 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.071..0.099 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=634.07..634.19 rows=51 width=262) (actual time=3.661..3.669 rows=1 loops=1)
                                Buffers: shared hit=902
                                CTE acl_m
                                  ->  Result  (cost=32.32..630.58 rows=69 width=1259) (actual time=0.120..2.619 rows=563 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=902
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.39 rows=99 width=16) (actual time=0.018..0.047 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.39 rows=99 width=16) (actual time=0.016..0.032 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..620.18 rows=69 width=1251) (actual time=0.117..2.443 rows=563 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 1689
                                              Buffers: shared hit=902
                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..583.14 rows=69 width=1251) (actual time=0.087..1.703 rows=563 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 37
                                                    Buffers: shared hit=889
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=563)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.025 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.022 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.005 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=3.49..3.66 rows=69 width=262) (actual time=3.660..3.660 rows=1 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=902
                                      ->  CTE Scan on acl_m  (cost=0.00..1.38 rows=69 width=262) (actual time=0.123..3.436 rows=563 loops=1)
                                            Buffers: shared hit=902
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.269 ms
Execution Time: 6.584 ms
```

#### n_distinct 9000 (9000): page L1, district thin (101 units, 600 tickets)

```
Nested Loop Left Join  (cost=669.62..1759.80 rows=1 width=1301) (actual time=5.418..7.493 rows=51 loops=1)
  Buffers: shared hit=3143
  ->  Nested Loop  (cost=669.32..1751.47 rows=1 width=1266) (actual time=5.396..7.244 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2990
        ->  Subquery Scan on x  (cost=669.04..1743.15 rows=1 width=1259) (actual time=5.376..7.050 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2837
              ->  Limit  (cost=669.04..1742.01 rows=51 width=1259) (actual time=5.371..6.982 rows=51 loops=1)
                    Buffers: shared hit=2837
                    ->  Merge Append  (cost=669.04..4182.48 rows=167 width=1259) (actual time=5.369..6.969 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2837
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.115..1.674 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.115..1.664 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.108..1.627 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.022..0.029 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.022..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.020..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.016..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.016..0.016 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.014..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.092..0.100 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.092..0.099 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.062..0.089 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.084 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.048 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.040 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=622.97..623.09 rows=49 width=1259) (actual time=5.114..5.120 rows=1 loops=1)
                                Buffers: shared hit=2004
                                ->  Sort  (cost=622.97..623.09 rows=49 width=1259) (actual time=5.113..5.119 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=2004
                                      ->  Result  (cost=1.15..621.59 rows=49 width=1259) (actual time=0.050..4.613 rows=559 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=2004
                                            ->  Nested Loop  (cost=1.15..621.59 rows=49 width=1259) (actual time=0.048..4.422 rows=559 loops=1)
                                                  Buffers: shared hit=2004
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.019..0.065 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.013..0.038 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=0.72..88.06 rows=7 width=1259) (actual time=0.010..0.041 rows=6 loops=101)
                                                        Buffers: shared hit=1999
                                                        ->  Result  (cost=0.72..88.06 rows=7 width=1259) (actual time=0.009..0.040 rows=6 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=1999
                                                              ->  Nested Loop Anti Join  (cost=0.72..88.06 rows=7 width=1251) (actual time=0.008..0.037 rows=6 loops=101)
                                                                    Buffers: shared hit=1999
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..57.76 rows=7 width=1251) (actual time=0.006..0.014 rows=6 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=880
                                                                    ->  Result  (cost=0.29..4.32 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=559)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=1119
                                                                          ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..4.32 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=559)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d… (414 chars)
                                                                                Rows Removed by Filter: 0
                                                                                Heap Fetches: 0
                                                                                Buffers: shared hit=1119
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.095 ms
Execution Time: 7.792 ms
```

#### n_distinct 9000 (9000): page C2, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=679.59..1757.09 rows=1 width=1301) (actual time=4.994..8.209 rows=51 loops=1)
  Buffers: shared hit=2332
  ->  Nested Loop  (cost=679.29..1748.76 rows=1 width=1266) (actual time=4.962..7.778 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2179
        ->  Subquery Scan on x  (cost=679.01..1740.44 rows=1 width=1259) (actual time=4.927..7.434 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2026
              ->  Limit  (cost=679.01..1739.30 rows=51 width=1259) (actual time=4.913..7.297 rows=51 loops=1)
                    Buffers: shared hit=2026
                    ->  Merge Append  (cost=679.01..4192.50 rows=169 width=1259) (actual time=4.912..7.280 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2026
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.119..2.409 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.118..2.393 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.112..2.350 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.028..0.038 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.027..0.035 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.023..0.029 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.025..0.027 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.021..0.021 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.094..0.109 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.094..0.108 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.062..0.094 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.088 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.043..0.049 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.040 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=632.95..633.07 rows=51 width=1259) (actual time=4.638..4.657 rows=3 loops=1)
                                Buffers: shared hit=1207
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.015..0.049 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.014..0.034 rows=101 loops=1)
                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=622.29..622.46 rows=69 width=1259) (actual time=4.636..4.651 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=1207
                                      ->  Result  (cost=21.93..620.18 rows=69 width=1259) (actual time=0.162..3.942 rows=835 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=1207
                                            ->  Nested Loop Anti Join  (cost=21.93..620.18 rows=69 width=1251) (actual time=0.160..3.695 rows=835 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 2505
                                                  Buffers: shared hit=1207
                                                  ->  Index Scan using records_ou_list on records r_4  (cost=0.42..583.14 rows=69 width=1251) (actual time=0.090..2.618 rows=835 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_g… (479 chars)
                                                        Rows Removed by Filter: 65
                                                        Buffers: shared hit=1194
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.056..0.068 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.020..0.030 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.016..0.019 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.004..0.004 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.154 ms
Execution Time: 8.529 ms
```

#### n_distinct 9000 (9000): page C3, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=680.97..1758.47 rows=1 width=1000) (actual time=7.076..9.945 rows=51 loops=1)
  Buffers: shared hit=2332
  ->  Nested Loop  (cost=680.67..1750.14 rows=1 width=965) (actual time=7.053..9.492 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2179
        ->  Subquery Scan on x  (cost=680.39..1741.82 rows=1 width=958) (actual time=7.027..9.197 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2026
              ->  Limit  (cost=680.39..1740.68 rows=51 width=958) (actual time=7.011..9.066 rows=51 loops=1)
                    Buffers: shared hit=2026
                    ->  Merge Append  (cost=680.39..4193.88 rows=169 width=958) (actual time=7.009..9.048 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2026
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.131..2.109 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.131..2.096 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.123..2.049 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.033..0.039 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.032..0.037 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.029..0.032 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.030..0.032 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.024..0.025 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.126..0.135 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.125..0.133 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.080..0.116 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.076..0.109 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.049..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.036..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=634.33..634.45 rows=51 width=262) (actual time=6.682..6.693 rows=3 loops=1)
                                Buffers: shared hit=1207
                                CTE acl_m
                                  ->  Result  (cost=32.58..630.84 rows=69 width=1259) (actual time=0.181..4.748 rows=835 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                        Buffers: shared hit=1207
                                        InitPlan 1 (returns $0)
                                          ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.025..0.075 rows=101 loops=1)
                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                Buffers: shared hit=5
                                                ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.023..0.054 rows=101 loops=1)
                                                      Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f8b34cd-9d5b-e69e-d928-d6de6a6d9738}'::uuid[])))
                                                      Heap Fetches: 0
                                                      Buffers: shared hit=5
                                        ->  Nested Loop Anti Join  (cost=21.93..620.18 rows=69 width=1251) (actual time=0.173..4.389 rows=835 loops=1)
                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                              Rows Removed by Join Filter: 2505
                                              Buffers: shared hit=1207
                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..583.14 rows=69 width=1251) (actual time=0.126..3.181 rows=835 loops=1)
                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = ANY ($0)))
                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group… (475 chars)
                                                    Rows Removed by Filter: 65
                                                    Buffers: shared hit=1194
                                              ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=835)
                                                    Buffers: shared hit=13
                                                    ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.030..0.040 rows=3 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=13
                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.026..0.035 rows=3 loops=1)
                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a… (457 chars)
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95… (398 chars)
                                                                Heap Blocks: exact=3
                                                                Buffers: shared hit=13
                                                                ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                      Buffers: shared hit=10
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            Buffers: shared hit=2
                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-… (335 chars)
                                                                            Buffers: shared hit=8
                                ->  Sort  (cost=3.49..3.66 rows=69 width=262) (actual time=6.680..6.683 rows=3 loops=1)
                                      Sort Key: acl_m.sort_key DESC, acl_m.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=1207
                                      ->  CTE Scan on acl_m  (cost=0.00..1.38 rows=69 width=262) (actual time=0.188..6.296 rows=835 loops=1)
                                            Buffers: shared hit=1207
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.709 ms
Execution Time: 10.349 ms
```

#### n_distinct 9000 (9000): page L1, district mixed (101 units, 900 tickets)

```
Nested Loop Left Join  (cost=669.62..1759.80 rows=1 width=1301) (actual time=7.523..9.653 rows=51 loops=1)
  Buffers: shared hit=3371
  ->  Nested Loop  (cost=669.32..1751.47 rows=1 width=1266) (actual time=7.484..9.393 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3218
        ->  Subquery Scan on x  (cost=669.04..1743.15 rows=1 width=1259) (actual time=7.465..9.195 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=3065
              ->  Limit  (cost=669.04..1742.01 rows=51 width=1259) (actual time=7.460..9.123 rows=51 loops=1)
                    Buffers: shared hit=3065
                    ->  Merge Append  (cost=669.04..4182.48 rows=167 width=1259) (actual time=7.458..9.107 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=3065
                          ->  Limit  (cost=0.42..3245.56 rows=51 width=1259) (actual time=0.128..1.721 rows=48 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..525012.33 rows=8251 width=1259) (actual time=0.127..1.710 rows=48 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..525012.33 rows=8251 width=1251) (actual time=0.120..1.675 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 725
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..204.85 rows=51 width=1259) (actual time=0.024..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..653.49 rows=163 width=1259) (actual time=0.023..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..653.49 rows=163 width=1251) (actual time=0.020..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.023..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.67 rows=15 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.67 rows=15 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.120..0.128 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.119..0.126 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.079..0.115 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.076..0.110 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.054..0.059 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.042..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.034..0.036 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.015 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=622.97..623.09 rows=49 width=1259) (actual time=7.158..7.174 rows=3 loops=1)
                                Buffers: shared hit=2246
                                ->  Sort  (cost=622.97..623.09 rows=49 width=1259) (actual time=7.157..7.173 rows=3 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 191kB
                                      Buffers: shared hit=2246
                                      ->  Result  (cost=1.15..621.59 rows=49 width=1259) (actual time=0.069..6.525 rows=637 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=2246
                                            ->  Nested Loop  (cost=1.15..621.59 rows=49 width=1259) (actual time=0.066..6.281 rows=637 loops=1)
                                                  Buffers: shared hit=2246
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.024..0.086 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.013 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.017..0.048 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=0.72..88.06 rows=7 width=1259) (actual time=0.012..0.058 rows=6 loops=101)
                                                        Buffers: shared hit=2241
                                                        ->  Result  (cost=0.72..88.06 rows=7 width=1259) (actual time=0.011..0.057 rows=6 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=2241
                                                              ->  Nested Loop Anti Join  (cost=0.72..88.06 rows=7 width=1251) (actual time=0.010..0.053 rows=6 loops=101)
                                                                    Buffers: shared hit=2241
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..57.76 rows=7 width=1251) (actual time=0.007..0.022 rows=6 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 0
                                                                          Buffers: shared hit=966
                                                                    ->  Result  (cost=0.29..4.32 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=637)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=1275
                                                                          ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..4.32 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=637)
                                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d… (414 chars)
                                                                                Rows Removed by Filter: 0
                                                                                Heap Fetches: 0
                                                                                Buffers: shared hit=1275
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.479 ms
Execution Time: 9.970 ms
```

#### count-only caller: CC count, area (10 units, 220 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=0.780..0.792 rows=1 loops=1)
  Buffers: shared hit=257
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.111..0.763 rows=220 loops=1)
        Buffers: shared hit=257
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.110..0.732 rows=220 loops=1)
              Buffers: shared hit=257
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.110..0.696 rows=220 loops=1)
                    Buffers: shared hit=257
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.011 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.025..0.026 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.025..0.025 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.014..0.014 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.014..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.011..0.015 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.011..0.014 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.011..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.010..0.012 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.009..0.011 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.008..0.009 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.046..0.596 rows=220 loops=1)
                          Buffers: shared hit=247
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.046..0.534 rows=220 loops=1)
                                Buffers: shared hit=247
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.046..0.507 rows=220 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=247
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.045..0.469 rows=220 loops=1)
                                            Buffers: shared hit=247
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.019..0.027 rows=10 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.005 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.013..0.018 rows=10 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.010..0.040 rows=22 loops=10)
                                                  Buffers: shared hit=243
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.010..0.037 rows=22 loops=10)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=243
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.009..0.033 rows=22 loops=10)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=243
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.008..0.025 rows=22 loops=10)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=241
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=220)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.004..0.005 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.004 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.931 ms
Execution Time: 0.997 ms
```

#### page W0, district (101 units, 3379 tickets)

```
Nested Loop Left Join  (cost=47.37..1867.45 rows=1 width=1301) (actual time=0.782..7.971 rows=51 loops=1)
  Buffers: shared hit=4622
  ->  Nested Loop  (cost=47.07..1859.12 rows=1 width=1266) (actual time=0.761..7.661 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4469
        ->  Subquery Scan on x  (cost=46.79..1850.80 rows=1 width=1259) (actual time=0.747..7.414 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4316
              ->  Limit  (cost=46.79..1849.66 rows=51 width=1259) (actual time=0.744..7.337 rows=51 loops=1)
                    Buffers: shared hit=4316
                    ->  Merge Append  (cost=46.79..4819.08 rows=135 width=1259) (actual time=0.742..7.294 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4316
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.122..1.192 rows=44 loops=1)
                                Buffers: shared hit=746
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.121..1.181 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=746
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.116..1.150 rows=44 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=746
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.023..0.028 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.022..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.019..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.096..0.105 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.095..0.104 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.063..0.094 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.061..0.089 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.045..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.042 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1427.09 rows=51 width=1259) (actual time=0.481..5.915 rows=7 loops=1)
                                Buffers: shared hit=3534
                                ->  Result  (cost=0.72..1562656.28 rows=55873 width=1259) (actual time=0.477..5.908 rows=7 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=3534
                                      ->  Nested Loop Anti Join  (cost=0.72..1562656.28 rows=55873 width=1251) (actual time=0.474..5.899 rows=7 loops=1)
                                            Buffers: shared hit=3534
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1543573.86 rows=55873 width=1251) (actual time=0.462..5.830 rows=7 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 942
                                                  Buffers: shared hit=3520
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.004..0.004 rows=0 loops=853)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=2560
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=853)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=2560
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=7)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                  Buffers: shared hit=14
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=7)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=14
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.810 ms
Execution Time: 8.253 ms
```

#### page L1, district (101 units, 3379 tickets)

```
Nested Loop Left Join  (cost=1435.24..2716.52 rows=1 width=1301) (actual time=9.476..11.550 rows=51 loops=1)
  Buffers: shared hit=3211
  ->  Nested Loop  (cost=1434.95..2708.19 rows=1 width=1266) (actual time=9.453..11.268 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3058
        ->  Subquery Scan on x  (cost=1434.66..2699.88 rows=1 width=1259) (actual time=9.433..11.072 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2905
              ->  Limit  (cost=1434.66..2698.73 rows=51 width=1259) (actual time=9.429..10.994 rows=51 loops=1)
                    Buffers: shared hit=2905
                    ->  Merge Append  (cost=1434.66..4780.72 rows=135 width=1259) (actual time=9.428..10.983 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2905
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.113..1.613 rows=44 loops=1)
                                Buffers: shared hit=746
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.113..1.604 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=746
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.105..1.570 rows=44 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=746
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.024..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.022..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.019..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.017..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.017..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.130..0.136 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.130..0.135 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.079..0.115 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.075..0.110 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.060 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.050 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1388.60..1388.73 rows=51 width=1259) (actual time=9.137..9.153 rows=7 loops=1)
                                Buffers: shared hit=2123
                                ->  Sort  (cost=1388.60..1388.95 rows=140 width=1259) (actual time=9.136..9.151 rows=7 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 181kB
                                      Buffers: shared hit=2123
                                      ->  Result  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.105..7.604 rows=1806 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=2123
                                            ->  Nested Loop  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.103..7.004 rows=1806 loops=1)
                                                  Buffers: shared hit=2123
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.030..0.102 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.008..0.009 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.020..0.068 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.012..0.062 rows=18 loops=101)
                                                        Buffers: shared hit=2118
                                                        ->  Result  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.011..0.059 rows=18 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=2118
                                                              ->  Nested Loop Anti Join  (cost=21.93..196.84 rows=20 width=1251) (actual time=0.010..0.051 rows=18 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 54
                                                                    Buffers: shared hit=2118
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.53 rows=20 width=1251) (actual time=0.008..0.030 rows=18 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 1
                                                                          Buffers: shared hit=2105
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=1806)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.024..0.032 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.023..0.030 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.018..0.022 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.849 ms
Execution Time: 11.972 ms
```

#### count-only caller: CC count, district (101 units, 3379 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=3.672..3.693 rows=1 loops=1)
  Buffers: shared hit=1088
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.149..3.540 rows=1001 loops=1)
        Buffers: shared hit=1088
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.148..3.361 rows=1001 loops=1)
              Buffers: shared hit=1088
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.147..3.180 rows=1001 loops=1)
                    Buffers: shared hit=1088
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.016..0.018 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.014..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.030..0.033 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.030..0.032 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.029..0.031 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.027..0.027 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.016..0.018 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.016..0.017 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.013..0.018 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.013..0.018 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.013..0.017 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.012..0.016 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.011..0.013 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.008..0.010 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.070..2.941 rows=1001 loops=1)
                          Buffers: shared hit=1078
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.069..2.773 rows=1001 loops=1)
                                Buffers: shared hit=1078
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.069..2.631 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1078
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.068..2.405 rows=1001 loops=1)
                                            Buffers: shared hit=1078
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.023..0.048 rows=41 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.006 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.016..0.031 rows=41 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.011..0.053 rows=24 loops=41)
                                                  Buffers: shared hit=1074
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.010..0.050 rows=24 loops=41)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1074
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.009..0.044 rows=24 loops=41)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1074
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.008..0.035 rows=24 loops=41)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1072
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.004..0.006 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.004 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.799 ms
Execution Time: 3.884 ms
```

#### count-only caller: CC count, district thin (101 units, 600 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=3.125..3.149 rows=1 loops=1)
  Buffers: shared hit=901
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.120..3.056 rows=600 loops=1)
        Buffers: shared hit=901
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.118..2.957 rows=600 loops=1)
              Buffers: shared hit=901
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.118..2.838 rows=600 loops=1)
                    Buffers: shared hit=901
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.012..0.014 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.013 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.027..0.029 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.026..0.028 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.026..0.027 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.025..0.025 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.014..0.016 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.014..0.015 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.014..0.015 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.013..0.021 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.013..0.020 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.013..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.012..0.018 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.011..0.017 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.010..0.014 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.008..0.009 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.050..2.650 rows=600 loops=1)
                          Buffers: shared hit=891
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.050..2.546 rows=600 loops=1)
                                Buffers: shared hit=891
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.050..2.459 rows=600 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=891
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.049..2.341 rows=600 loops=1)
                                            Buffers: shared hit=891
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.021..0.073 rows=101 loops=1)
                                                  Buffers: shared hit=5
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.015..0.041 rows=101 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.009..0.021 rows=6 loops=101)
                                                  Buffers: shared hit=886
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.009..0.020 rows=6 loops=101)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=886
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.008..0.018 rows=6 loops=101)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=886
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.007..0.015 rows=6 loops=101)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=884
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=600)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.004..0.006 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.004 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.560 ms
Execution Time: 3.314 ms
```

#### page W0, district hot (101 units, 22996 tickets)

```
Nested Loop Left Join  (cost=47.37..1867.45 rows=1 width=1301) (actual time=0.344..3.812 rows=51 loops=1)
  Buffers: shared hit=2486
  ->  Nested Loop  (cost=47.07..1859.12 rows=1 width=1266) (actual time=0.325..3.552 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2333
        ->  Subquery Scan on x  (cost=46.79..1850.80 rows=1 width=1259) (actual time=0.310..3.367 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2180
              ->  Limit  (cost=46.79..1849.66 rows=51 width=1259) (actual time=0.307..3.298 rows=51 loops=1)
                    Buffers: shared hit=2180
                    ->  Merge Append  (cost=46.79..4819.08 rows=135 width=1259) (actual time=0.306..3.285 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2180
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.105..0.595 rows=25 loops=1)
                                Buffers: shared hit=427
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.104..0.588 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=427
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.099..0.570 rows=25 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=427
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.025..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.024..0.029 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.021..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.020..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.016..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.093 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.088..0.092 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.059..0.083 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.057..0.080 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.045 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.038 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1427.09 rows=51 width=1259) (actual time=0.063..2.517 rows=26 loops=1)
                                Buffers: shared hit=1717
                                ->  Result  (cost=0.72..1562656.28 rows=55873 width=1259) (actual time=0.062..2.511 rows=26 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1717
                                      ->  Nested Loop Anti Join  (cost=0.72..1562656.28 rows=55873 width=1251) (actual time=0.060..2.498 rows=26 loops=1)
                                            Buffers: shared hit=1717
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1543573.86 rows=55873 width=1251) (actual time=0.051..2.309 rows=26 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 422
                                                  Buffers: shared hit=1665
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=403)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=1210
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=403)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=1210
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=26)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                  Buffers: shared hit=52
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=26)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=52
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.099 ms
Execution Time: 4.034 ms
```

#### page L1, district hot (101 units, 22996 tickets)

```
Nested Loop Left Join  (cost=1435.24..2716.52 rows=1 width=1301) (actual time=7.446..8.682 rows=51 loops=1)
  Buffers: shared hit=2527
  ->  Nested Loop  (cost=1434.95..2708.19 rows=1 width=1266) (actual time=7.427..8.431 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2374
        ->  Subquery Scan on x  (cost=1434.66..2699.88 rows=1 width=1259) (actual time=7.407..8.271 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2221
              ->  Limit  (cost=1434.66..2698.73 rows=51 width=1259) (actual time=7.403..8.216 rows=51 loops=1)
                    Buffers: shared hit=2221
                    ->  Merge Append  (cost=1434.66..4780.72 rows=135 width=1259) (actual time=7.402..8.206 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2221
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.131..0.884 rows=25 loops=1)
                                Buffers: shared hit=427
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.130..0.879 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=427
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.117..0.853 rows=25 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=427
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.027..0.032 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.027..0.031 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.022..0.025 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.021..0.021 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.116..0.122 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.115..0.121 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.077..0.108 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.073..0.103 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.056..0.061 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.042..0.050 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1388.60..1388.73 rows=51 width=1259) (actual time=7.097..7.114 rows=26 loops=1)
                                Buffers: shared hit=1758
                                ->  Sort  (cost=1388.60..1388.95 rows=140 width=1259) (actual time=7.096..7.109 rows=26 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 173kB
                                      Buffers: shared hit=1758
                                      ->  Result  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.274..5.980 rows=1400 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=1758
                                            ->  Nested Loop  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.271..5.507 rows=1400 loops=1)
                                                  Buffers: shared hit=1758
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.025..0.078 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.018..0.048 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.011..0.049 rows=14 loops=101)
                                                        Buffers: shared hit=1753
                                                        ->  Result  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.010..0.046 rows=14 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=1753
                                                              ->  Nested Loop Anti Join  (cost=21.93..196.84 rows=20 width=1251) (actual time=0.009..0.041 rows=14 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 42
                                                                    Buffers: shared hit=1753
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.53 rows=20 width=1251) (actual time=0.008..0.025 rows=14 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 1
                                                                          Buffers: shared hit=1740
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=1400)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.032..0.041 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.029..0.037 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.195 ms
Execution Time: 8.971 ms
```

#### count-only caller: CC count, district hot (101 units, 22996 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=2.953..2.971 rows=1 loops=1)
  Buffers: shared hit=1083
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.163..2.850 rows=1001 loops=1)
        Buffers: shared hit=1083
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.162..2.725 rows=1001 loops=1)
              Buffers: shared hit=1083
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.162..2.560 rows=1001 loops=1)
                    Buffers: shared hit=1083
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.012..0.014 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.013 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.026..0.028 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.026..0.027 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.025..0.026 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.014..0.016 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.013..0.015 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.013..0.017 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.012..0.017 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.012..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.011..0.014 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.010..0.012 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.009..0.010 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.008..0.009 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.096..2.354 rows=1001 loops=1)
                          Buffers: shared hit=1073
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.096..2.103 rows=1001 loops=1)
                                Buffers: shared hit=1073
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.096..1.981 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1073
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.095..1.780 rows=1001 loops=1)
                                            Buffers: shared hit=1073
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.023..0.035 rows=31 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.005 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.017..0.023 rows=31 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.008..0.051 rows=32 loops=31)
                                                  Buffers: shared hit=1069
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.008..0.047 rows=32 loops=31)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1069
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.006..0.042 rows=32 loops=31)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1069
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.006..0.032 rows=32 loops=31)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1067
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.004..0.005 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.004 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.450 ms
Execution Time: 3.130 ms
```

#### page W0, region (1011 units, 37930 tickets)

```
Nested Loop Left Join  (cost=47.37..1867.45 rows=1 width=1301) (actual time=0.367..4.329 rows=51 loops=1)
  Buffers: shared hit=1977
  ->  Nested Loop  (cost=47.07..1859.12 rows=1 width=1266) (actual time=0.351..3.892 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1824
        ->  Subquery Scan on x  (cost=46.79..1850.80 rows=1 width=1259) (actual time=0.340..3.669 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1671
              ->  Limit  (cost=46.79..1849.66 rows=51 width=1259) (actual time=0.338..3.593 rows=51 loops=1)
                    Buffers: shared hit=1671
                    ->  Merge Append  (cost=46.79..4819.08 rows=135 width=1259) (actual time=0.337..3.579 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1671
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.114..0.785 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.113..0.777 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.107..0.755 rows=21 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.026..0.028 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.025..0.026 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.022..0.022 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.023..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.020..0.021 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.109..0.114 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.108..0.113 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.074..0.103 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.071..0.098 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.051..0.055 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.046 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1427.09 rows=51 width=1259) (actual time=0.061..2.594 rows=31 loops=1)
                                Buffers: shared hit=1286
                                ->  Result  (cost=0.72..1562656.28 rows=55873 width=1259) (actual time=0.060..2.585 rows=31 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1286
                                      ->  Nested Loop Anti Join  (cost=0.72..1562656.28 rows=55873 width=1251) (actual time=0.057..2.567 rows=31 loops=1)
                                            Buffers: shared hit=1286
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1543573.86 rows=55873 width=1251) (actual time=0.046..2.357 rows=31 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 298
                                                  Buffers: shared hit=1224
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.005..0.005 rows=0 loops=296)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=889
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=296)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=889
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=31)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                  Buffers: shared hit=62
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=31)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=62
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.749 ms
Execution Time: 4.614 ms
```

#### page L1, region (1011 units, 37930 tickets)

```
Nested Loop Left Join  (cost=1435.24..2716.52 rows=1 width=1301) (actual time=99.063..100.216 rows=51 loops=1)
  Buffers: shared hit=21587
  ->  Nested Loop  (cost=1434.95..2708.19 rows=1 width=1266) (actual time=99.042..99.972 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21434
        ->  Subquery Scan on x  (cost=1434.66..2699.88 rows=1 width=1259) (actual time=99.021..99.813 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21281
              ->  Limit  (cost=1434.66..2698.73 rows=51 width=1259) (actual time=99.017..99.757 rows=51 loops=1)
                    Buffers: shared hit=21281
                    ->  Merge Append  (cost=1434.66..4780.72 rows=135 width=1259) (actual time=99.015..99.747 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21281
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.143..0.815 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.142..0.810 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.135..0.791 rows=21 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.027..0.029 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.022..0.022 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.017..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.016..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.127..0.135 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.127..0.134 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.082..0.117 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.078..0.112 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.061..0.066 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.048..0.056 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.040..0.042 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1388.60..1388.73 rows=51 width=1259) (actual time=98.694..98.717 rows=31 loops=1)
                                Buffers: shared hit=20896
                                ->  Sort  (cost=1388.60..1388.95 rows=140 width=1259) (actual time=98.693..98.712 rows=31 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 191kB
                                      Buffers: shared hit=20896
                                      ->  Result  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.122..83.003 rows=17706 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=20896
                                            ->  Nested Loop  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.119..76.842 rows=17706 loops=1)
                                                  Buffers: shared hit=20896
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.030..0.701 rows=1011 loops=1)
                                                        Buffers: shared hit=15
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.022..0.438 rows=1011 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=15
                                                  ->  Limit  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.010..0.069 rows=18 loops=1011)
                                                        Buffers: shared hit=20881
                                                        ->  Result  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.010..0.066 rows=18 loops=1011)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=20881
                                                              ->  Nested Loop Anti Join  (cost=21.93..196.84 rows=20 width=1251) (actual time=0.008..0.058 rows=18 loops=1011)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 53
                                                                    Buffers: shared hit=20881
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.53 rows=20 width=1251) (actual time=0.007..0.036 rows=18 loops=1011)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 1
                                                                          Buffers: shared hit=20868
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=17706)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.020..0.025 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.022 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.460 ms
Execution Time: 100.518 ms
```

#### count-only caller: CC count, region (1011 units, 37930 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=3.597..3.620 rows=1 loops=1)
  Buffers: shared hit=1096
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.328..3.478 rows=1001 loops=1)
        Buffers: shared hit=1096
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.326..3.342 rows=1001 loops=1)
              Buffers: shared hit=1096
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.316..3.147 rows=1001 loops=1)
                    Buffers: shared hit=1096
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.044..0.046 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.042..0.044 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.041..0.043 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.099..0.104 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.098..0.102 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.097..0.101 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.089..0.092 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.046..0.047 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.045..0.046 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.045..0.046 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.043..0.044 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.021..0.026 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.021..0.025 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.020..0.024 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.016..0.020 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.016..0.018 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.013..0.015 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.102..2.780 rows=1001 loops=1)
                          Buffers: shared hit=1086
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.101..2.601 rows=1001 loops=1)
                                Buffers: shared hit=1086
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.101..2.471 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1086
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.099..2.289 rows=1001 loops=1)
                                            Buffers: shared hit=1086
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.034..0.050 rows=38 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.008..0.008 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.024..0.031 rows=38 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.009..0.054 rows=26 loops=38)
                                                  Buffers: shared hit=1082
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.008..0.051 rows=26 loops=38)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1082
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.007..0.045 rows=26 loops=38)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1082
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.006..0.036 rows=26 loops=38)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1080
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.006..0.008 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.004..0.006 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.003..0.005 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.993 ms
Execution Time: 3.819 ms
```

#### page W0, root (10111 units, 380048 tickets)

```
Nested Loop Left Join  (cost=47.37..1867.45 rows=1 width=1301) (actual time=0.390..1.842 rows=51 loops=1)
  Buffers: shared hit=712
  ->  Nested Loop  (cost=47.07..1859.12 rows=1 width=1266) (actual time=0.371..1.568 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=559
        ->  Subquery Scan on x  (cost=46.79..1850.80 rows=1 width=1259) (actual time=0.359..1.365 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=406
              ->  Limit  (cost=46.79..1849.66 rows=51 width=1259) (actual time=0.357..1.301 rows=51 loops=1)
                    Buffers: shared hit=406
                    ->  Merge Append  (cost=46.79..4819.08 rows=135 width=1259) (actual time=0.355..1.287 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=406
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.124..0.193 rows=2 loops=1)
                                Buffers: shared hit=65
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.123..0.191 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=65
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.117..0.183 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=65
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.028..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.024..0.025 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.127..0.135 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.127..0.134 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.082..0.123 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.076..0.115 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.060 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.037..0.048 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.016..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1427.09 rows=51 width=1259) (actual time=0.047..0.880 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=0.72..1562656.28 rows=55873 width=1259) (actual time=0.047..0.866 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=0.72..1562656.28 rows=55873 width=1251) (actual time=0.044..0.842 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1543573.86 rows=55873 width=1251) (actual time=0.032..0.483 rows=50 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=206
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.007..0.007 rows=1 loops=50)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=151
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.005..0.005 rows=1 loops=50)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.034 ms
Execution Time: 2.101 ms
```

#### page L1, root (10111 units, 380048 tickets)

```
Nested Loop Left Join  (cost=1435.24..2716.52 rows=1 width=1301) (actual time=667.613..668.047 rows=51 loops=1)
  Buffers: shared hit=203596
  ->  Nested Loop  (cost=1434.95..2708.19 rows=1 width=1266) (actual time=667.592..667.860 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=203443
        ->  Subquery Scan on x  (cost=1434.66..2699.88 rows=1 width=1259) (actual time=667.568..667.718 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=203290
              ->  Limit  (cost=1434.66..2698.73 rows=51 width=1259) (actual time=667.564..667.678 rows=51 loops=1)
                    Buffers: shared hit=203290
                    ->  Merge Append  (cost=1434.66..4780.72 rows=135 width=1259) (actual time=667.563..667.671 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=203290
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.144..0.200 rows=2 loops=1)
                                Buffers: shared hit=65
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.143..0.198 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=65
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.138..0.192 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=65
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.024..0.026 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.015..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.015..0.016 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.013..0.014 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.086..0.093 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.086..0.092 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.057..0.083 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.056..0.079 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.041..0.045 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.037 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1388.60..1388.73 rows=51 width=1259) (actual time=667.287..667.305 rows=50 loops=1)
                                Buffers: shared hit=203190
                                ->  Sort  (cost=1388.60..1388.95 rows=140 width=1259) (actual time=667.286..667.299 rows=50 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 185kB
                                      Buffers: shared hit=203190
                                      ->  Result  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.111..558.810 rows=170441 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=203190
                                            ->  Nested Loop  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.109..512.516 rows=170441 loops=1)
                                                  Buffers: shared hit=203190
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.022..3.605 rows=10111 loops=1)
                                                        Buffers: shared hit=110
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.006 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.016..2.050 rows=10111 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=110
                                                  ->  Limit  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.005..0.045 rows=17 loops=10111)
                                                        Buffers: shared hit=203080
                                                        ->  Result  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.005..0.043 rows=17 loops=10111)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=203080
                                                              ->  Nested Loop Anti Join  (cost=21.93..196.84 rows=20 width=1251) (actual time=0.004..0.038 rows=17 loops=10111)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 51
                                                                    Buffers: shared hit=203080
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.53 rows=20 width=1251) (actual time=0.004..0.023 rows=17 loops=10111)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 1
                                                                          Buffers: shared hit=203067
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=170441)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.019..0.024 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.021 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.882 ms
Execution Time: 668.258 ms
```

#### count-only caller: CC count, root (10111 units, 380048 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=2.530..2.543 rows=1 loops=1)
  Buffers: shared hit=1050
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.116..2.446 rows=1001 loops=1)
        Buffers: shared hit=1050
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.115..2.336 rows=1001 loops=1)
              Buffers: shared hit=1050
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.115..2.187 rows=1001 loops=1)
                    Buffers: shared hit=1050
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.010..0.012 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.022..0.022 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.021..0.022 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.010..0.014 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.013 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.009..0.012 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.008..0.011 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.007..0.009 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.006..0.008 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.059..2.009 rows=1001 loops=1)
                          Buffers: shared hit=1040
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.058..1.878 rows=1001 loops=1)
                                Buffers: shared hit=1040
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.058..1.769 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1040
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.057..1.620 rows=1001 loops=1)
                                            Buffers: shared hit=1040
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.020..0.027 rows=24 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.014..0.017 rows=24 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.005..0.060 rows=42 loops=24)
                                                  Buffers: shared hit=1036
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.005..0.056 rows=42 loops=24)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1036
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.004..0.050 rows=42 loops=24)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1036
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.004..0.040 rows=42 loops=24)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1034
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.004..0.005 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.004 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.246 ms
Execution Time: 2.674 ms
```

#### count-only caller: CC count, district mixed (101 units, 900 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=3.105..3.120 rows=1 loops=1)
  Buffers: shared hit=1206
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.104..3.033 rows=900 loops=1)
        Buffers: shared hit=1206
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.103..2.937 rows=900 loops=1)
              Buffers: shared hit=1206
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.102..2.807 rows=900 loops=1)
                    Buffers: shared hit=1206
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.010..0.012 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.009..0.010 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.009..0.010 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.024..0.026 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.024..0.025 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.024..0.025 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.013..0.015 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.011..0.014 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.013 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.009..0.011 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.009..0.011 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.008..0.009 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.044..2.639 rows=900 loops=1)
                          Buffers: shared hit=1196
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.044..2.525 rows=900 loops=1)
                                Buffers: shared hit=1196
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.043..2.428 rows=900 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1196
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.042..2.296 rows=900 loops=1)
                                            Buffers: shared hit=1196
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.019..0.049 rows=101 loops=1)
                                                  Buffers: shared hit=5
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.013..0.029 rows=101 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.006..0.021 rows=9 loops=101)
                                                  Buffers: shared hit=1191
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.006..0.020 rows=9 loops=101)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1191
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.005..0.018 rows=9 loops=101)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1191
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.005..0.016 rows=9 loops=101)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1189
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=900)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.004 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.192 ms
Execution Time: 3.284 ms
```

#### page W0, uniform m=1001 (101 units, 1001 tickets)

```
Nested Loop Left Join  (cost=47.37..1867.45 rows=1 width=1301) (actual time=1.241..7.842 rows=51 loops=1)
  Buffers: shared hit=4887
  ->  Nested Loop  (cost=47.07..1859.12 rows=1 width=1266) (actual time=1.220..7.586 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4734
        ->  Subquery Scan on x  (cost=46.79..1850.80 rows=1 width=1259) (actual time=1.206..7.393 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4581
              ->  Limit  (cost=46.79..1849.66 rows=51 width=1259) (actual time=1.203..7.333 rows=51 loops=1)
                    Buffers: shared hit=4581
                    ->  Merge Append  (cost=46.79..4819.08 rows=135 width=1259) (actual time=1.202..7.320 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4581
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.108..1.186 rows=47 loops=1)
                                Buffers: shared hit=766
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.108..1.178 rows=47 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=766
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.102..1.151 rows=47 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 709
                                            Buffers: shared hit=766
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.023..0.027 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.023..0.026 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.020..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.105..0.109 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.104..0.108 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.069..0.096 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.067..0.092 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.049..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.036..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1427.09 rows=51 width=1259) (actual time=0.942..5.956 rows=4 loops=1)
                                Buffers: shared hit=3779
                                ->  Result  (cost=0.72..1562656.28 rows=55873 width=1259) (actual time=0.941..5.952 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=3779
                                      ->  Nested Loop Anti Join  (cost=0.72..1562656.28 rows=55873 width=1251) (actual time=0.938..5.947 rows=4 loops=1)
                                            Buffers: shared hit=3779
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1543573.86 rows=55873 width=1251) (actual time=0.923..5.906 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 1012
                                                  Buffers: shared hit=3771
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.004..0.004 rows=0 loops=914)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=2743
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=914)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{57c53194-0147-c7be-2dc3-fc23cc4d837a}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=2743
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=4)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                  Buffers: shared hit=8
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=4)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.992 ms
Execution Time: 8.077 ms
```

#### page L1, uniform m=1001 (101 units, 1001 tickets)

```
Nested Loop Left Join  (cost=1435.24..2716.52 rows=1 width=1301) (actual time=4.153..5.623 rows=51 loops=1)
  Buffers: shared hit=2439
  ->  Nested Loop  (cost=1434.95..2708.19 rows=1 width=1266) (actual time=4.135..5.431 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2286
        ->  Subquery Scan on x  (cost=1434.66..2699.88 rows=1 width=1259) (actual time=4.122..5.280 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2133
              ->  Limit  (cost=1434.66..2698.73 rows=51 width=1259) (actual time=4.118..5.237 rows=51 loops=1)
                    Buffers: shared hit=2133
                    ->  Merge Append  (cost=1434.66..4780.72 rows=135 width=1259) (actual time=4.117..5.229 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2133
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.096..1.175 rows=47 loops=1)
                                Buffers: shared hit=766
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.096..1.169 rows=47 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=766
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.092..1.148 rows=47 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 709
                                            Buffers: shared hit=766
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.019..0.023 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.018..0.022 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.016..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.014..0.014 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.012..0.013 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.086 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.085 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.054..0.077 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.053..0.075 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.043 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.036 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1388.60..1388.73 rows=51 width=1259) (actual time=3.902..3.911 rows=4 loops=1)
                                Buffers: shared hit=1331
                                ->  Sort  (cost=1388.60..1388.95 rows=140 width=1259) (actual time=3.902..3.910 rows=4 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 189kB
                                      Buffers: shared hit=1331
                                      ->  Result  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.059..3.275 rows=944 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=1331
                                            ->  Nested Loop  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.058..3.009 rows=944 loops=1)
                                                  Buffers: shared hit=1331
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.019..0.051 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.003..0.004 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.014..0.031 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.008..0.027 rows=9 loops=101)
                                                        Buffers: shared hit=1326
                                                        ->  Result  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.008..0.025 rows=9 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=1326
                                                              ->  Nested Loop Anti Join  (cost=21.93..196.84 rows=20 width=1251) (actual time=0.007..0.022 rows=9 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 28
                                                                    Buffers: shared hit=1326
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.53 rows=20 width=1251) (actual time=0.005..0.014 rows=9 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 1
                                                                          Buffers: shared hit=1313
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=944)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.015..0.020 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.018 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.860 ms
Execution Time: 5.828 ms
```

#### count-only caller: CC count, uniform m=1001 (101 units, 1001 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=3.068..3.084 rows=1 loops=1)
  Buffers: shared hit=1330
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.191..2.978 rows=1001 loops=1)
        Buffers: shared hit=1330
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.189..2.865 rows=1001 loops=1)
              Buffers: shared hit=1330
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.188..2.714 rows=1001 loops=1)
                    Buffers: shared hit=1330
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.012..0.014 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.003..0.004 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.071..0.072 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.070..0.071 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.070..0.071 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.069..0.069 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.015..0.017 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.015..0.015 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.013..0.016 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.013..0.016 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.013..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.011..0.013 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.010..0.012 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.008..0.009 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.076..2.475 rows=1001 loops=1)
                          Buffers: shared hit=1320
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.076..2.342 rows=1001 loops=1)
                                Buffers: shared hit=1320
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.075..2.231 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1320
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.074..2.077 rows=1001 loops=1)
                                            Buffers: shared hit=1320
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.045..0.076 rows=101 loops=1)
                                                  Buffers: shared hit=5
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.006 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.037..0.052 rows=101 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.007..0.018 rows=10 loops=101)
                                                  Buffers: shared hit=1315
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.007..0.017 rows=10 loops=101)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1315
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.006..0.015 rows=10 loops=101)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1315
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.005..0.012 rows=10 loops=101)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1313
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.004..0.007 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.006 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.003..0.005 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.665 ms
Execution Time: 3.250 ms
```

#### page W0, uniform m=3000 (101 units, 3000 tickets)

```
Nested Loop Left Join  (cost=47.37..1867.45 rows=1 width=1301) (actual time=0.538..4.205 rows=51 loops=1)
  Buffers: shared hit=3945
  ->  Nested Loop  (cost=47.07..1859.12 rows=1 width=1266) (actual time=0.523..4.033 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3792
        ->  Subquery Scan on x  (cost=46.79..1850.80 rows=1 width=1259) (actual time=0.513..3.905 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=3639
              ->  Limit  (cost=46.79..1849.66 rows=51 width=1259) (actual time=0.511..3.864 rows=51 loops=1)
                    Buffers: shared hit=3639
                    ->  Merge Append  (cost=46.79..4819.08 rows=135 width=1259) (actual time=0.510..3.855 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=3639
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.094..0.694 rows=47 loops=1)
                                Buffers: shared hit=766
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.094..0.688 rows=47 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=766
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.090..0.668 rows=47 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 709
                                            Buffers: shared hit=766
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.017..0.020 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.017..0.019 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.015..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.013..0.013 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.079..0.083 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.079..0.082 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.052..0.073 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.050..0.070 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.040 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.027..0.033 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1427.09 rows=51 width=1259) (actual time=0.303..3.028 rows=4 loops=1)
                                Buffers: shared hit=2837
                                ->  Result  (cost=0.72..1562656.28 rows=55873 width=1259) (actual time=0.302..3.026 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2837
                                      ->  Nested Loop Anti Join  (cost=0.72..1562656.28 rows=55873 width=1251) (actual time=0.301..3.023 rows=4 loops=1)
                                            Buffers: shared hit=2837
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1543573.86 rows=55873 width=1251) (actual time=0.293..2.996 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 759
                                                  Buffers: shared hit=2829
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=685)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=2056
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=685)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{d58eb9e3-a329-a6ba-b01c-a525874bdde5}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=2056
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.006 rows=0 loops=4)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                  Buffers: shared hit=8
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=4)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.701 ms
Execution Time: 4.383 ms
```

#### page L1, uniform m=3000 (101 units, 3000 tickets)

```
Nested Loop Left Join  (cost=1435.24..2716.52 rows=1 width=1301) (actual time=10.839..12.227 rows=51 loops=1)
  Buffers: shared hit=4464
  ->  Nested Loop  (cost=1434.95..2708.19 rows=1 width=1266) (actual time=10.822..12.060 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4311
        ->  Subquery Scan on x  (cost=1434.66..2699.88 rows=1 width=1259) (actual time=10.808..11.928 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4158
              ->  Limit  (cost=1434.66..2698.73 rows=51 width=1259) (actual time=10.805..11.885 rows=51 loops=1)
                    Buffers: shared hit=4158
                    ->  Merge Append  (cost=1434.66..4780.72 rows=135 width=1259) (actual time=10.804..11.877 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4158
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.103..1.147 rows=47 loops=1)
                                Buffers: shared hit=766
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.103..1.141 rows=47 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=766
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.097..1.119 rows=47 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 709
                                            Buffers: shared hit=766
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.024..0.028 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.024..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.021..0.023 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.086..0.090 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.086..0.089 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.059..0.081 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.057..0.078 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.045 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.037 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1388.60..1388.73 rows=51 width=1259) (actual time=10.569..10.576 rows=4 loops=1)
                                Buffers: shared hit=3356
                                ->  Sort  (cost=1388.60..1388.95 rows=140 width=1259) (actual time=10.568..10.575 rows=4 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=3356
                                      ->  Result  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.074..8.745 rows=2812 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=3356
                                            ->  Nested Loop  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.072..7.890 rows=2812 loops=1)
                                                  Buffers: shared hit=3356
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.021..0.053 rows=101 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.015..0.031 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=4
                                                  ->  Limit  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.008..0.070 rows=28 loops=101)
                                                        Buffers: shared hit=3352
                                                        ->  Result  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.008..0.066 rows=28 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=3352
                                                              ->  Nested Loop Anti Join  (cost=21.93..196.84 rows=20 width=1251) (actual time=0.007..0.059 rows=28 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 84
                                                                    Buffers: shared hit=3352
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.53 rows=20 width=1251) (actual time=0.006..0.034 rows=28 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 2
                                                                          Buffers: shared hit=3339
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=2812)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.016..0.020 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.018 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.798 ms
Execution Time: 12.436 ms
```

#### count-only caller: CC count, uniform m=3000 (101 units, 3000 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=2.819..2.836 rows=1 loops=1)
  Buffers: shared hit=1131
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.116..2.732 rows=1001 loops=1)
        Buffers: shared hit=1131
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.115..2.621 rows=1001 loops=1)
              Buffers: shared hit=1131
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.115..2.468 rows=1001 loops=1)
                    Buffers: shared hit=1131
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.010..0.013 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.009..0.011 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.009..0.010 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.023..0.025 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.023..0.024 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.023..0.024 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.012..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.011..0.015 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.011..0.015 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.010..0.012 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.009..0.011 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.008..0.009 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.057..2.283 rows=1001 loops=1)
                          Buffers: shared hit=1121
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.056..2.150 rows=1001 loops=1)
                                Buffers: shared hit=1121
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.056..2.037 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1121
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.055..1.882 rows=1001 loops=1)
                                            Buffers: shared hit=1121
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.022..0.032 rows=34 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.005 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.015..0.020 rows=34 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.009..0.049 rows=29 loops=34)
                                                  Buffers: shared hit=1117
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.008..0.046 rows=29 loops=34)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1117
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.007..0.042 rows=29 loops=34)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1117
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.007..0.034 rows=29 loops=34)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1115
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.005 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.261 ms
Execution Time: 2.971 ms
```

#### page W0, uniform m=10000 (101 units, 10000 tickets)

```
Nested Loop Left Join  (cost=47.37..1867.45 rows=1 width=1301) (actual time=0.276..3.131 rows=51 loops=1)
  Buffers: shared hit=2933
  ->  Nested Loop  (cost=47.07..1859.12 rows=1 width=1266) (actual time=0.263..2.955 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2780
        ->  Subquery Scan on x  (cost=46.79..1850.80 rows=1 width=1259) (actual time=0.254..2.808 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2627
              ->  Limit  (cost=46.79..1849.66 rows=51 width=1259) (actual time=0.253..2.768 rows=51 loops=1)
                    Buffers: shared hit=2627
                    ->  Merge Append  (cost=46.79..4819.08 rows=135 width=1259) (actual time=0.252..2.759 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2627
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.090..0.530 rows=35 loops=1)
                                Buffers: shared hit=553
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.089..0.525 rows=35 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=553
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.085..0.509 rows=35 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 510
                                            Buffers: shared hit=553
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.017..0.020 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.017..0.020 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.015..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.014..0.014 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.012..0.012 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.076..0.079 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.076..0.079 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.051..0.071 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.049..0.068 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.036..0.038 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.026..0.031 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1427.09 rows=51 width=1259) (actual time=0.052..2.099 rows=16 loops=1)
                                Buffers: shared hit=2038
                                ->  Result  (cost=0.72..1562656.28 rows=55873 width=1259) (actual time=0.052..2.095 rows=16 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2038
                                      ->  Nested Loop Anti Join  (cost=0.72..1562656.28 rows=55873 width=1251) (actual time=0.050..2.088 rows=16 loops=1)
                                            Buffers: shared hit=2038
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1543573.86 rows=55873 width=1251) (actual time=0.042..2.021 rows=16 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 526
                                                  Buffers: shared hit=2006
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=485)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=1456
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=485)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{9f2d6775-5705-baa0-3ece-2e38f6daea06}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=1456
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=16)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                  Buffers: shared hit=32
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=16)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=32
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.690 ms
Execution Time: 3.325 ms
```

#### page L1, uniform m=10000 (101 units, 10000 tickets)

```
Nested Loop Left Join  (cost=1435.24..2716.52 rows=1 width=1301) (actual time=18.087..19.205 rows=51 loops=1)
  Buffers: shared hit=6119
  ->  Nested Loop  (cost=1434.95..2708.19 rows=1 width=1266) (actual time=18.068..19.033 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=5966
        ->  Subquery Scan on x  (cost=1434.66..2699.88 rows=1 width=1259) (actual time=18.052..18.906 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=5813
              ->  Limit  (cost=1434.66..2698.73 rows=51 width=1259) (actual time=18.048..18.864 rows=51 loops=1)
                    Buffers: shared hit=5813
                    ->  Merge Append  (cost=1434.66..4780.72 rows=135 width=1259) (actual time=18.047..18.856 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=5813
                          ->  Limit  (cost=0.42..3206.46 rows=51 width=1259) (actual time=0.096..0.863 rows=35 loops=1)
                                Buffers: shared hit=553
                                ->  Result  (cost=0.42..524784.22 rows=8348 width=1259) (actual time=0.095..0.859 rows=35 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=553
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..524784.22 rows=8348 width=1251) (actual time=0.091..0.842 rows=35 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 510
                                            Buffers: shared hit=553
                          ->  Limit  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.019..0.023 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..76.96 rows=18 width=1259) (actual time=0.019..0.022 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.96 rows=18 width=1251) (actual time=0.017..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.015..0.016 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..61.78 rows=14 width=1259) (actual time=0.014..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.78 rows=14 width=1251) (actual time=0.013..0.013 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.088 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.087 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.054..0.079 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.053..0.076 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.042 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.028..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-877… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1388.60..1388.73 rows=51 width=1259) (actual time=17.832..17.845 rows=16 loops=1)
                                Buffers: shared hit=5224
                                ->  Sort  (cost=1388.60..1388.95 rows=140 width=1259) (actual time=17.831..17.843 rows=16 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 185kB
                                      Buffers: shared hit=5224
                                      ->  Result  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.062..14.780 rows=4590 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                            Buffers: shared hit=5224
                                            ->  Nested Loop  (cost=22.36..1383.93 rows=140 width=1259) (actual time=0.060..13.484 rows=4590 loops=1)
                                                  Buffers: shared hit=5224
                                                  ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.020..0.060 rows=101 loops=1)
                                                        Buffers: shared hit=5
                                                        ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.005 rows=1 loops=1)
                                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.015..0.036 rows=101 loops=1)
                                                              Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Limit  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.010..0.120 rows=45 loops=101)
                                                        Buffers: shared hit=5219
                                                        ->  Result  (cost=21.93..196.84 rows=20 width=1259) (actual time=0.009..0.115 rows=45 loops=101)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                              Buffers: shared hit=5219
                                                              ->  Nested Loop Anti Join  (cost=21.93..196.84 rows=20 width=1251) (actual time=0.008..0.103 rows=45 loops=101)
                                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                                    Rows Removed by Join Filter: 136
                                                                    Buffers: shared hit=5219
                                                                    ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.53 rows=20 width=1251) (actual time=0.007..0.062 rows=45 loops=101)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL… (497 chars)
                                                                          Rows Removed by Filter: 3
                                                                          Buffers: shared hit=5206
                                                                    ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=4590)
                                                                          Buffers: shared hit=13
                                                                          ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.015..0.020 rows=3 loops=1)
                                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                                Buffers: shared hit=13
                                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.014..0.018 rows=3 loops=1)
                                                                                      Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (prin… (479 chars)
                                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c… (420 chars)
                                                                                      Heap Blocks: exact=3
                                                                                      Buffers: shared hit=13
                                                                                      ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                                            Buffers: shared hit=10
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                                  Buffers: shared hit=2
                                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d… (357 chars)
                                                                                                  Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.034 ms
Execution Time: 19.419 ms
```

#### count-only caller: CC count, uniform m=10000 (101 units, 10000 tickets)

```
Aggregate  (cost=1391.67..1391.68 rows=1 width=8) (actual time=3.068..3.085 rows=1 loops=1)
  Buffers: shared hit=1063
  ->  Limit  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.107..2.978 rows=1001 loops=1)
        Buffers: shared hit=1063
        ->  Result  (cost=0.55..1389.38 rows=183 width=4) (actual time=0.106..2.864 rows=1001 loops=1)
              Buffers: shared hit=1063
              ->  Append  (cost=0.55..1387.55 rows=183 width=0) (actual time=0.106..2.710 rows=1001 loops=1)
                    Buffers: shared hit=1063
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.010..0.010 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.024..0.026 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.023..0.025 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.023..0.024 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.013..0.015 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.011..0.015 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.014 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.009..0.012 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.009..0.011 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.008..0.009 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.47..1222.03 rows=147 width=0) (actual time=0.046..2.522 rows=1001 loops=1)
                          Buffers: shared hit=1053
                          ->  Limit  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.046..2.370 rows=1001 loops=1)
                                Buffers: shared hit=1053
                                ->  Result  (cost=9.47..1220.56 rows=147 width=4) (actual time=0.046..2.256 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1053
                                      ->  Nested Loop  (cost=9.47..1220.56 rows=147 width=0) (actual time=0.045..2.050 rows=1001 loops=1)
                                            Buffers: shared hit=1053
                                            ->  Nested Loop  (cost=0.42..4.64 rows=7 width=16) (actual time=0.022..0.028 rows=11 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  Function Scan on unnest acl_u  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.005 rows=1 loops=1)
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.016..0.019 rows=11 loops=1)
                                                        Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = acl_u.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Limit  (cost=9.04..173.28 rows=21 width=4) (actual time=0.010..0.170 rows=91 loops=11)
                                                  Buffers: shared hit=1049
                                                  ->  Result  (cost=9.04..173.28 rows=21 width=4) (actual time=0.010..0.162 rows=91 loops=11)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=1049
                                                        ->  Nested Loop Anti Join  (cost=9.04..173.28 rows=21 width=0) (actual time=0.009..0.148 rows=91 loops=11)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              Buffers: shared hit=1049
                                                              ->  Index Scan using records_ou_list on records r_4  (cost=0.42..160.33 rows=21 width=32) (actual time=0.008..0.125 rows=91 loops=11)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)))
                                                                    Buffers: shared hit=1047
                                                              ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=2
                                                                    ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.003..0.005 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                                          Buffers: shared hit=2
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.002..0.004 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_… (326 chars)
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                                                Buffers: shared hit=2
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                                      Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.265 ms
Execution Time: 3.220 ms
```

#### count-only caller: today's W0 count (SP-1c), district (101 units, 3379 tickets)

```
Aggregate  (cost=15538.67..15538.68 rows=1 width=8) (actual time=515.917..515.933 rows=1 loops=1)
  Buffers: shared hit=584538
  ->  Limit  (cost=0.55..15526.16 rows=1001 width=4) (actual time=41.891..515.785 rows=1001 loops=1)
        Buffers: shared hit=584538
        ->  Result  (cost=0.55..16084.52 rows=1037 width=4) (actual time=41.890..515.664 rows=1001 loops=1)
              Buffers: shared hit=584538
              ->  Append  (cost=0.55..16074.15 rows=1037 width=0) (actual time=41.889..515.505 rows=1001 loops=1)
                    Buffers: shared hit=584538
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.009..0.011 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.009..0.010 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.008..0.009 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.023..0.025 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.023..0.024 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.023..0.023 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.011..0.016 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.011..0.015 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.011..0.015 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.010..0.013 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.009..0.012 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.008..0.010 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.007..0.009 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.04..15904.36 rows=1001 width=0) (actual time=41.832..515.306 rows=1001 loops=1)
                          Buffers: shared hit=584528
                          ->  Limit  (cost=9.04..15894.35 rows=1001 width=4) (actual time=41.831..515.156 rows=1001 loops=1)
                                Buffers: shared hit=584528
                                ->  Result  (cost=9.04..920182.72 rows=57984 width=4) (actual time=41.816..515.017 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=584528
                                      ->  Nested Loop Anti Join  (cost=9.04..920182.72 rows=57984 width=0) (actual time=41.815..514.848 rows=1001 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            Buffers: shared hit=584528
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..919300.32 rows=57984 width=32) (actual time=41.795..514.493 rows=1001 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (SubPlan 1))
                                                  Rows Removed by Filter: 180373
                                                  Buffers: shared hit=584526
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.002..0.002 rows=0 loops=181374)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=544123
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.001..0.001 rows=0 loops=181374)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=544123
                                            ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                  Buffers: shared hit=2
                                                  ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.012..0.014 rows=0 loops=1)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uu… (308 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.109 ms
Execution Time: 516.095 ms
```

#### count-only caller: today's W0 count (SP-1c), uniform m=1001 (101 units, 1001 tickets)

```
Aggregate  (cost=15538.67..15538.68 rows=1 width=8) (actual time=1407.022..1407.061 rows=1 loops=1)
  Buffers: shared hit=1212018
  ->  Limit  (cost=0.55..15526.16 rows=1001 width=4) (actual time=1.952..1406.753 rows=1001 loops=1)
        Buffers: shared hit=1212018
        ->  Result  (cost=0.55..16084.52 rows=1037 width=4) (actual time=1.950..1406.557 rows=1001 loops=1)
              Buffers: shared hit=1212018
              ->  Append  (cost=0.55..16074.15 rows=1037 width=0) (actual time=1.950..1406.305 rows=1001 loops=1)
                    Buffers: shared hit=1212018
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.010..0.014 rows=0 loops=1)
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.008..0.012 rows=0 loops=1)
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.008..0.010 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{}'::uuid[])))
                                            Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..77.05 rows=19 width=0) (actual time=0.021..0.024 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..76.86 rows=19 width=4) (actual time=0.020..0.022 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..76.86 rows=19 width=4) (actual time=0.020..0.021 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..76.86 rows=19 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..61.86 rows=15 width=0) (actual time=0.011..0.015 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..61.71 rows=15 width=4) (actual time=0.011..0.014 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..61.71 rows=15 width=4) (actual time=0.010..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..61.71 rows=15 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=13.05..21.11 rows=1 width=0) (actual time=0.010..0.021 rows=0 loops=1)
                          Buffers: shared hit=2
                          ->  Limit  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.020 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Result  (cost=13.05..21.10 rows=1 width=4) (actual time=0.010..0.019 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=2
                                      ->  Nested Loop  (cost=13.05..21.10 rows=1 width=0) (actual time=0.009..0.017 rows=0 loops=1)
                                            Buffers: shared hit=2
                                            ->  HashAggregate  (cost=12.62..12.63 rows=1 width=32) (actual time=0.009..0.015 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=2
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..12.61 rows=1 width=32) (actual time=0.007..0.012 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[]))… (302 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                        Buffers: shared hit=2
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.006..0.009 rows=0 loops=1)
                                                              Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635… (317 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=9.04..15904.36 rows=1001 width=0) (actual time=1.898..1405.980 rows=1001 loops=1)
                          Buffers: shared hit=1212008
                          ->  Limit  (cost=9.04..15894.35 rows=1001 width=4) (actual time=1.897..1405.716 rows=1001 loops=1)
                                Buffers: shared hit=1212008
                                ->  Result  (cost=9.04..920182.72 rows=57984 width=4) (actual time=1.897..1405.416 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                      Buffers: shared hit=1212008
                                      ->  Nested Loop Anti Join  (cost=9.04..920182.72 rows=57984 width=0) (actual time=1.895..1405.129 rows=1001 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            Buffers: shared hit=1212008
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..919300.32 rows=57984 width=32) (actual time=1.882..1404.520 rows=1001 loops=1)
                                                  Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (entity_definition_id = '01a1fccd-c9f8-7bec-8779-2635ee002f7e'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) AND (SubPlan 1))
                                                  Rows Removed by Filter: 374711
                                                  Buffers: shared hit=1212006
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=375712)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                          Buffers: shared hit=1127137
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=375712)
                                                                Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (ancestor_id = ANY ('{57c53194-0147-c7be-2dc3-fc23cc4d837a}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=1127137
                                            ->  Materialize  (cost=8.60..12.63 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                  Buffers: shared hit=2
                                                  ->  Result  (cost=8.60..12.62 rows=1 width=32) (actual time=0.010..0.015 rows=0 loops=1)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid)
                                                        Buffers: shared hit=2
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..12.62 rows=1 width=32) (actual time=0.007..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uu… (308 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                                              Buffers: shared hit=2
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=1 width=0) (actual time=0.006..0.009 rows=0 loops=1)
                                                                    Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = '98c56c0e-7b1e-4531-e31c-de61d4010134'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.001..0.001 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '978391de-7cc5-47e7-a34f-a58dba140516'::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Planning Time: 1.090 ms
Execution Time: 1407.204 ms
```

## 2026-11-21T06:46:09Z phase=e-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-it-pgb-noprep-423403037 nava-it-pgb-pooled-423403037 nava-it-pg-423403037 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (phase A (B1 key, no records_ou_list)): `SELECT 1`, 8 clients x 200: p50 1.17, p95 2.03 ms -> **PASS**. Foreign containers: `nava-it-pgb-noprep-423403037 nava-it-pgb-pooled-423403037 nava-it-pg-423403037 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### SP-1e paired wall-clock, phase A (B1 key, no records_ou_list), 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, today's list with no region leaf)

| Set | variant | caller | pairs | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 15.10 | 23.95 | 19.34 | 26.68 | **+3.95** | **+2.73** |
| area (10 units, 220 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 10.56 | 16.90 | 17.01 | 24.46 | **+6.09** | **+7.56** |
| district (101 units, 3379 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 14.10 | 21.79 | 19.91 | 29.51 | **+5.67** | **+7.72** |
| district (101 units, 3379 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 12.25 | 18.42 | 43.43 | 59.69 | **+31.45** | **+41.27** |
| district thin (101 units, 600 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 16.54 | 24.10 | 23.32 | 33.84 | **+6.66** | **+9.75** |
| district thin (101 units, 600 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 11.04 | 20.21 | 20.86 | 31.93 | **+9.12** | **+11.72** |
| district hot (101 units, 22996 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 13.96 | 22.00 | 17.89 | 26.70 | **+3.57** | **+4.71** |
| district hot (101 units, 22996 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 9.78 | 16.66 | 17.68 | 27.97 | **+8.15** | **+11.31** |
| region (1011 units, 37930 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 13.98 | 21.78 | 18.06 | 26.90 | **+3.95** | **+5.12** |
| region (1011 units, 37930 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 11.25 | 16.15 | 20.70 | 27.19 | **+9.37** | **+11.04** |
| root (10111 units, 380048 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 13.58 | 21.53 | 16.81 | 23.88 | **+2.51** | **+2.34** |
| root (10111 units, 380048 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 9.70 | 17.13 | 15.23 | 25.75 | **+5.54** | **+8.62** |
| district mixed (101 units, 900 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 13.64 | 21.75 | 20.84 | 30.61 | **+6.69** | **+8.86** |
| district mixed (101 units, 900 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 9.88 | 16.86 | 19.85 | 29.86 | **+9.49** | **+13.00** |
| uniform m=1001 (101 units, 1001 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 14.20 | 21.82 | 20.65 | 29.30 | **+6.12** | **+7.48** |
| uniform m=1001 (101 units, 1001 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 14.43 | 34.45 | 98.59 | 196.04 | **+85.43** | **+161.59** |
| uniform m=3000 (101 units, 3000 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 19.37 | 26.73 | 28.80 | 39.78 | **+9.59** | **+13.05** |
| uniform m=3000 (101 units, 3000 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 14.29 | 18.72 | 62.16 | 82.01 | **+47.66** | **+63.29** |
| uniform m=10000 (101 units, 10000 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | agents | 300 | 18.78 | 23.64 | 25.90 | 35.45 | **+7.06** | **+11.82** |
| uniform m=10000 (101 units, 10000 tickets) | MC3: P3e, CC count, page C3 <= K / W0 > K | count-only | 300 | 15.47 | 20.75 | 35.95 | 47.60 | **+20.45** | **+26.85** |

Floor gate (phase A (B1 key, no records_ou_list), at end): `SELECT 1`, 8 clients x 200: p50 2.04, p95 3.01 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Floor gate (phase B (B1 key + records_ou_list)): `SELECT 1`, 8 clients x 200: p50 2.25, p95 3.36 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### SP-1e paired wall-clock, phase B (B1 key + records_ou_list), 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, today's list with no region leaf)

| Set | variant | caller | pairs | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 17.83 | 22.10 | 23.40 | 30.04 | **+5.40** | **+7.94** |
| area (10 units, 220 tickets) | L1 at every size (no probe), CC count | agents | 300 | 16.48 | 21.35 | 19.15 | 24.34 | **+2.49** | **+2.99** |
| area (10 units, 220 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 12.54 | 16.33 | 19.77 | 25.48 | **+7.12** | **+9.15** |
| district (101 units, 3379 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 16.12 | 21.59 | 23.92 | 29.40 | **+7.85** | **+7.81** |
| district (101 units, 3379 tickets) | L1 at every size (no probe), CC count | agents | 300 | 16.16 | 23.18 | 23.02 | 30.95 | **+6.93** | **+7.77** |
| district (101 units, 3379 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 12.56 | 16.13 | 49.04 | 61.94 | **+36.48** | **+45.81** |
| district thin (101 units, 600 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 15.89 | 20.93 | 23.32 | 29.87 | **+7.30** | **+8.93** |
| district thin (101 units, 600 tickets) | L1 at every size (no probe), CC count | agents | 300 | 16.18 | 20.54 | 20.50 | 25.07 | **+4.34** | **+4.53** |
| district thin (101 units, 600 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 12.63 | 16.68 | 23.56 | 29.68 | **+10.71** | **+13.00** |
| district hot (101 units, 22996 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 16.25 | 20.78 | 22.01 | 27.98 | **+5.90** | **+7.20** |
| district hot (101 units, 22996 tickets) | L1 at every size (no probe), CC count | agents | 300 | 15.44 | 20.31 | 21.07 | 26.66 | **+5.46** | **+6.35** |
| district hot (101 units, 22996 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 11.14 | 15.06 | 21.98 | 27.73 | **+10.87** | **+12.67** |
| region (1011 units, 37930 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 14.13 | 18.95 | 18.20 | 24.46 | **+4.02** | **+5.51** |
| region (1011 units, 37930 tickets) | L1 at every size (no probe), CC count | agents | 300 | 14.14 | 20.70 | 58.67 | 67.13 | **+44.15** | **+46.43** |
| region (1011 units, 37930 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 10.33 | 16.17 | 19.43 | 25.45 | **+8.94** | **+9.27** |
| root (10111 units, 380048 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 14.02 | 19.22 | 16.93 | 23.58 | **+2.99** | **+4.36** |

## 2026-11-21T06:56:47Z phase=e-wall secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Floor gate (phase A (B1 key, no records_ou_list)): `SELECT 1`, 8 clients x 200: p50 1.66, p95 2.30 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### SP-1e paired wall-clock, phase A (B1 key, no records_ou_list), 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, today's list with no region leaf)

| Set | variant | caller | pairs | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |

Floor gate (phase A (B1 key, no records_ou_list), at end): `SELECT 1`, 8 clients x 200: p50 1.70, p95 2.30 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Floor gate (phase B (B1 key + records_ou_list)): `SELECT 1`, 8 clients x 200: p50 1.20, p95 2.25 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### SP-1e paired wall-clock, phase B (B1 key + records_ou_list), 300 pairs per cell (BEGIN, set_config, statements, COMMIT from the client; baseline = same caller, today's list with no region leaf)

| Set | variant | caller | pairs | base p50 | base p95 | var p50 | var p95 | **median per-pair Δ** | **Δp95** |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| root (10111 units, 380048 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 11.55 | 16.35 | 19.48 | 25.40 | **+7.75** | **+9.06** |
| district mixed (101 units, 900 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 16.23 | 22.55 | 23.82 | 30.93 | **+7.81** | **+8.38** |
| district mixed (101 units, 900 tickets) | L1 at every size (no probe), CC count | agents | 300 | 16.49 | 20.67 | 21.07 | 26.86 | **+4.75** | **+6.19** |
| district mixed (101 units, 900 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 13.44 | 17.11 | 25.70 | 34.02 | **+12.39** | **+16.91** |
| uniform m=1001 (101 units, 1001 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 15.98 | 24.47 | 25.16 | 33.69 | **+9.20** | **+9.22** |
| uniform m=1001 (101 units, 1001 tickets) | L1 at every size (no probe), CC count | agents | 300 | 15.08 | 20.39 | 19.46 | 25.66 | **+4.57** | **+5.27** |
| uniform m=1001 (101 units, 1001 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 11.33 | 16.27 | 79.23 | 93.12 | **+67.57** | **+76.84** |
| uniform m=3000 (101 units, 3000 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 14.45 | 19.88 | 21.51 | 27.01 | **+6.76** | **+7.13** |
| uniform m=3000 (101 units, 3000 tickets) | L1 at every size (no probe), CC count | agents | 300 | 13.91 | 18.76 | 22.67 | 27.03 | **+8.48** | **+8.27** |
| uniform m=3000 (101 units, 3000 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 10.74 | 16.22 | 45.38 | 52.07 | **+34.22** | **+35.84** |
| uniform m=10000 (101 units, 10000 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | agents | 300 | 14.84 | 20.58 | 20.23 | 26.63 | **+5.42** | **+6.05** |
| uniform m=10000 (101 units, 10000 tickets) | L1 at every size (no probe), CC count | agents | 300 | 14.23 | 20.26 | 26.98 | 32.48 | **+12.41** | **+12.22** |
| uniform m=10000 (101 units, 10000 tickets) | ML1: P3e, CC count, page L1 <= K / W0 > K | count-only | 300 | 11.26 | 16.19 | 26.10 | 31.02 | **+14.51** | **+14.82** |

Floor gate (phase B (B1 key + records_ou_list), at end): `SELECT 1`, 8 clients x 200: p50 1.13, p95 1.78 ms -> **PASS**. Foreign containers: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T07:00:03Z phase=e-write secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### L1 write cost on records (ABA; replica role: index maintenance only)

| Pass | records_ou_list | single-row INSERT n | p50 ms | p95 ms | WAL bytes / row | 50,000-row INSERT ms | WAL bytes / row (bulk) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | false | 2000 | 5.312 | 6.759 | 9447 | 5145 | 10765 |
| 2 | true | 2000 | 6.241 | 21.269 | 8660 | 7698 | 10015 |
| 3 | false | 2000 | 6.933 | 32.026 | 9519 | 11556 | 9013 |

records_ou_list size: 62 MB (records_org_unit on the B1 key: 6 MB).
