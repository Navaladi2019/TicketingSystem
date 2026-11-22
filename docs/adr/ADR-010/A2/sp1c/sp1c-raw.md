
## 2026-11-21T04:50:48Z phase=setup secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

### Closure build (live units only)

| Run | INSERT ms | rows |
| --- | --- | --- |
| 1 | 234.4 | 49321 |
| 2 | 264.8 | 49321 |
| 3 | 254.9 | 49321 |

## 2026-11-21T04:51:18Z phase=explain secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### Plans: region branch of the page statement, agents[0]

| Set | leaf | page plan / exec ms | region branch ms | region branch plan | records read / removed | closure rows |
| --- | --- | --- | --- | --- | --- | --- |
| baseline | none | 5.5 / 6.8 | n/a | n/a | n/a | n/a |
| area (10 units, 220 tickets) | C1 | 4.3 / 6.0 | 2.93 | Limit > Sort > Result > Nested Loop > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey | 220 / 10 | 10 |
| area (10 units, 220 tickets) | W0 | 2.5 / 45.8 | 43.65 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 6396 / 6395 | 0 |
| area (10 units, 220 tickets) | W1 | 1.9 / 3.0 | 1.09 | Limit > Sort > Result > Nested Loop > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey | 220 / 10 | 10 |
| area (10 units, 220 tickets) | W0, no AND NOT | 1.9 / 26.9 | 24.93 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 6396 / 6395 | 0 |
| area (10 units, 220 tickets) | C1, no AND NOT | 2.5 / 4.3 | 0.85 | Limit > Sort > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Scan records_org_unit | 220 / 0 | 10 |
| area (10 units, 220 tickets) | W0 count | 2.4 / 1.5 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| district (101 units, 3379 tickets) | C1 | 3.6 / 9.3 | 6.38 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 949 / 96 | 0 |
| district (101 units, 3379 tickets) | W0 | 1.5 / 9.5 | 7.09 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 949 / 942 | 0 |
| district (101 units, 3379 tickets) | W1 | 2.5 / 5.8 | 4.06 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 949 / 96 | 0 |
| district (101 units, 3379 tickets) | W0, no AND NOT | 2.2 / 5.8 | 4.29 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 949 / 942 | 0 |
| district (101 units, 3379 tickets) | C1, no AND NOT | 3.1 / 5.0 | 3.54 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 949 / 41 | 0 |
| district (101 units, 3379 tickets) | W0 count | 2.3 / 2.4 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| district thin (101 units, 600 tickets) | C1 | 2.1 / 6.2 | 4.11 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 913 / 92 | 0 |
| district thin (101 units, 600 tickets) | W0 | 1.4 / 8.3 | 5.77 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 913 / 912 | 0 |
| district thin (101 units, 600 tickets) | W1 | 3.0 / 5.9 | 3.89 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 913 / 92 | 0 |
| district thin (101 units, 600 tickets) | W0, no AND NOT | 3.0 / 6.7 | 4.03 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 913 / 912 | 0 |
| district thin (101 units, 600 tickets) | C1, no AND NOT | 3.2 / 8.8 | 5.80 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 913 / 39 | 0 |
| district thin (101 units, 600 tickets) | W0 count | 1.4 / 1.0 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| district hot (101 units, 22996 tickets) | C1 | 1.5 / 4.8 | 2.74 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 448 / 45 | 0 |
| district hot (101 units, 22996 tickets) | W0 | 3.3 / 3.1 | 1.93 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 448 / 422 | 0 |
| district hot (101 units, 22996 tickets) | W1 | 2.0 / 4.1 | 2.48 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 448 / 45 | 0 |
| district hot (101 units, 22996 tickets) | W0, no AND NOT | 1.8 / 3.7 | 2.13 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 416 / 389 | 0 |
| district hot (101 units, 22996 tickets) | C1, no AND NOT | 2.1 / 3.2 | 1.69 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 416 / 18 | 0 |
| district hot (101 units, 22996 tickets) | W0 count | 2.4 / 1.6 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| region (1011 units, 37930 tickets) | C1 | 1.6 / 2.3 | 1.14 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 329 / 33 | 0 |
| region (1011 units, 37930 tickets) | W0 | 1.5 / 2.4 | 1.27 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 329 / 298 | 0 |
| region (1011 units, 37930 tickets) | W1 | 1.7 / 2.4 | 1.23 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 329 / 33 | 0 |
| region (1011 units, 37930 tickets) | W0, no AND NOT | 1.6 / 2.7 | 1.47 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 329 / 298 | 0 |
| region (1011 units, 37930 tickets) | C1, no AND NOT | 1.4 / 2.2 | 1.10 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 329 / 15 | 0 |
| region (1011 units, 37930 tickets) | W0 count | 1.6 / 1.6 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| root (10111 units, 380048 tickets) | C1 | 1.7 / 1.8 | 0.84 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 52 / 2 | 50 |
| root (10111 units, 380048 tickets) | W0 | 1.7 / 1.5 | 0.55 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 52 / 2 | 50 |
| root (10111 units, 380048 tickets) | W1 | 2.0 / 1.8 | 0.67 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 52 / 2 | 50 |
| root (10111 units, 380048 tickets) | W0, no AND NOT | 1.6 / 1.2 | 0.34 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 51 / 1 | 50 |
| root (10111 units, 380048 tickets) | C1, no AND NOT | 2.6 / 2.0 | 0.53 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 51 / 1 | 50 |
| root (10111 units, 380048 tickets) | W0 count | 1.4 / 1.4 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |

### Probe plans (records counted = min(all, K+1))

| Set | K | records under root (any type) | probe plan / exec ms | plan | records read |
| --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | 250 | 220 | 0.36 / 0.34 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 220 |
| area (10 units, 220 tickets) | 1000 | 220 | 0.60 / 0.59 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 220 |
| area (10 units, 220 tickets) | 4000 | 220 | 0.42 / 0.35 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 220 |
| district (101 units, 3379 tickets) | 250 | 3379 | 0.36 / 0.34 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 252 |
| district (101 units, 3379 tickets) | 1000 | 3379 | 0.38 / 1.31 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 986 |
| district (101 units, 3379 tickets) | 4000 | 3379 | 0.42 / 2.98 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 3333 |
| district thin (101 units, 600 tickets) | 250 | 600 | 0.37 / 0.55 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 245 |
| district thin (101 units, 600 tickets) | 1000 | 600 | 0.41 / 0.96 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 606 |
| district thin (101 units, 600 tickets) | 4000 | 600 | 0.38 / 0.94 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 606 |
| district hot (101 units, 22996 tickets) | 250 | 22996 | 0.54 / 0.62 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 252 |
| district hot (101 units, 22996 tickets) | 1000 | 22996 | 0.45 / 1.24 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 984 |
| district hot (101 units, 22996 tickets) | 4000 | 22996 | 0.62 / 4.16 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 4004 |
| region (1011 units, 37930 tickets) | 250 | 37930 | 0.43 / 0.76 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 252 |
| region (1011 units, 37930 tickets) | 1000 | 37930 | 0.36 / 3.06 | Aggregate > Limit > Result > Merge Join > Index Only Scan records_org_unit > Sort > Index Only Scan org_unit_closure_pkey | 12028 |
| region (1011 units, 37930 tickets) | 4000 | 37930 | 0.37 / 12.56 | Aggregate > Limit > Result > Merge Join > Index Only Scan records_org_unit > Sort > Index Only Scan org_unit_closure_pkey | 44656 |
| root (10111 units, 380048 tickets) | 250 | 380048 | 0.39 / 0.86 | Aggregate > Limit > Result > Nested Loop > Index Only Scan records_org_unit > Index Only Scan org_unit_closure_pkey | 251 |
| root (10111 units, 380048 tickets) | 1000 | 380048 | 0.49 / 7.00 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 1014 |
| root (10111 units, 380048 tickets) | 4000 | 380048 | 0.38 / 7.12 | Aggregate > Limit > Result > Hash Join > Index Only Scan records_org_unit > Hash > Index Only Scan org_unit_closure_pkey | 4001 |

### Plans

#### baseline

```
Nested Loop Left Join  (cost=46.63..1449.11 rows=1 width=1301) (actual time=0.371..2.196 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.33..1440.78 rows=1 width=1266) (actual time=0.351..1.975 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=46.05..1432.46 rows=1 width=1259) (actual time=0.338..1.778 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=46.05..1431.32 rows=51 width=1259) (actual time=0.334..1.706 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=46.05..3169.69 rows=115 width=1259) (actual time=0.333..1.694 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.125..1.456 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.124..1.447 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.117..1.412 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.024 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.019..0.023 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.016..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.016..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.168..0.174 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.168..0.173 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.135..0.163 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.133..0.159 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.054..0.058 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.041..0.049 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.650 ms
Execution Time: 2.420 ms
```

#### area (10 units, 220 tickets) C1

```
Nested Loop Left Join  (cost=631.22..1902.98 rows=1 width=1301) (actual time=2.892..5.887 rows=51 loops=1)
  Buffers: shared hit=1633
  ->  Nested Loop  (cost=630.92..1894.65 rows=1 width=1266) (actual time=2.843..5.433 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1480
        ->  Subquery Scan on x  (cost=630.64..1886.33 rows=1 width=1259) (actual time=2.814..5.109 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1327
              ->  Limit  (cost=630.64..1885.19 rows=51 width=1259) (actual time=2.809..4.990 rows=51 loops=1)
                    Buffers: shared hit=1327
                    ->  Merge Append  (cost=630.64..3754.70 rows=127 width=1259) (actual time=2.807..4.971 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1327
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.134..2.247 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.134..2.232 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.125..2.185 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.026..0.035 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.026..0.034 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.022..0.028 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.125..0.132 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.124..0.130 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.087..0.118 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.084..0.113 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.058..0.063 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.043..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.034..0.036 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=584.58..584.61 rows=12 width=1259) (actual time=2.490..2.495 rows=1 loops=1)
                                Buffers: shared hit=494
                                ->  Sort  (cost=584.58..584.61 rows=12 width=1259) (actual time=2.488..2.493 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 205kB
                                      Buffers: shared hit=494
                                      ->  Result  (cost=5.20..584.36 rows=12 width=1259) (actual time=0.099..2.126 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=494
                                            ->  Nested Loop Anti Join  (cost=5.20..584.36 rows=12 width=1251) (actual time=0.092..2.023 rows=210 loops=1)
                                                  Buffers: shared hit=494
                                                  ->  Nested Loop  (cost=4.91..580.30 rows=12 width=1251) (actual time=0.072..0.565 rows=210 loops=1)
                                                        Buffers: shared hit=73
                                                        ->  HashAggregate  (cost=4.48..4.51 rows=3 width=16) (actual time=0.025..0.033 rows=10 loops=1)
                                                              Group Key: acl_ou.descendant_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.017..0.019 rows=10 loops=1)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                        ->  Index Scan using records_org_unit on records r_4  (cost=0.42..191.81 rows=12 width=1267) (actual time=0.009..0.040 rows=21 loops=10)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-70… (559 chars)
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=69
                                                  ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=210)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        Buffers: shared hit=421
                                                        ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=210)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Rows Removed by Filter: 0
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=421
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.486 ms
Execution Time: 6.346 ms
```

#### area (10 units, 220 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=27.170..28.578 rows=51 loops=1)
  Buffers: shared hit=24638
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=27.149..28.418 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=24485
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=27.137..28.303 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=24332
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=27.133..28.262 rows=51 loops=1)
                    Buffers: shared hit=24332
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=27.132..28.254 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=24332
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.123..1.216 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.122..1.209 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.117..1.186 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.020..0.026 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.020..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.017..0.020 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.082..0.089 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.082..0.088 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.055..0.079 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.053..0.075 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.043 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.036 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=26.884..26.887 rows=1 loops=1)
                                Buffers: shared hit=23499
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=26.882..26.885 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=23499
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=26.880..26.882 rows=1 loops=1)
                                            Buffers: shared hit=23499
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=26.860..26.861 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 6395
                                                  Buffers: shared hit=23497
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=5679)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=17038
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=5679)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=17038
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.017..0.017 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.879 ms
Execution Time: 28.796 ms
```

#### area (10 units, 220 tickets) W1

```
Nested Loop Left Join  (cost=631.22..1902.98 rows=1 width=1301) (actual time=2.181..3.705 rows=51 loops=1)
  Buffers: shared hit=1633
  ->  Nested Loop  (cost=630.92..1894.65 rows=1 width=1266) (actual time=2.159..3.530 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1480
        ->  Subquery Scan on x  (cost=630.64..1886.33 rows=1 width=1259) (actual time=2.144..3.398 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1327
              ->  Limit  (cost=630.64..1885.19 rows=51 width=1259) (actual time=2.141..3.348 rows=51 loops=1)
                    Buffers: shared hit=1327
                    ->  Merge Append  (cost=630.64..3754.70 rows=127 width=1259) (actual time=2.139..3.338 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1327
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.131..1.302 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.131..1.296 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.123..1.269 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.027..0.032 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.026..0.031 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.023..0.027 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.024..0.026 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.021..0.021 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.115..0.120 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.115..0.119 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.077..0.108 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.075..0.104 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.059 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.048 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=584.58..584.61 rows=12 width=1259) (actual time=1.836..1.840 rows=1 loops=1)
                                Buffers: shared hit=494
                                ->  Sort  (cost=584.58..584.61 rows=12 width=1259) (actual time=1.835..1.838 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 205kB
                                      Buffers: shared hit=494
                                      ->  Result  (cost=5.20..584.36 rows=12 width=1259) (actual time=0.089..1.590 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=494
                                            ->  Nested Loop Anti Join  (cost=5.20..584.36 rows=12 width=1251) (actual time=0.087..1.507 rows=210 loops=1)
                                                  Buffers: shared hit=494
                                                  ->  Nested Loop  (cost=4.91..580.30 rows=12 width=1251) (actual time=0.070..0.414 rows=210 loops=1)
                                                        Buffers: shared hit=73
                                                        ->  HashAggregate  (cost=4.48..4.51 rows=3 width=16) (actual time=0.023..0.029 rows=10 loops=1)
                                                              Group Key: acl_ou.descendant_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.016..0.018 rows=10 loops=1)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                        ->  Index Scan using records_org_unit on records r_4  (cost=0.42..191.81 rows=12 width=1267) (actual time=0.007..0.028 rows=21 loops=10)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-70… (559 chars)
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=69
                                                  ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=210)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        Buffers: shared hit=421
                                                        ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=210)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Rows Removed by Filter: 0
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=421
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.799 ms
Execution Time: 3.971 ms
```

#### area (10 units, 220 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.974..1.983 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.043..1.801 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.041..1.609 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.041..1.329 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.040..1.163 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.039..0.895 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.039..0.741 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.029..0.505 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.171 ms
Execution Time: 2.145 ms
```

#### district (101 units, 3379 tickets) C1

```
Nested Loop Left Join  (cost=47.78..16297.37 rows=1 width=1301) (actual time=1.070..6.845 rows=51 loops=1)
  Buffers: shared hit=4624
  ->  Nested Loop  (cost=47.49..16289.03 rows=1 width=1266) (actual time=1.038..6.495 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4471
        ->  Subquery Scan on x  (cost=47.20..16280.72 rows=1 width=1259) (actual time=1.017..6.258 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4318
              ->  Limit  (cost=47.20..16279.57 rows=51 width=1259) (actual time=1.012..6.181 rows=51 loops=1)
                    Buffers: shared hit=4318
                    ->  Merge Append  (cost=47.20..52881.97 rows=166 width=1259) (actual time=1.010..6.166 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4318
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.165..0.995 rows=44 loops=1)
                                Buffers: shared hit=747
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.164..0.985 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=747
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.154..0.950 rows=44 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=747
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.032..0.039 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.031..0.038 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.025..0.029 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.028..0.029 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.028..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.025..0.025 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.198..0.202 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.197..0.202 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.093..0.184 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.090..0.178 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.068..0.072 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.049..0.059 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.038..0.039 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.032..0.032 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..49711.22 rows=51 width=1259) (actual time=0.581..4.869 rows=7 loops=1)
                                Buffers: shared hit=3535
                                ->  Result  (cost=1.14..393783.00 rows=404 width=1259) (actual time=0.580..4.864 rows=7 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3535
                                      ->  Nested Loop Anti Join  (cost=1.14..393783.00 rows=404 width=1251) (actual time=0.574..4.853 rows=7 loops=1)
                                            Buffers: shared hit=3535
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.77 rows=404 width=1251) (actual time=0.548..4.752 rows=7 loops=1)
                                                  Buffers: shared hit=3521
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.014..2.078 rows=853 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 96
                                                        Buffers: shared hit=961
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=853)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2560
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=7)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=14
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=7)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=14
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.540 ms
Execution Time: 7.186 ms
```

#### district (101 units, 3379 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=1.105..9.607 rows=51 loops=1)
  Buffers: shared hit=4624
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=1.066..9.239 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4471
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=1.047..8.976 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4318
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=1.043..8.886 rows=51 loops=1)
                    Buffers: shared hit=4318
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=1.041..8.867 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4318
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.164..1.122 rows=44 loops=1)
                                Buffers: shared hit=747
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.163..1.108 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=747
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.153..1.067 rows=44 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=747
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.032..0.039 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.031..0.037 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.026..0.030 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.029..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.025..0.026 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.207..0.212 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.206..0.211 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.106..0.194 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.102..0.187 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.077..0.081 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.058..0.068 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.045..0.046 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.023..0.023 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.032..0.032 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.602..7.427 rows=7 loops=1)
                                Buffers: shared hit=3535
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.601..7.422 rows=7 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3535
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.595..7.408 rows=7 loops=1)
                                            Buffers: shared hit=3535
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.579..7.327 rows=7 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 942
                                                  Buffers: shared hit=3521
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.005..0.005 rows=0 loops=853)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=2560
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=853)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=2560
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=7)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=14
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=7)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=14
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.397 ms
Execution Time: 9.963 ms
```

#### district (101 units, 3379 tickets) W1

```
Nested Loop Left Join  (cost=47.78..16297.37 rows=1 width=1301) (actual time=0.977..8.438 rows=51 loops=1)
  Buffers: shared hit=4624
  ->  Nested Loop  (cost=47.49..16289.03 rows=1 width=1266) (actual time=0.943..8.068 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4471
        ->  Subquery Scan on x  (cost=47.20..16280.72 rows=1 width=1259) (actual time=0.924..7.760 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4318
              ->  Limit  (cost=47.20..16279.57 rows=51 width=1259) (actual time=0.920..7.664 rows=51 loops=1)
                    Buffers: shared hit=4318
                    ->  Merge Append  (cost=47.20..52881.97 rows=166 width=1259) (actual time=0.918..7.644 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4318
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.151..1.117 rows=44 loops=1)
                                Buffers: shared hit=747
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.151..1.104 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=747
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.144..1.065 rows=44 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=747
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.028..0.036 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.028..0.034 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.024..0.028 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.028 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.024..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.138..0.150 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.138..0.149 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.092..0.130 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.089..0.125 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.067..0.073 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.050..0.061 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.039..0.041 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..49711.22 rows=51 width=1259) (actual time=0.568..6.275 rows=7 loops=1)
                                Buffers: shared hit=3535
                                ->  Result  (cost=1.14..393783.00 rows=404 width=1259) (actual time=0.567..6.271 rows=7 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3535
                                      ->  Nested Loop Anti Join  (cost=1.14..393783.00 rows=404 width=1251) (actual time=0.563..6.260 rows=7 loops=1)
                                            Buffers: shared hit=3535
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.77 rows=404 width=1251) (actual time=0.543..6.158 rows=7 loops=1)
                                                  Buffers: shared hit=3521
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.012..2.623 rows=853 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 96
                                                        Buffers: shared hit=961
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.003..0.003 rows=0 loops=853)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2560
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=7)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=14
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=7)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=14
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.349 ms
Execution Time: 8.777 ms
```

#### district (101 units, 3379 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=0.869..0.873 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.036..0.797 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.035..0.710 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.035..0.604 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.034..0.528 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.033..0.440 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.033..0.361 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.025..0.250 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.188 ms
Execution Time: 1.020 ms
```

#### district thin (101 units, 600 tickets) C1

```
Nested Loop Left Join  (cost=47.78..16450.04 rows=1 width=1301) (actual time=5.748..7.693 rows=51 loops=1)
  Buffers: shared hit=4529
  ->  Nested Loop  (cost=47.49..16441.70 rows=1 width=1266) (actual time=5.720..7.395 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4376
        ->  Subquery Scan on x  (cost=47.20..16433.39 rows=1 width=1259) (actual time=5.694..7.161 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4223
              ->  Limit  (cost=47.20..16432.24 rows=51 width=1259) (actual time=5.690..7.077 rows=51 loops=1)
                    Buffers: shared hit=4223
                    ->  Merge Append  (cost=47.20..53378.89 rows=166 width=1259) (actual time=5.688..7.060 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4223
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.145..1.476 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.144..1.464 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.136..1.424 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.035..0.042 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.034..0.040 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.029..0.033 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.028..0.030 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.023..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.125..0.132 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.124..0.130 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.084..0.117 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.081..0.112 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.059..0.064 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.044..0.053 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.034..0.035 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..50208.15 rows=51 width=1259) (actual time=5.347..5.351 rows=1 loops=1)
                                Buffers: shared hit=3390
                                ->  Result  (cost=1.14..393781.61 rows=400 width=1259) (actual time=5.346..5.349 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3390
                                      ->  Nested Loop Anti Join  (cost=1.14..393781.61 rows=400 width=1251) (actual time=5.343..5.345 rows=1 loops=1)
                                            Buffers: shared hit=3390
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.73 rows=400 width=1251) (actual time=5.314..5.315 rows=1 loops=1)
                                                  Buffers: shared hit=3388
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.010..2.389 rows=821 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 92
                                                        Buffers: shared hit=924
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.003..0.003 rows=0 loops=821)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2464
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.023..0.024 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.014..0.014 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.185 ms
Execution Time: 8.027 ms
```

#### district thin (101 units, 600 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=4.970..7.366 rows=51 loops=1)
  Buffers: shared hit=4529
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=4.940..6.998 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4376
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=4.912..6.731 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4223
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=4.908..6.627 rows=51 loops=1)
                    Buffers: shared hit=4223
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=4.906..6.609 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4223
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.213..1.864 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.212..1.850 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.203..1.806 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.044..0.053 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.042..0.050 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.035..0.040 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.031..0.033 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.030..0.031 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.025..0.025 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.157..0.166 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.157..0.164 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.109..0.147 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.105..0.140 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.078..0.083 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.056..0.068 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.039..0.041 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=4.453..4.459 rows=1 loops=1)
                                Buffers: shared hit=3390
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=4.452..4.457 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3390
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=4.445..4.449 rows=1 loops=1)
                                            Buffers: shared hit=3390
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=4.422..4.424 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 912
                                                  Buffers: shared hit=3388
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=821)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=2464
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=821)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=2464
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.018..0.019 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.014..0.015 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.883 ms
Execution Time: 7.770 ms
```

#### district thin (101 units, 600 tickets) W1

```
Nested Loop Left Join  (cost=47.78..16450.04 rows=1 width=1301) (actual time=5.192..7.555 rows=51 loops=1)
  Buffers: shared hit=4529
  ->  Nested Loop  (cost=47.49..16441.70 rows=1 width=1266) (actual time=5.153..7.200 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4376
        ->  Subquery Scan on x  (cost=47.20..16433.39 rows=1 width=1259) (actual time=5.133..6.921 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4223
              ->  Limit  (cost=47.20..16432.24 rows=51 width=1259) (actual time=5.127..6.814 rows=51 loops=1)
                    Buffers: shared hit=4223
                    ->  Merge Append  (cost=47.20..53378.89 rows=166 width=1259) (actual time=5.125..6.763 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4223
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.125..1.713 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.125..1.699 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.117..1.649 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.030..0.037 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.029..0.035 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.025..0.029 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.024..0.026 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.117..0.125 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.117..0.124 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.080..0.112 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.078..0.108 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.059..0.065 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.044..0.054 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.034..0.036 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.015 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..50208.15 rows=51 width=1259) (actual time=4.820..4.825 rows=1 loops=1)
                                Buffers: shared hit=3390
                                ->  Result  (cost=1.14..393781.61 rows=400 width=1259) (actual time=4.819..4.824 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3390
                                      ->  Nested Loop Anti Join  (cost=1.14..393781.61 rows=400 width=1251) (actual time=4.815..4.819 rows=1 loops=1)
                                            Buffers: shared hit=3390
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.73 rows=400 width=1251) (actual time=4.778..4.780 rows=1 loops=1)
                                                  Buffers: shared hit=3388
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.009..2.192 rows=821 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 92
                                                        Buffers: shared hit=924
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=821)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2464
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.031..0.033 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.016..0.016 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.147 ms
Execution Time: 7.851 ms
```

#### district thin (101 units, 600 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=0.873..0.876 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.062..0.798 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.060..0.709 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.059..0.600 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.058..0.523 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.057..0.438 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.056..0.368 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.044..0.260 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 2.274 ms
Execution Time: 1.116 ms
```

#### district hot (101 units, 22996 tickets) C1

```
Nested Loop Left Join  (cost=47.78..15374.25 rows=1 width=1301) (actual time=0.521..4.614 rows=51 loops=1)
  Buffers: shared hit=2487
  ->  Nested Loop  (cost=47.49..15365.91 rows=1 width=1266) (actual time=0.495..4.209 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2334
        ->  Subquery Scan on x  (cost=47.20..15357.60 rows=1 width=1259) (actual time=0.478..3.978 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2181
              ->  Limit  (cost=47.20..15356.45 rows=51 width=1259) (actual time=0.475..3.901 rows=51 loops=1)
                    Buffers: shared hit=2181
                    ->  Merge Append  (cost=47.20..49877.31 rows=166 width=1259) (actual time=0.473..3.884 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2181
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.170..0.787 rows=25 loops=1)
                                Buffers: shared hit=428
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.169..0.780 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=428
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.160..0.755 rows=25 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=428
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.034..0.042 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.034..0.040 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.028..0.033 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.030..0.031 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.025..0.026 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.151..0.160 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.151..0.159 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.102..0.143 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.099..0.137 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.066..0.072 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.049..0.059 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.037..0.039 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.018..0.018 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..46706.57 rows=51 width=1259) (actual time=0.082..2.830 rows=26 loops=1)
                                Buffers: shared hit=1717
                                ->  Result  (cost=1.14..393792.02 rows=430 width=1259) (actual time=0.081..2.820 rows=26 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1717
                                      ->  Nested Loop Anti Join  (cost=1.14..393792.02 rows=430 width=1251) (actual time=0.077..2.799 rows=26 loops=1)
                                            Buffers: shared hit=1717
                                            ->  Nested Loop Semi Join  (cost=0.84..393647.03 rows=430 width=1251) (actual time=0.062..2.592 rows=26 loops=1)
                                                  Buffers: shared hit=1665
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.013..0.969 rows=403 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 45
                                                        Buffers: shared hit=455
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.003..0.003 rows=0 loops=403)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1210
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=26)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=52
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=26)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=52
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.631 ms
Execution Time: 4.993 ms
```

#### district hot (101 units, 22996 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=0.548..4.844 rows=51 loops=1)
  Buffers: shared hit=2487
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=0.518..4.506 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2334
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=0.493..4.251 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2181
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=0.490..4.182 rows=51 loops=1)
                    Buffers: shared hit=2181
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=0.488..4.164 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2181
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.162..0.776 rows=25 loops=1)
                                Buffers: shared hit=428
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.161..0.769 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=428
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.151..0.744 rows=25 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=428
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.034..0.038 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.033..0.036 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.027..0.030 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.028..0.029 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.028 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.024..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.153..0.158 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.153..0.157 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.105..0.140 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.095..0.128 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.070..0.075 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.052..0.062 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.039..0.040 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.106..3.132 rows=26 loops=1)
                                Buffers: shared hit=1717
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.105..3.123 rows=26 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1717
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.101..3.104 rows=26 loops=1)
                                            Buffers: shared hit=1717
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.086..2.910 rows=26 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 422
                                                  Buffers: shared hit=1665
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.004..0.004 rows=0 loops=403)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=1210
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=403)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=1210
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=26)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=52
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=26)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=52
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.486 ms
Execution Time: 5.165 ms
```

#### district hot (101 units, 22996 tickets) W1

```
Nested Loop Left Join  (cost=47.78..15374.25 rows=1 width=1301) (actual time=0.291..4.612 rows=51 loops=1)
  Buffers: shared hit=2487
  ->  Nested Loop  (cost=47.49..15365.91 rows=1 width=1266) (actual time=0.277..4.236 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2334
        ->  Subquery Scan on x  (cost=47.20..15357.60 rows=1 width=1259) (actual time=0.266..3.937 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2181
              ->  Limit  (cost=47.20..15356.45 rows=51 width=1259) (actual time=0.264..3.831 rows=51 loops=1)
                    Buffers: shared hit=2181
                    ->  Merge Append  (cost=47.20..49877.31 rows=166 width=1259) (actual time=0.263..3.813 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2181
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.091..0.770 rows=25 loops=1)
                                Buffers: shared hit=428
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.091..0.762 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=428
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.088..0.736 rows=25 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=428
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.025 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.017..0.023 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.013..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.013..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.013..0.014 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.012..0.012 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.088 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.081..0.087 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.049..0.078 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.048..0.075 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.036..0.040 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.027..0.032 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..46706.57 rows=51 width=1259) (actual time=0.056..2.872 rows=26 loops=1)
                                Buffers: shared hit=1717
                                ->  Result  (cost=1.14..393792.02 rows=430 width=1259) (actual time=0.055..2.862 rows=26 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1717
                                      ->  Nested Loop Anti Join  (cost=1.14..393792.02 rows=430 width=1251) (actual time=0.054..2.844 rows=26 loops=1)
                                            Buffers: shared hit=1717
                                            ->  Nested Loop Semi Join  (cost=0.84..393647.03 rows=430 width=1251) (actual time=0.030..2.549 rows=26 loops=1)
                                                  Buffers: shared hit=1665
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.007..1.001 rows=403 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 45
                                                        Buffers: shared hit=455
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.003..0.003 rows=0 loops=403)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1210
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=26)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=52
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=26)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=52
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.407 ms
Execution Time: 4.830 ms
```

#### district hot (101 units, 22996 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.156..1.163 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.044..1.059 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.042..0.948 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.042..0.807 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.041..0.705 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.040..0.585 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.040..0.482 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.032..0.353 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.631 ms
Execution Time: 1.329 ms
```

#### region (1011 units, 37930 tickets) C1

```
Nested Loop Left Join  (cost=47.78..2585.22 rows=1 width=1301) (actual time=0.329..2.098 rows=51 loops=1)
  Buffers: shared hit=1977
  ->  Nested Loop  (cost=47.49..2576.88 rows=1 width=1266) (actual time=0.310..1.903 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1824
        ->  Subquery Scan on x  (cost=47.20..2568.57 rows=1 width=1259) (actual time=0.297..1.792 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1671
              ->  Limit  (cost=47.20..2567.42 rows=51 width=1259) (actual time=0.295..1.754 rows=51 loops=1)
                    Buffers: shared hit=1671
                    ->  Merge Append  (cost=47.20..8250.27 rows=166 width=1259) (actual time=0.294..1.746 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1671
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.095..0.445 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.095..0.442 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.091..0.431 rows=21 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.016..0.016 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.014..0.014 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.021..0.022 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.021..0.021 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.103..0.106 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.102..0.105 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.062..0.096 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.091 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.045 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.037 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..5079.52 rows=51 width=1259) (actual time=0.056..1.138 rows=31 loops=1)
                                Buffers: shared hit=1286
                                ->  Result  (cost=1.14..395019.94 rows=3967 width=1259) (actual time=0.055..1.133 rows=31 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1286
                                      ->  Nested Loop Anti Join  (cost=1.14..395019.94 rows=3967 width=1251) (actual time=0.051..1.120 rows=31 loops=1)
                                            Buffers: shared hit=1286
                                            ->  Nested Loop Semi Join  (cost=0.84..393682.40 rows=3967 width=1251) (actual time=0.042..1.000 rows=31 loops=1)
                                                  Buffers: shared hit=1224
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.014..0.405 rows=296 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=335
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=296)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=889
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=31)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=62
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=31)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=62
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.586 ms
Execution Time: 2.266 ms
```

#### region (1011 units, 37930 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=0.322..2.318 rows=51 loops=1)
  Buffers: shared hit=1977
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=0.308..2.135 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1824
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=0.297..1.992 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1671
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=0.296..1.953 rows=51 loops=1)
                    Buffers: shared hit=1671
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=0.294..1.944 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1671
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.114..0.486 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.113..0.483 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.106..0.469 rows=21 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.020..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.020..0.021 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.015..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.093..0.096 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.093..0.096 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.065..0.087 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.063..0.084 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.050 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.040 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.047..1.304 rows=31 loops=1)
                                Buffers: shared hit=1286
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.046..1.299 rows=31 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1286
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.044..1.287 rows=31 loops=1)
                                            Buffers: shared hit=1286
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.035..1.163 rows=31 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 298
                                                  Buffers: shared hit=1224
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=296)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=889
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=296)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=889
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=31)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=62
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=31)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=62
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.624 ms
Execution Time: 2.513 ms
```

#### region (1011 units, 37930 tickets) W1

```
Nested Loop Left Join  (cost=47.78..2585.22 rows=1 width=1301) (actual time=0.283..2.138 rows=51 loops=1)
  Buffers: shared hit=1977
  ->  Nested Loop  (cost=47.49..2576.88 rows=1 width=1266) (actual time=0.268..1.948 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1824
        ->  Subquery Scan on x  (cost=47.20..2568.57 rows=1 width=1259) (actual time=0.257..1.833 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1671
              ->  Limit  (cost=47.20..2567.42 rows=51 width=1259) (actual time=0.255..1.791 rows=51 loops=1)
                    Buffers: shared hit=1671
                    ->  Merge Append  (cost=47.20..8250.27 rows=166 width=1259) (actual time=0.254..1.783 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1671
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.096..0.491 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.096..0.488 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.090..0.475 rows=21 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.016..0.016 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.014..0.014 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.015..0.016 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.015..0.016 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.014..0.014 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.077..0.081 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.077..0.080 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.052..0.073 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.050..0.071 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.041 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.028..0.034 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..5079.52 rows=51 width=1259) (actual time=0.047..1.162 rows=31 loops=1)
                                Buffers: shared hit=1286
                                ->  Result  (cost=1.14..395019.94 rows=3967 width=1259) (actual time=0.047..1.157 rows=31 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1286
                                      ->  Nested Loop Anti Join  (cost=1.14..395019.94 rows=3967 width=1251) (actual time=0.045..1.146 rows=31 loops=1)
                                            Buffers: shared hit=1286
                                            ->  Nested Loop Semi Join  (cost=0.84..393682.40 rows=3967 width=1251) (actual time=0.031..1.017 rows=31 loops=1)
                                                  Buffers: shared hit=1224
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.006..0.378 rows=296 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=335
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=296)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=889
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=31)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=62
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=31)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=62
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.681 ms
Execution Time: 2.317 ms
```

#### region (1011 units, 37930 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.105..1.112 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.043..1.010 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.042..0.893 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.041..0.756 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.040..0.657 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.040..0.541 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.039..0.451 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.031..0.320 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.291 ms
Execution Time: 1.256 ms
```

#### root (10111 units, 380048 tickets) C1

```
Nested Loop Left Join  (cost=47.78..1189.61 rows=1 width=1301) (actual time=0.300..1.190 rows=51 loops=1)
  Buffers: shared hit=713
  ->  Nested Loop  (cost=47.49..1181.27 rows=1 width=1266) (actual time=0.283..0.991 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=560
        ->  Subquery Scan on x  (cost=47.20..1172.96 rows=1 width=1259) (actual time=0.273..0.840 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=407
              ->  Limit  (cost=47.20..1171.81 rows=51 width=1259) (actual time=0.271..0.803 rows=51 loops=1)
                    Buffers: shared hit=407
                    ->  Merge Append  (cost=47.20..3707.70 rows=166 width=1259) (actual time=0.270..0.794 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=407
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.106..0.156 rows=2 loops=1)
                                Buffers: shared hit=66
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.106..0.155 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=66
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.100..0.149 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=66
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.019..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.090..0.093 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.090..0.092 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.062..0.083 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.081 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.046..0.048 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..536.95 rows=51 width=1259) (actual time=0.033..0.490 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=1.14..407094.63 rows=38748 width=1259) (actual time=0.032..0.482 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=1.14..407094.63 rows=38748 width=1251) (actual time=0.030..0.465 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Nested Loop Semi Join  (cost=0.84..394030.21 rows=38748 width=1251) (actual time=0.022..0.270 rows=50 loops=1)
                                                  Buffers: shared hit=206
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.007..0.065 rows=50 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 2
                                                        Buffers: shared hit=55
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.003..0.003 rows=1 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.002..0.003 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.717 ms
Execution Time: 1.380 ms
```

#### root (10111 units, 380048 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=0.439..1.756 rows=51 loops=1)
  Buffers: shared hit=713
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=0.411..1.467 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=560
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=0.395..1.293 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=407
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=0.391..1.235 rows=51 loops=1)
                    Buffers: shared hit=407
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=0.389..1.224 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=407
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.138..0.203 rows=2 loops=1)
                                Buffers: shared hit=66
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.138..0.201 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=66
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.128..0.191 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=66
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.035..0.036 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.034..0.035 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.028..0.029 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.129..0.133 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.129..0.132 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.085..0.118 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.082..0.113 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.059..0.063 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.045..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.034..0.035 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.057..0.803 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.057..0.793 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.052..0.770 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.040..0.500 rows=50 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=206
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.006..0.006 rows=1 loops=50)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=151
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.005..0.005 rows=1 loops=50)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.696 ms
Execution Time: 2.023 ms
```

#### root (10111 units, 380048 tickets) W1

```
Nested Loop Left Join  (cost=47.78..1189.61 rows=1 width=1301) (actual time=0.301..1.250 rows=51 loops=1)
  Buffers: shared hit=713
  ->  Nested Loop  (cost=47.49..1181.27 rows=1 width=1266) (actual time=0.285..1.035 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=560
        ->  Subquery Scan on x  (cost=47.20..1172.96 rows=1 width=1259) (actual time=0.275..0.910 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=407
              ->  Limit  (cost=47.20..1171.81 rows=51 width=1259) (actual time=0.273..0.868 rows=51 loops=1)
                    Buffers: shared hit=407
                    ->  Merge Append  (cost=47.20..3707.70 rows=166 width=1259) (actual time=0.272..0.859 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=407
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.111..0.160 rows=2 loops=1)
                                Buffers: shared hit=66
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.110..0.159 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=66
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.105..0.153 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=66
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.094 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.093 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.057..0.083 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.055..0.080 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.042 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..536.95 rows=51 width=1259) (actual time=0.033..0.551 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=1.14..407094.63 rows=38748 width=1259) (actual time=0.033..0.543 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=1.14..407094.63 rows=38748 width=1251) (actual time=0.031..0.526 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Nested Loop Semi Join  (cost=0.84..394030.21 rows=38748 width=1251) (actual time=0.023..0.318 rows=50 loops=1)
                                                  Buffers: shared hit=206
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.008..0.066 rows=50 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 2
                                                        Buffers: shared hit=55
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.005 ms
Execution Time: 1.448 ms
```

#### root (10111 units, 380048 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.157..1.164 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.041..1.059 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.040..0.944 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.040..0.795 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.039..0.686 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.038..0.563 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.038..0.466 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.029..0.322 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.398 ms
Execution Time: 1.309 ms
```

#### area (10 units, 220 tickets) probe K=1000

```
Aggregate  (cost=22.56..22.57 rows=1 width=8) (actual time=0.383..0.386 rows=1 loops=1)
  Buffers: shared hit=37
  ->  Limit  (cost=4.92..22.23 rows=26 width=4) (actual time=0.121..0.326 rows=220 loops=1)
        Buffers: shared hit=37
        ->  Result  (cost=4.92..22.23 rows=26 width=4) (actual time=0.119..0.289 rows=220 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=37
              ->  Nested Loop  (cost=4.92..22.23 rows=26 width=0) (actual time=0.106..0.228 rows=220 loops=1)
                    Buffers: shared hit=37
                    ->  HashAggregate  (cost=4.48..4.51 rows=3 width=16) (actual time=0.058..0.063 rows=10 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.041..0.045 rows=10 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=4
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=28 width=16) (actual time=0.008..0.012 rows=22 loops=10)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=33
Planning Time: 0.582 ms
Execution Time: 0.484 ms
```

#### district (101 units, 3379 tickets) probe K=1000

```
Aggregate  (cost=595.13..595.14 rows=1 width=8) (actual time=1.115..1.119 rows=1 loops=1)
  Buffers: shared hit=114
  ->  Limit  (cost=11.21..583.60 rows=922 width=4) (actual time=0.125..1.009 rows=1001 loops=1)
        Buffers: shared hit=114
        ->  Result  (cost=11.21..583.60 rows=922 width=4) (actual time=0.123..0.882 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=114
              ->  Nested Loop  (cost=11.21..583.60 rows=922 width=0) (actual time=0.110..0.696 rows=1001 loops=1)
                    Buffers: shared hit=114
                    ->  HashAggregate  (cost=10.78..11.83 rows=105 width=16) (actual time=0.085..0.099 rows=34 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.52 rows=105 width=16) (actual time=0.038..0.054 rows=101 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.17 rows=28 width=16) (actual time=0.008..0.012 rows=29 loops=34)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=109
Planning Time: 0.474 ms
Execution Time: 1.201 ms
```

#### district thin (101 units, 600 tickets) probe K=1000

```
Aggregate  (cost=589.08..589.09 rows=1 width=8) (actual time=0.754..0.756 rows=1 loops=1)
  Buffers: shared hit=341
  ->  Limit  (cost=11.19..577.67 rows=913 width=4) (actual time=0.109..0.705 rows=600 loops=1)
        Buffers: shared hit=341
        ->  Result  (cost=11.19..577.67 rows=913 width=4) (actual time=0.108..0.645 rows=600 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=341
              ->  Nested Loop  (cost=11.19..577.67 rows=913 width=0) (actual time=0.099..0.554 rows=600 loops=1)
                    Buffers: shared hit=341
                    ->  HashAggregate  (cost=10.75..11.79 rows=104 width=16) (actual time=0.078..0.093 rows=101 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.49 rows=104 width=16) (actual time=0.026..0.043 rows=101 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.16 rows=28 width=16) (actual time=0.003..0.004 rows=6 loops=101)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=336
Planning Time: 0.387 ms
Execution Time: 0.802 ms
```

#### district hot (101 units, 22996 tickets) probe K=1000

```
Aggregate  (cost=629.43..629.44 rows=1 width=8) (actual time=1.409..1.412 rows=1 loops=1)
  Buffers: shared hit=137
  ->  Limit  (cost=11.37..617.15 rows=983 width=4) (actual time=0.161..1.301 rows=1001 loops=1)
        Buffers: shared hit=137
        ->  Result  (cost=11.37..617.15 rows=983 width=4) (actual time=0.159..1.170 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=137
              ->  Nested Loop  (cost=11.37..617.15 rows=983 width=0) (actual time=0.147..0.991 rows=1001 loops=1)
                    Buffers: shared hit=137
                    ->  HashAggregate  (cost=10.94..12.05 rows=112 width=16) (actual time=0.111..0.134 rows=41 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.039..0.067 rows=101 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.12 rows=28 width=16) (actual time=0.007..0.014 rows=24 loops=41)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=132
Planning Time: 0.555 ms
Execution Time: 1.499 ms
```

#### region (1011 units, 37930 tickets) probe K=1000

```
Aggregate  (cost=383.81..383.82 rows=1 width=8) (actual time=5.278..5.282 rows=1 loops=1)
  Buffers: shared hit=120
  ->  Limit  (cost=121.27..371.30 rows=1001 width=4) (actual time=1.800..5.094 rows=1001 loops=1)
        Buffers: shared hit=120
        ->  Result  (cost=121.27..2384.24 rows=9060 width=4) (actual time=1.797..4.964 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=120
              ->  Merge Semi Join  (cost=121.27..2384.24 rows=9060 width=0) (actual time=1.782..4.773 rows=1001 loops=1)
                    Merge Cond: (r.org_unit_id = acl_ou.descendant_id)
                    Buffers: shared hit=120
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4037.03 rows=188149 width=16) (actual time=0.040..2.083 rows=12028 loops=1)
                          Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                          Heap Fetches: 0
                          Buffers: shared hit=105
                    ->  Sort  (cost=120.71..123.29 rows=1032 width=16) (actual time=0.570..0.584 rows=38 loops=1)
                          Sort Key: acl_ou.descendant_id
                          Sort Method: quicksort  Memory: 48kB
                          Buffers: shared hit=15
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..69.06 rows=1032 width=16) (actual time=0.025..0.338 rows=1011 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=15
Planning Time: 0.524 ms
Execution Time: 5.370 ms
```

#### root (10111 units, 380048 tickets) probe K=1000

```
Aggregate  (cost=872.05..872.06 rows=1 width=8) (actual time=6.375..6.378 rows=1 loops=1)
  Buffers: shared hit=233
  ->  Limit  (cost=695.67..859.54 rows=1001 width=4) (actual time=5.678..6.293 rows=1001 loops=1)
        Buffers: shared hit=233
        ->  Result  (cost=695.67..15184.43 rows=88505 width=4) (actual time=5.676..6.190 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=233
              ->  Nested Loop  (cost=695.67..15184.43 rows=88505 width=0) (actual time=5.665..6.038 rows=1001 loops=1)
                    Buffers: shared hit=233
                    ->  HashAggregate  (cost=695.24..763.22 rows=6798 width=16) (actual time=5.622..5.637 rows=39 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 913kB
                          Buffers: shared hit=110
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..670.03 rows=10081 width=16) (actual time=0.032..2.026 rows=10111 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=110
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..1.84 rows=28 width=16) (actual time=0.004..0.007 rows=26 loops=39)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=123
Planning Time: 0.377 ms
Execution Time: 6.503 ms
```

## 2026-11-21T04:52:55Z phase=paired secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-it-pgb-noprep-1851634772 nava-it-pgb-pooled-1851634772 nava-it-pg-1851634772 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

Transport floor (`SELECT 1` x400, 1 client): p50 2.79, p95 5.27 ms. P3-36 gate (p50 <= 3 ms): **PASS**.

| Set | variant | pairs | base p50 | base p95 | var p50 | var p95 | Δp50 | **Δp95** | median per-pair Δ | probe picked C1 / W0 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | C1 | 184 | 36.03 | 60.94 | 41.91 | 64.12 | +5.88 | **+3.18** | +3.73 |  |
| area (10 units, 220 tickets) | W0 | 86 | 51.99 | 86.69 | 108.36 | 181.09 | +56.36 | **+94.39** | +53.43 |  |
| area (10 units, 220 tickets) | probe only K=250 | 146 | 45.11 | 74.86 | 53.34 | 81.01 | +8.23 | **+6.16** | +6.68 |  |
| area (10 units, 220 tickets) | M K=250 | 154 | 40.37 | 65.46 | 51.37 | 85.32 | +11.00 | **+19.86** | +11.51 | C1 |
| area (10 units, 220 tickets) | probe only K=1000 | 139 | 41.40 | 116.78 | 47.04 | 120.19 | +5.64 | **+3.41** | +5.95 |  |
| area (10 units, 220 tickets) | M K=1000 | 105 | 56.93 | 112.14 | 71.63 | 144.63 | +14.70 | **+32.50** | +14.64 | C1 |
| district (101 units, 3379 tickets) | C1 | 104 | 60.88 | 144.49 | 64.31 | 146.34 | +3.43 | **+1.85** | +4.37 |  |
| district (101 units, 3379 tickets) | W0 | 145 | 38.12 | 129.15 | 43.76 | 118.60 | +5.64 | **-10.56** | +6.41 |  |
| district (101 units, 3379 tickets) | probe only K=250 | 161 | 37.50 | 89.89 | 41.97 | 102.91 | +4.47 | **+13.02** | +6.36 |  |
| district (101 units, 3379 tickets) | M K=250 | 141 | 39.66 | 77.38 | 54.13 | 114.81 | +14.48 | **+37.43** | +13.59 | W0 |
| district (101 units, 3379 tickets) | probe only K=1000 | 158 | 40.66 | 65.11 | 48.35 | 81.41 | +7.68 | **+16.30** | +7.96 |  |
| district (101 units, 3379 tickets) | M K=1000 | 140 | 45.36 | 69.08 | 59.02 | 85.35 | +13.66 | **+16.27** | +13.78 | W0 |
| district thin (101 units, 600 tickets) | C1 | 192 | 29.94 | 53.27 | 40.94 | 63.82 | +11.00 | **+10.55** | +10.05 |  |
| district thin (101 units, 600 tickets) | W0 | 221 | 26.54 | 39.27 | 38.20 | 60.95 | +11.66 | **+21.68** | +11.27 |  |
| district thin (101 units, 600 tickets) | probe only K=250 | 267 | 24.83 | 36.77 | 29.40 | 38.15 | +4.57 | **+1.38** | +4.01 |  |
| district thin (101 units, 600 tickets) | M K=250 | 270 | 20.26 | 29.44 | 33.06 | 50.56 | +12.80 | **+21.13** | +11.31 | W0 |
| district thin (101 units, 600 tickets) | probe only K=1000 | 310 | 21.62 | 30.72 | 26.08 | 35.18 | +4.46 | **+4.46** | +3.96 |  |
| district thin (101 units, 600 tickets) | M K=1000 | 266 | 21.00 | 30.38 | 33.97 | 51.18 | +12.96 | **+20.80** | +12.19 | C1 |
| district hot (101 units, 22996 tickets) | C1 | 290 | 23.36 | 33.92 | 26.18 | 36.49 | +2.82 | **+2.58** | +2.41 |  |
| district hot (101 units, 22996 tickets) | W0 | 115 | 59.21 | 129.71 | 54.88 | 152.47 | -4.34 | **+22.75** | +0.55 |  |
| district hot (101 units, 22996 tickets) | probe only K=250 | 116 | 40.19 | 153.74 | 48.66 | 192.05 | +8.47 | **+38.32** | +6.29 |  |
| district hot (101 units, 22996 tickets) | M K=250 | 69 | 88.60 | 175.05 | 115.85 | 279.76 | +27.25 | **+104.71** | +21.95 | W0 |
| district hot (101 units, 22996 tickets) | probe only K=1000 | 82 | 63.47 | 203.01 | 80.69 | 257.77 | +17.22 | **+54.77** | +15.14 |  |
| district hot (101 units, 22996 tickets) | M K=1000 | 159 | 23.86 | 127.72 | 29.71 | 174.67 | +5.85 | **+46.96** | +5.83 | W0 |
| region (1011 units, 37930 tickets) | C1 | 277 | 23.15 | 40.85 | 25.43 | 48.15 | +2.29 | **+7.30** | +2.40 |  |
| region (1011 units, 37930 tickets) | W0 | 182 | 25.23 | 81.62 | 28.39 | 102.49 | +3.16 | **+20.86** | +3.23 |  |
| region (1011 units, 37930 tickets) | probe only K=250 | 123 | 41.25 | 111.65 | 50.30 | 108.95 | +9.05 | **-2.69** | +7.22 |  |
| region (1011 units, 37930 tickets) | M K=250 | 192 | 25.27 | 61.27 | 32.28 | 109.93 | +7.01 | **+48.67** | +7.03 | W0 |
| region (1011 units, 37930 tickets) | probe only K=1000 | 112 | 51.17 | 110.94 | 64.21 | 154.48 | +13.04 | **+43.53** | +7.00 |  |
| region (1011 units, 37930 tickets) | M K=1000 | 129 | 45.38 | 98.74 | 59.82 | 103.76 | +14.44 | **+5.02** | +12.84 | W0 |
| root (10111 units, 380048 tickets) | C1 | 220 | 28.08 | 67.12 | 29.89 | 68.79 | +1.81 | **+1.67** | +1.19 |  |
| root (10111 units, 380048 tickets) | W0 | 238 | 28.47 | 47.55 | 30.38 | 50.32 | +1.91 | **+2.77** | +1.50 |  |
| root (10111 units, 380048 tickets) | probe only K=250 | 179 | 27.94 | 75.75 | 33.90 | 109.13 | +5.96 | **+33.38** | +5.34 |  |
| root (10111 units, 380048 tickets) | M K=250 | 293 | 19.38 | 49.32 | 24.03 | 55.52 | +4.65 | **+6.20** | +4.47 | W0 |
| root (10111 units, 380048 tickets) | probe only K=1000 | 110 | 44.86 | 125.06 | 57.99 | 181.17 | +13.14 | **+56.11** | +12.07 |  |
| root (10111 units, 380048 tickets) | M K=1000 | 138 | 42.60 | 80.80 | 61.31 | 95.21 | +18.71 | **+14.41** | +17.07 | W0 |

Transport floor at end: p50 4.39, p95 12.02 ms. Foreign containers at end: `nava-iso-pgb-2371064927 nava-iso-pg-2371064927 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T05:07:23Z phase=server secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-iso-pg-2388326083 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### Server-side paired pass: planning + execution ms per request (count + page [+ probe]), EXPLAIN (ANALYZE, TIMING OFF), 300 pairs per variant, baseline and variant alternate

| Set | variant | pairs | base p50 | base p95 | var p50 | var p95 | Δp50 | **Δp95** | median per-pair Δ | p95 per-pair Δ |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | C1 | 300 | 6.43 | 10.61 | 9.63 | 15.33 | +3.21 | **+4.72** | +3.23 | +7.78 |
| area (10 units, 220 tickets) | W0 | 300 | 5.93 | 10.81 | 36.34 | 82.29 | +30.41 | **+71.48** | +30.56 | +72.12 |
| area (10 units, 220 tickets) | probe K=100 | 300 | 5.38 | 8.08 | 6.00 | 8.61 | +0.62 | **+0.53** | +0.40 | +2.79 |
| area (10 units, 220 tickets) | probe K=250 | 300 | 5.12 | 6.85 | 5.62 | 7.48 | +0.50 | **+0.63** | +0.47 | +2.49 |
| area (10 units, 220 tickets) | probe K=500 | 300 | 4.94 | 7.38 | 5.46 | 7.95 | +0.52 | **+0.57** | +0.52 | +2.23 |
| area (10 units, 220 tickets) | probe K=1000 | 300 | 4.44 | 7.30 | 4.99 | 7.86 | +0.56 | **+0.56** | +0.41 | +2.54 |
| area (10 units, 220 tickets) | M K=250 | 300 | 4.20 | 6.53 | 6.97 | 9.67 | +2.77 | **+3.14** | +2.43 | +4.46 |
| area (10 units, 220 tickets) | M K=1000 | 300 | 5.01 | 6.69 | 7.79 | 10.54 | +2.78 | **+3.86** | +2.66 | +5.04 |
| district (101 units, 3379 tickets) | C1 | 300 | 4.25 | 6.61 | 7.06 | 12.05 | +2.82 | **+5.45** | +2.89 | +6.29 |
| district (101 units, 3379 tickets) | W0 | 300 | 4.40 | 6.07 | 8.00 | 11.63 | +3.60 | **+5.56** | +3.50 | +6.55 |
| district (101 units, 3379 tickets) | probe K=100 | 300 | 5.82 | 10.98 | 6.67 | 12.42 | +0.85 | **+1.44** | +0.77 | +3.53 |
| district (101 units, 3379 tickets) | probe K=250 | 300 | 6.30 | 11.54 | 6.63 | 11.56 | +0.32 | **+0.01** | +0.44 | +3.24 |
| district (101 units, 3379 tickets) | probe K=500 | 300 | 5.91 | 9.05 | 6.60 | 10.01 | +0.69 | **+0.96** | +0.63 | +3.67 |
| district (101 units, 3379 tickets) | probe K=1000 | 300 | 5.61 | 8.48 | 6.63 | 9.34 | +1.03 | **+0.86** | +0.95 | +3.53 |
| district (101 units, 3379 tickets) | M K=250 | 300 | 6.15 | 9.55 | 11.99 | 19.99 | +5.85 | **+10.43** | +5.92 | +11.29 |
| district (101 units, 3379 tickets) | M K=1000 | 300 | 6.72 | 13.24 | 14.01 | 28.73 | +7.29 | **+15.49** | +7.00 | +17.21 |
| district thin (101 units, 600 tickets) | C1 | 300 | 7.22 | 13.38 | 17.23 | 41.84 | +10.01 | **+28.45** | +8.61 | +29.55 |
| district thin (101 units, 600 tickets) | W0 | 300 | 6.56 | 10.75 | 15.29 | 36.77 | +8.73 | **+26.02** | +8.25 | +26.72 |
| district thin (101 units, 600 tickets) | probe K=100 | 300 | 7.76 | 22.33 | 8.81 | 24.15 | +1.05 | **+1.82** | +1.08 | +9.59 |
| district thin (101 units, 600 tickets) | probe K=250 | 300 | 5.56 | 8.30 | 6.30 | 8.71 | +0.74 | **+0.41** | +0.70 | +2.61 |
| district thin (101 units, 600 tickets) | probe K=500 | 300 | 5.94 | 9.27 | 7.08 | 10.02 | +1.14 | **+0.74** | +0.98 | +3.13 |
| district thin (101 units, 600 tickets) | probe K=1000 | 300 | 4.94 | 8.88 | 5.83 | 10.65 | +0.89 | **+1.77** | +0.83 | +3.46 |
| district thin (101 units, 600 tickets) | M K=250 | 300 | 5.37 | 8.46 | 13.00 | 26.70 | +7.63 | **+18.24** | +7.17 | +20.76 |
| district thin (101 units, 600 tickets) | M K=1000 | 300 | 8.01 | 14.30 | 21.18 | 50.78 | +13.17 | **+36.48** | +11.91 | +38.80 |
| district hot (101 units, 22996 tickets) | C1 | 300 | 7.28 | 11.34 | 10.04 | 16.16 | +2.77 | **+4.83** | +2.71 | +7.32 |
| district hot (101 units, 22996 tickets) | W0 | 300 | 5.78 | 9.25 | 7.77 | 12.38 | +1.99 | **+3.12** | +1.83 | +5.63 |
| district hot (101 units, 22996 tickets) | probe K=100 | 300 | 5.89 | 8.88 | 6.44 | 9.76 | +0.55 | **+0.87** | +0.59 | +3.59 |
| district hot (101 units, 22996 tickets) | probe K=250 | 300 | 6.69 | 9.05 | 7.23 | 10.00 | +0.55 | **+0.96** | +0.66 | +3.17 |
| district hot (101 units, 22996 tickets) | probe K=500 | 300 | 6.18 | 10.49 | 7.22 | 13.23 | +1.04 | **+2.74** | +0.88 | +4.33 |
| district hot (101 units, 22996 tickets) | probe K=1000 | 300 | 5.78 | 8.41 | 6.71 | 9.69 | +0.93 | **+1.29** | +0.95 | +3.71 |
| district hot (101 units, 22996 tickets) | M K=250 | 300 | 6.50 | 11.88 | 10.03 | 20.03 | +3.53 | **+8.15** | +3.30 | +9.30 |
| district hot (101 units, 22996 tickets) | M K=1000 | 300 | 5.61 | 8.71 | 8.89 | 13.32 | +3.28 | **+4.61** | +2.85 | +7.31 |
| region (1011 units, 37930 tickets) | C1 | 300 | 4.61 | 7.57 | 5.94 | 10.52 | +1.33 | **+2.94** | +1.35 | +4.04 |
| region (1011 units, 37930 tickets) | W0 | 300 | 5.61 | 8.32 | 7.26 | 11.08 | +1.64 | **+2.75** | +1.56 | +5.07 |
| region (1011 units, 37930 tickets) | probe K=100 | 300 | 5.74 | 8.60 | 7.21 | 9.78 | +1.47 | **+1.18** | +1.19 | +4.42 |
| region (1011 units, 37930 tickets) | probe K=250 | 300 | 6.45 | 12.39 | 7.53 | 13.91 | +1.07 | **+1.52** | +1.09 | +4.04 |
| region (1011 units, 37930 tickets) | probe K=500 | 300 | 6.08 | 9.37 | 7.65 | 11.51 | +1.57 | **+2.14** | +1.57 | +4.40 |
| region (1011 units, 37930 tickets) | probe K=1000 | 300 | 5.93 | 9.60 | 8.46 | 12.74 | +2.53 | **+3.14** | +2.39 | +6.04 |
| region (1011 units, 37930 tickets) | M K=250 | 300 | 6.06 | 8.40 | 9.51 | 13.22 | +3.45 | **+4.81** | +3.30 | +6.60 |
| region (1011 units, 37930 tickets) | M K=1000 | 300 | 5.79 | 8.57 | 9.80 | 14.43 | +4.01 | **+5.86** | +3.96 | +8.30 |
| root (10111 units, 380048 tickets) | C1 | 300 | 5.76 | 9.01 | 5.97 | 9.98 | +0.21 | **+0.97** | +0.39 | +2.83 |
| root (10111 units, 380048 tickets) | W0 | 300 | 6.00 | 8.56 | 6.24 | 9.43 | +0.23 | **+0.88** | +0.19 | +2.71 |
| root (10111 units, 380048 tickets) | probe K=100 | 300 | 5.97 | 8.50 | 6.56 | 9.43 | +0.59 | **+0.93** | +0.55 | +3.27 |
| root (10111 units, 380048 tickets) | probe K=250 | 300 | 6.05 | 8.66 | 6.89 | 9.60 | +0.84 | **+0.94** | +0.89 | +3.20 |
| root (10111 units, 380048 tickets) | probe K=500 | 300 | 6.13 | 9.24 | 7.82 | 11.37 | +1.69 | **+2.13** | +1.67 | +4.38 |
| root (10111 units, 380048 tickets) | probe K=1000 | 300 | 5.75 | 8.47 | 11.12 | 14.75 | +5.37 | **+6.28** | +5.14 | +9.25 |
| root (10111 units, 380048 tickets) | M K=250 | 300 | 5.81 | 8.80 | 6.99 | 10.33 | +1.18 | **+1.53** | +1.09 | +4.20 |
| root (10111 units, 380048 tickets) | M K=1000 | 300 | 5.70 | 8.32 | 10.73 | 15.07 | +5.03 | **+6.75** | +4.96 | +8.98 |

Foreign containers at end: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

## 2026-11-21T05:30:10Z phase=explain secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### Plans: region branch of the page statement, agents[0]

| Set | leaf | page plan / exec ms | region branch ms | region branch plan | records read / removed | closure rows |
| --- | --- | --- | --- | --- | --- | --- |
| baseline | none | 4.5 / 6.2 | n/a | n/a | n/a | n/a |
| area (10 units, 220 tickets) | C1 | 2.7 / 5.2 | 2.03 | Limit > Sort > Result > Nested Loop > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey | 220 / 10 | 10 |
| area (10 units, 220 tickets) | W0 | 2.8 / 52.7 | 50.29 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 6396 / 6395 | 0 |
| area (10 units, 220 tickets) | W1 | 1.9 / 3.4 | 1.27 | Limit > Sort > Result > Nested Loop > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Scan records_org_unit > Result > Index Only Scan record_shares_pkey | 220 / 10 | 10 |
| area (10 units, 220 tickets) | C2 | 2.2 / 4.1 | 1.05 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 220 / 10 | 10 |
| area (10 units, 220 tickets) | W0, no AND NOT | 2.2 / 33.9 | 31.13 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 6396 / 6395 | 0 |
| area (10 units, 220 tickets) | C1, no AND NOT | 1.8 / 2.8 | 0.51 | Limit > Sort > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Scan records_org_unit | 220 / 0 | 10 |
| area (10 units, 220 tickets) | W0 count | 1.3 / 1.5 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| district (101 units, 3379 tickets) | C1 | 3.6 / 6.3 | 4.65 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 949 / 96 | 0 |
| district (101 units, 3379 tickets) | W0 | 1.8 / 5.4 | 4.01 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 949 / 942 | 0 |
| district (101 units, 3379 tickets) | W1 | 2.6 / 6.2 | 4.34 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 949 / 96 | 0 |
| district (101 units, 3379 tickets) | C2 | 2.0 / 10.4 | 8.09 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 3379 / 184 | 101 |
| district (101 units, 3379 tickets) | W0, no AND NOT | 2.7 / 7.5 | 5.59 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 949 / 942 | 0 |
| district (101 units, 3379 tickets) | C1, no AND NOT | 2.7 / 5.0 | 3.55 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 949 / 41 | 0 |
| district (101 units, 3379 tickets) | W0 count | 2.3 / 2.4 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| district thin (101 units, 600 tickets) | C1 | 2.1 / 5.2 | 3.34 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 913 / 92 | 0 |
| district thin (101 units, 600 tickets) | W0 | 1.6 / 6.3 | 4.17 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 913 / 912 | 0 |
| district thin (101 units, 600 tickets) | W1 | 2.0 / 5.0 | 3.25 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 913 / 92 | 0 |
| district thin (101 units, 600 tickets) | C2 | 1.6 / 4.2 | 2.17 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 600 / 37 | 101 |
| district thin (101 units, 600 tickets) | W0, no AND NOT | 2.0 / 8.3 | 5.41 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 913 / 912 | 0 |
| district thin (101 units, 600 tickets) | C1, no AND NOT | 2.0 / 5.4 | 3.34 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 913 / 39 | 0 |
| district thin (101 units, 600 tickets) | W0 count | 1.8 / 1.7 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| district hot (101 units, 22996 tickets) | C1 | 2.0 / 3.4 | 1.82 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 448 / 45 | 0 |
| district hot (101 units, 22996 tickets) | W0 | 2.1 / 3.5 | 2.14 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 448 / 422 | 0 |
| district hot (101 units, 22996 tickets) | W1 | 2.1 / 3.0 | 1.63 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 448 / 45 | 0 |
| district hot (101 units, 22996 tickets) | C2 | 3.0 / 80.2 | 78.25 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 22996 / 1436 | 101 |
| district hot (101 units, 22996 tickets) | W0, no AND NOT | 1.6 / 2.8 | 1.66 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 416 / 389 | 0 |
| district hot (101 units, 22996 tickets) | C1, no AND NOT | 3.0 / 4.6 | 2.52 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 416 / 18 | 0 |
| district hot (101 units, 22996 tickets) | W0 count | 2.3 / 1.2 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| region (1011 units, 37930 tickets) | C1 | 3.3 / 4.6 | 2.42 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 329 / 33 | 0 |
| region (1011 units, 37930 tickets) | W0 | 2.2 / 4.5 | 2.31 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 329 / 298 | 0 |
| region (1011 units, 37930 tickets) | W1 | 2.3 / 3.9 | 1.98 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 329 / 33 | 0 |
| region (1011 units, 37930 tickets) | C2 | 4.8 / 148.5 | 146.73 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 37930 / 2320 | 1011 |
| region (1011 units, 37930 tickets) | W0, no AND NOT | 2.9 / 2.7 | 1.40 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 329 / 298 | 0 |
| region (1011 units, 37930 tickets) | C1, no AND NOT | 2.2 / 3.2 | 1.57 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 329 / 15 | 0 |
| region (1011 units, 37930 tickets) | W0 count | 1.2 / 1.3 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |
| root (10111 units, 380048 tickets) | C1 | 2.4 / 1.7 | 0.73 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 52 / 2 | 50 |
| root (10111 units, 380048 tickets) | W0 | 2.6 / 1.6 | 0.55 | Limit > Result > Nested Loop > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 52 / 2 | 50 |
| root (10111 units, 380048 tickets) | W1 | 2.3 / 1.9 | 0.69 | Limit > Result > Nested Loop > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey > Result > Index Only Scan record_shares_pkey | 52 / 2 | 50 |
| root (10111 units, 380048 tickets) | C2 | 2.3 / 1154.5 | 1153.34 | Limit > InitPlan(Result) > Index Only Scan org_unit_closure_pkey > Sort > Result > Nested Loop > Index Scan records_org_unit > Materialize > Result > Bitmap Heap Scan record_shares > BitmapOr > Bitmap Index Scan record_shares_principal > Bitmap Index Scan record_shares_principal | 380048 / 23400 | 10111 |
| root (10111 units, 380048 tickets) | W0, no AND NOT | 1.3 / 1.1 | 0.28 | Limit > Result > Index Scan records_list_updated > SubPlan(Result) > Index Only Scan org_unit_closure_pkey | 51 / 1 | 50 |
| root (10111 units, 380048 tickets) | C1, no AND NOT | 1.5 / 1.4 | 0.34 | Limit > Result > Nested Loop > Index Scan records_list_updated > Index Only Scan org_unit_closure_pkey | 51 / 1 | 50 |
| root (10111 units, 380048 tickets) | W0 count | 1.2 / 1.2 | 0.00 | Subquery Scan [never executed] > Limit [never executed] > Result [never executed] > Nested Loop [never executed] > Index Scan records_org_unit [never executed] > SubPlan(Result) [never executed] > Index Only Scan org_unit_closure_pkey [never executed] > Materialize [never executed] > Result [never executed] > Bitmap Heap Scan record_shares [never executed] > BitmapOr [never executed] > Bitmap Index Scan record_shares_principal [never executed] > Bitmap Index Scan record_shares_principal [never executed] | 0 / 0 | 0 |

### Probe plans (records counted = min(all, K+1))

| Set | K | records under root (any type) | probe plan / exec ms | plan | records read |
| --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | 250 | 220 | 0.48 / 0.39 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 220 |
| area (10 units, 220 tickets) | 1000 | 220 | 0.55 / 0.45 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 220 |
| area (10 units, 220 tickets) | 4000 | 220 | 0.27 / 0.28 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 220 |
| district (101 units, 3379 tickets) | 250 | 3379 | 0.56 / 0.58 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 252 |
| district (101 units, 3379 tickets) | 1000 | 3379 | 0.28 / 0.72 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 986 |
| district (101 units, 3379 tickets) | 4000 | 3379 | 0.58 / 4.07 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 3333 |
| district thin (101 units, 600 tickets) | 250 | 600 | 0.54 / 0.73 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 245 |
| district thin (101 units, 600 tickets) | 1000 | 600 | 0.54 / 1.38 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 606 |
| district thin (101 units, 600 tickets) | 4000 | 600 | 0.55 / 1.48 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 606 |
| district hot (101 units, 22996 tickets) | 250 | 22996 | 0.54 / 0.50 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 252 |
| district hot (101 units, 22996 tickets) | 1000 | 22996 | 0.55 / 1.43 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 984 |
| district hot (101 units, 22996 tickets) | 4000 | 22996 | 0.29 / 1.81 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 4004 |
| region (1011 units, 37930 tickets) | 250 | 37930 | 0.40 / 1.27 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 252 |
| region (1011 units, 37930 tickets) | 1000 | 37930 | 0.50 / 2.62 | Aggregate > Limit > Result > Merge Join > Index Only Scan records_org_unit > Sort > Index Only Scan org_unit_closure_pkey | 12028 |
| region (1011 units, 37930 tickets) | 4000 | 37930 | 0.26 / 12.84 | Aggregate > Limit > Result > Merge Join > Index Only Scan records_org_unit > Sort > Index Only Scan org_unit_closure_pkey | 44656 |
| root (10111 units, 380048 tickets) | 250 | 380048 | 0.39 / 0.82 | Aggregate > Limit > Result > Nested Loop > Index Only Scan records_org_unit > Index Only Scan org_unit_closure_pkey | 251 |
| root (10111 units, 380048 tickets) | 1000 | 380048 | 0.48 / 6.07 | Aggregate > Limit > Result > Nested Loop > Aggregate > Index Only Scan org_unit_closure_pkey > Index Only Scan records_org_unit | 1014 |
| root (10111 units, 380048 tickets) | 4000 | 380048 | 0.50 / 10.37 | Aggregate > Limit > Result > Hash Join > Index Only Scan records_org_unit > Hash > Index Only Scan org_unit_closure_pkey | 4001 |

### Plans

#### baseline

```
Nested Loop Left Join  (cost=46.63..1449.11 rows=1 width=1301) (actual time=0.416..2.824 rows=51 loops=1)
  Buffers: shared hit=1139
  ->  Nested Loop  (cost=46.33..1440.78 rows=1 width=1266) (actual time=0.391..2.473 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=986
        ->  Subquery Scan on x  (cost=46.05..1432.46 rows=1 width=1259) (actual time=0.374..2.258 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=833
              ->  Limit  (cost=46.05..1431.32 rows=51 width=1259) (actual time=0.371..2.183 rows=51 loops=1)
                    Buffers: shared hit=833
                    ->  Merge Append  (cost=46.05..3169.69 rows=115 width=1259) (actual time=0.369..2.167 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=833
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.163..1.926 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.162..1.915 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.153..1.876 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.031..0.038 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.030..0.036 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.026..0.030 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.029 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.028 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.024..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.144..0.151 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.143..0.150 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.096..0.135 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.093..0.130 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.071..0.076 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.051..0.063 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.039..0.041 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.444 ms
Execution Time: 3.122 ms
```

#### area (10 units, 220 tickets) C1

```
Nested Loop Left Join  (cost=631.22..1902.98 rows=1 width=1301) (actual time=1.527..3.191 rows=51 loops=1)
  Buffers: shared hit=1633
  ->  Nested Loop  (cost=630.92..1894.65 rows=1 width=1266) (actual time=1.483..2.983 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1480
        ->  Subquery Scan on x  (cost=630.64..1886.33 rows=1 width=1259) (actual time=1.471..2.849 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1327
              ->  Limit  (cost=630.64..1885.19 rows=51 width=1259) (actual time=1.468..2.799 rows=51 loops=1)
                    Buffers: shared hit=1327
                    ->  Merge Append  (cost=630.64..3754.70 rows=127 width=1259) (actual time=1.467..2.788 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1327
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.121..1.401 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.121..1.394 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.114..1.367 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.018..0.024 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.018..0.023 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.016..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.130..0.139 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.130..0.138 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.100..0.128 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.099..0.125 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.082..0.087 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.070..0.078 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.062..0.064 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.045..0.046 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=584.58..584.61 rows=12 width=1259) (actual time=1.174..1.182 rows=1 loops=1)
                                Buffers: shared hit=494
                                ->  Sort  (cost=584.58..584.61 rows=12 width=1259) (actual time=1.174..1.181 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 205kB
                                      Buffers: shared hit=494
                                      ->  Result  (cost=5.20..584.36 rows=12 width=1259) (actual time=0.075..1.012 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=494
                                            ->  Nested Loop Anti Join  (cost=5.20..584.36 rows=12 width=1251) (actual time=0.072..0.952 rows=210 loops=1)
                                                  Buffers: shared hit=494
                                                  ->  Nested Loop  (cost=4.91..580.30 rows=12 width=1251) (actual time=0.055..0.279 rows=210 loops=1)
                                                        Buffers: shared hit=73
                                                        ->  HashAggregate  (cost=4.48..4.51 rows=3 width=16) (actual time=0.019..0.024 rows=10 loops=1)
                                                              Group Key: acl_ou.descendant_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.014..0.016 rows=10 loops=1)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                        ->  Index Scan using records_org_unit on records r_4  (cost=0.42..191.81 rows=12 width=1267) (actual time=0.005..0.019 rows=21 loops=10)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-70… (559 chars)
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=69
                                                  ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=210)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        Buffers: shared hit=421
                                                        ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=210)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Rows Removed by Filter: 0
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=421
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.483 ms
Execution Time: 3.493 ms
```

#### area (10 units, 220 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=26.659..28.078 rows=51 loops=1)
  Buffers: shared hit=24638
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=26.638..27.906 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=24485
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=26.624..27.777 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=24332
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=26.621..27.732 rows=51 loops=1)
                    Buffers: shared hit=24332
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=26.620..27.723 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=24332
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.143..1.217 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.143..1.189 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.136..1.164 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.021..0.027 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.020..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.017..0.020 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.021..0.022 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.021..0.022 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.018..0.019 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.099..0.105 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.099..0.104 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.066..0.092 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.064..0.087 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.036..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=26.331..26.334 rows=1 loops=1)
                                Buffers: shared hit=23499
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=26.329..26.333 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=23499
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=26.326..26.329 rows=1 loops=1)
                                            Buffers: shared hit=23499
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=26.304..26.305 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 6395
                                                  Buffers: shared hit=23497
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=5679)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=17038
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=5679)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=17038
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.014..0.015 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.254 ms
Execution Time: 28.324 ms
```

#### area (10 units, 220 tickets) W1

```
Nested Loop Left Join  (cost=631.22..1902.98 rows=1 width=1301) (actual time=1.965..3.990 rows=51 loops=1)
  Buffers: shared hit=1633
  ->  Nested Loop  (cost=630.92..1894.65 rows=1 width=1266) (actual time=1.942..3.760 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1480
        ->  Subquery Scan on x  (cost=630.64..1886.33 rows=1 width=1259) (actual time=1.924..3.587 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1327
              ->  Limit  (cost=630.64..1885.19 rows=51 width=1259) (actual time=1.920..3.524 rows=51 loops=1)
                    Buffers: shared hit=1327
                    ->  Merge Append  (cost=630.64..3754.70 rows=127 width=1259) (actual time=1.918..3.512 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1327
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.129..1.687 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.128..1.678 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.121..1.612 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.023..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.023..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.020..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.022..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.115..0.121 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.114..0.120 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.076..0.108 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.074..0.104 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.060 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.041..0.050 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=584.58..584.61 rows=12 width=1259) (actual time=1.624..1.630 rows=1 loops=1)
                                Buffers: shared hit=494
                                ->  Sort  (cost=584.58..584.61 rows=12 width=1259) (actual time=1.623..1.629 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 205kB
                                      Buffers: shared hit=494
                                      ->  Result  (cost=5.20..584.36 rows=12 width=1259) (actual time=0.076..1.405 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=494
                                            ->  Nested Loop Anti Join  (cost=5.20..584.36 rows=12 width=1251) (actual time=0.073..1.321 rows=210 loops=1)
                                                  Buffers: shared hit=494
                                                  ->  Nested Loop  (cost=4.91..580.30 rows=12 width=1251) (actual time=0.062..0.355 rows=210 loops=1)
                                                        Buffers: shared hit=73
                                                        ->  HashAggregate  (cost=4.48..4.51 rows=3 width=16) (actual time=0.023..0.027 rows=10 loops=1)
                                                              Group Key: acl_ou.descendant_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.015..0.018 rows=10 loops=1)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                        ->  Index Scan using records_org_unit on records r_4  (cost=0.42..191.81 rows=12 width=1267) (actual time=0.006..0.024 rows=21 loops=10)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-70… (559 chars)
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=69
                                                  ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=210)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        Buffers: shared hit=421
                                                        ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=210)
                                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Rows Removed by Filter: 0
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=421
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.533 ms
Execution Time: 4.260 ms
```

#### area (10 units, 220 tickets) C2

```
Nested Loop Left Join  (cost=1903.67..2880.92 rows=1 width=1301) (actual time=1.248..3.239 rows=51 loops=1)
  Buffers: shared hit=1225
  ->  Nested Loop  (cost=1903.37..2872.58 rows=1 width=1266) (actual time=1.225..3.035 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1072
        ->  Subquery Scan on x  (cost=1903.09..2864.27 rows=1 width=1259) (actual time=1.209..2.879 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=919
              ->  Limit  (cost=1903.09..2863.12 rows=51 width=1259) (actual time=1.202..2.825 rows=51 loops=1)
                    Buffers: shared hit=919
                    ->  Merge Append  (cost=1903.09..5027.90 rows=166 width=1259) (actual time=1.200..2.814 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=919
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.137..1.711 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.136..1.704 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.126..1.671 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.022..0.029 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.021..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.018..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.025..0.026 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.025..0.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.022..0.023 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.143..0.150 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.142..0.149 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.082..0.116 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.078..0.111 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.058..0.063 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.042..0.052 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1857.03..1857.15 rows=51 width=1259) (actual time=0.869..0.877 rows=1 loops=1)
                                Buffers: shared hit=86
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..4.48 rows=3 width=16) (actual time=0.020..0.024 rows=10 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                        Buffers: shared hit=4
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.018..0.021 rows=10 loops=1)
                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=4
                                ->  Sort  (cost=1852.55..1852.86 rows=123 width=1259) (actual time=0.867..0.873 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 205kB
                                      Buffers: shared hit=86
                                      ->  Result  (cost=21.92..1848.45 rows=123 width=1259) (actual time=0.091..0.636 rows=210 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=86
                                            ->  Nested Loop Anti Join  (cost=21.92..1848.45 rows=123 width=1251) (actual time=0.087..0.570 rows=210 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 630
                                                  Buffers: shared hit=86
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1810.60 rows=123 width=1251) (actual time=0.049..0.274 rows=210 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Rows Removed by Filter: 10
                                                        Buffers: shared hit=73
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=210)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.024..0.032 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.022..0.029 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.016..0.018 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.343 ms
Execution Time: 3.580 ms
```

#### area (10 units, 220 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.799..1.807 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.106..1.652 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.104..1.393 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.103..1.175 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.062..0.981 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.060..0.804 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.059..0.660 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.044..0.449 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 2.371 ms
Execution Time: 2.026 ms
```

#### district (101 units, 3379 tickets) C1

```
Nested Loop Left Join  (cost=47.78..16297.37 rows=1 width=1301) (actual time=0.663..4.867 rows=51 loops=1)
  Buffers: shared hit=4624
  ->  Nested Loop  (cost=47.49..16289.03 rows=1 width=1266) (actual time=0.643..4.654 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4471
        ->  Subquery Scan on x  (cost=47.20..16280.72 rows=1 width=1259) (actual time=0.632..4.424 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4318
              ->  Limit  (cost=47.20..16279.57 rows=51 width=1259) (actual time=0.630..4.376 rows=51 loops=1)
                    Buffers: shared hit=4318
                    ->  Merge Append  (cost=47.20..52881.97 rows=166 width=1259) (actual time=0.628..4.367 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4318
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.162..0.730 rows=44 loops=1)
                                Buffers: shared hit=747
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.161..0.723 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=747
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.117..0.663 rows=44 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=747
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.023..0.028 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.022..0.027 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.019..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.022..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.116..0.122 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.116..0.121 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.068..0.111 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.066..0.107 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.049..0.054 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..49711.22 rows=51 width=1259) (actual time=0.300..3.441 rows=7 loops=1)
                                Buffers: shared hit=3535
                                ->  Result  (cost=1.14..393783.00 rows=404 width=1259) (actual time=0.299..3.439 rows=7 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3535
                                      ->  Nested Loop Anti Join  (cost=1.14..393783.00 rows=404 width=1251) (actual time=0.296..3.432 rows=7 loops=1)
                                            Buffers: shared hit=3535
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.77 rows=404 width=1251) (actual time=0.286..3.373 rows=7 loops=1)
                                                  Buffers: shared hit=3521
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.009..1.620 rows=853 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 96
                                                        Buffers: shared hit=961
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=853)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2560
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=7)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=14
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=7)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=14
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.308 ms
Execution Time: 5.138 ms
```

#### district (101 units, 3379 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=1.033..8.668 rows=51 loops=1)
  Buffers: shared hit=4624
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=0.958..8.280 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4471
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=0.940..8.038 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4318
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=0.937..7.958 rows=51 loops=1)
                    Buffers: shared hit=4318
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=0.935..7.940 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4318
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.157..1.143 rows=44 loops=1)
                                Buffers: shared hit=747
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.156..1.133 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=747
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.148..1.095 rows=44 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=747
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.030..0.037 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.029..0.035 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.025..0.030 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.029 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.028 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.023..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.141..0.150 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.141..0.148 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.094..0.131 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.091..0.126 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.068..0.073 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.049..0.061 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.038..0.039 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.575..6.548 rows=7 loops=1)
                                Buffers: shared hit=3535
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.575..6.544 rows=7 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3535
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.572..6.534 rows=7 loops=1)
                                            Buffers: shared hit=3535
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.555..6.451 rows=7 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 942
                                                  Buffers: shared hit=3521
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.005..0.005 rows=0 loops=853)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=2560
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=853)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=2560
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=7)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=14
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=7)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=14
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.246 ms
Execution Time: 9.026 ms
```

#### district (101 units, 3379 tickets) W1

```
Nested Loop Left Join  (cost=47.78..16297.37 rows=1 width=1301) (actual time=0.513..5.618 rows=51 loops=1)
  Buffers: shared hit=4624
  ->  Nested Loop  (cost=47.49..16289.03 rows=1 width=1266) (actual time=0.497..5.343 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4471
        ->  Subquery Scan on x  (cost=47.20..16280.72 rows=1 width=1259) (actual time=0.485..5.168 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4318
              ->  Limit  (cost=47.20..16279.57 rows=51 width=1259) (actual time=0.483..5.107 rows=51 loops=1)
                    Buffers: shared hit=4318
                    ->  Merge Append  (cost=47.20..52881.97 rows=166 width=1259) (actual time=0.482..5.096 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4318
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.102..1.131 rows=44 loops=1)
                                Buffers: shared hit=747
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.102..1.123 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=747
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.096..1.098 rows=44 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=747
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.018..0.024 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.018..0.023 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.015..0.018 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.016..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.016..0.016 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.014..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.084..0.089 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.083..0.088 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.056..0.080 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.054..0.077 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.043 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..49711.22 rows=51 width=1259) (actual time=0.259..3.810 rows=7 loops=1)
                                Buffers: shared hit=3535
                                ->  Result  (cost=1.14..393783.00 rows=404 width=1259) (actual time=0.259..3.807 rows=7 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3535
                                      ->  Nested Loop Anti Join  (cost=1.14..393783.00 rows=404 width=1251) (actual time=0.257..3.801 rows=7 loops=1)
                                            Buffers: shared hit=3535
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.77 rows=404 width=1251) (actual time=0.249..3.726 rows=7 loops=1)
                                                  Buffers: shared hit=3521
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.006..1.805 rows=853 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 96
                                                        Buffers: shared hit=961
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=853)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2560
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=7)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=14
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=7)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=14
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.720 ms
Execution Time: 5.923 ms
```

#### district (101 units, 3379 tickets) C2

```
Nested Loop Left Join  (cost=1909.71..2886.96 rows=1 width=1301) (actual time=9.997..11.700 rows=51 loops=1)
  Buffers: shared hit=2008
  ->  Nested Loop  (cost=1909.41..2878.62 rows=1 width=1266) (actual time=9.967..11.482 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1855
        ->  Subquery Scan on x  (cost=1909.13..2870.31 rows=1 width=1259) (actual time=9.934..11.321 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1702
              ->  Limit  (cost=1909.13..2869.16 rows=51 width=1259) (actual time=9.911..11.252 rows=51 loops=1)
                    Buffers: shared hit=1702
                    ->  Merge Append  (cost=1909.13..5033.94 rows=166 width=1259) (actual time=9.909..11.241 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1702
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.107..1.392 rows=44 loops=1)
                                Buffers: shared hit=747
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.107..1.386 rows=44 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=747
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.103..1.363 rows=44 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 693
                                            Buffers: shared hit=747
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.025 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.019..0.024 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.016..0.020 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.111..0.117 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.110..0.116 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.063..0.092 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.087 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.047 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.039 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1863.07..1863.19 rows=51 width=1259) (actual time=9.645..9.658 rows=7 loops=1)
                                Buffers: shared hit=919
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.52 rows=105 width=16) (actual time=0.024..0.051 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.52 rows=105 width=16) (actual time=0.022..0.036 rows=101 loops=1)
                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=1852.55..1852.86 rows=123 width=1259) (actual time=9.642..9.653 rows=7 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=919
                                      ->  Result  (cost=21.92..1848.45 rows=123 width=1259) (actual time=0.119..7.342 rows=3195 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=919
                                            ->  Nested Loop Anti Join  (cost=21.92..1848.45 rows=123 width=1251) (actual time=0.117..6.439 rows=3195 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 9585
                                                  Buffers: shared hit=919
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1810.60 rows=123 width=1251) (actual time=0.088..2.880 rows=3195 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Rows Removed by Filter: 184
                                                        Buffers: shared hit=906
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=3195)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.018..0.025 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.016..0.022 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.012..0.014 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.889 ms
Execution Time: 11.924 ms
```

#### district (101 units, 3379 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=0.980..0.988 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.038..0.898 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.037..0.800 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.036..0.671 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.036..0.579 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.035..0.475 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.035..0.392 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.027..0.268 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.543 ms
Execution Time: 1.131 ms
```

#### district thin (101 units, 600 tickets) C1

```
Nested Loop Left Join  (cost=47.78..16450.04 rows=1 width=1301) (actual time=4.562..6.290 rows=51 loops=1)
  Buffers: shared hit=4529
  ->  Nested Loop  (cost=47.49..16441.70 rows=1 width=1266) (actual time=4.542..6.049 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4376
        ->  Subquery Scan on x  (cost=47.20..16433.39 rows=1 width=1259) (actual time=4.531..5.881 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4223
              ->  Limit  (cost=47.20..16432.24 rows=51 width=1259) (actual time=4.527..5.821 rows=51 loops=1)
                    Buffers: shared hit=4223
                    ->  Merge Append  (cost=47.20..53378.89 rows=166 width=1259) (actual time=4.525..5.808 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4223
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.162..1.406 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.161..1.396 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.152..1.364 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.029..0.035 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.028..0.033 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.025..0.028 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.029..0.031 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.026..0.026 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.182..0.190 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.182..0.189 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.093..0.172 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.090..0.167 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.067..0.072 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.049..0.059 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.038..0.040 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.028..0.028 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..50208.15 rows=51 width=1259) (actual time=4.119..4.123 rows=1 loops=1)
                                Buffers: shared hit=3390
                                ->  Result  (cost=1.14..393781.61 rows=400 width=1259) (actual time=4.118..4.122 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3390
                                      ->  Nested Loop Anti Join  (cost=1.14..393781.61 rows=400 width=1251) (actual time=4.113..4.117 rows=1 loops=1)
                                            Buffers: shared hit=3390
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.73 rows=400 width=1251) (actual time=4.091..4.093 rows=1 loops=1)
                                                  Buffers: shared hit=3388
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.013..1.874 rows=821 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 92
                                                        Buffers: shared hit=924
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=821)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2464
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.018..0.019 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.282 ms
Execution Time: 6.663 ms
```

#### district thin (101 units, 600 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=4.292..5.892 rows=51 loops=1)
  Buffers: shared hit=4529
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=4.273..5.717 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4376
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=4.261..5.594 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4223
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=4.258..5.549 rows=51 loops=1)
                    Buffers: shared hit=4223
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=4.256..5.539 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4223
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.148..1.391 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.148..1.384 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.141..1.358 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.034 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.018..0.031 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.015..0.024 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.085..0.091 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.084..0.090 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.058..0.080 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.055..0.077 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.040..0.043 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.036 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=3.980..3.983 rows=1 loops=1)
                                Buffers: shared hit=3390
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=3.979..3.982 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3390
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=3.977..3.979 rows=1 loops=1)
                                            Buffers: shared hit=3390
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=3.958..3.959 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 912
                                                  Buffers: shared hit=3388
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.002..0.002 rows=0 loops=821)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=2464
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=821)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=2464
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.015..0.016 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.147 ms
Execution Time: 6.088 ms
```

#### district thin (101 units, 600 tickets) W1

```
Nested Loop Left Join  (cost=47.78..16450.04 rows=1 width=1301) (actual time=3.164..4.253 rows=51 loops=1)
  Buffers: shared hit=4529
  ->  Nested Loop  (cost=47.49..16441.70 rows=1 width=1266) (actual time=3.142..4.098 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4376
        ->  Subquery Scan on x  (cost=47.20..16433.39 rows=1 width=1259) (actual time=3.110..3.967 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4223
              ->  Limit  (cost=47.20..16432.24 rows=51 width=1259) (actual time=3.108..3.929 rows=51 loops=1)
                    Buffers: shared hit=4223
                    ->  Merge Append  (cost=47.20..53378.89 rows=166 width=1259) (actual time=3.107..3.921 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4223
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.099..0.890 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.099..0.883 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.095..0.864 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.017..0.022 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.017..0.021 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.015..0.017 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.017..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.087..0.092 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.087..0.091 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.058..0.083 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.056..0.080 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.045 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.038 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..50208.15 rows=51 width=1259) (actual time=2.882..2.885 rows=1 loops=1)
                                Buffers: shared hit=3390
                                ->  Result  (cost=1.14..393781.61 rows=400 width=1259) (actual time=2.882..2.884 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=3390
                                      ->  Nested Loop Anti Join  (cost=1.14..393781.61 rows=400 width=1251) (actual time=2.880..2.882 rows=1 loops=1)
                                            Buffers: shared hit=3390
                                            ->  Nested Loop Semi Join  (cost=0.84..393646.73 rows=400 width=1251) (actual time=2.862..2.863 rows=1 loops=1)
                                                  Buffers: shared hit=3388
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.007..1.410 rows=821 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 92
                                                        Buffers: shared hit=924
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.001..0.001 rows=0 loops=821)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2464
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.014..0.015 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=2
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.704 ms
Execution Time: 4.432 ms
```

#### district thin (101 units, 600 tickets) C2

```
Nested Loop Left Join  (cost=1909.69..2886.94 rows=1 width=1301) (actual time=2.259..3.817 rows=51 loops=1)
  Buffers: shared hit=1881
  ->  Nested Loop  (cost=1909.39..2878.60 rows=1 width=1266) (actual time=2.241..3.641 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1728
        ->  Subquery Scan on x  (cost=1909.11..2870.29 rows=1 width=1259) (actual time=2.226..3.512 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1575
              ->  Limit  (cost=1909.11..2869.14 rows=51 width=1259) (actual time=2.218..3.460 rows=51 loops=1)
                    Buffers: shared hit=1575
                    ->  Merge Append  (cost=1909.11..5033.92 rows=166 width=1259) (actual time=2.216..3.451 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1575
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.118..1.319 rows=50 loops=1)
                                Buffers: shared hit=797
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.117..1.312 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=797
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.114..1.290 rows=50 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=797
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.025 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.019..0.024 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.016..0.020 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.108..0.114 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.108..0.113 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.058..0.085 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.056..0.082 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.046 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.038 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1863.05..1863.17 rows=51 width=1259) (actual time=1.948..1.954 rows=1 loops=1)
                                Buffers: shared hit=742
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.49 rows=104 width=16) (actual time=0.017..0.041 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.49 rows=104 width=16) (actual time=0.015..0.029 rows=101 loops=1)
                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=1852.55..1852.86 rows=123 width=1259) (actual time=1.947..1.952 rows=1 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=742
                                      ->  Result  (cost=21.92..1848.45 rows=123 width=1259) (actual time=0.093..1.578 rows=563 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=742
                                            ->  Nested Loop Anti Join  (cost=21.92..1848.45 rows=123 width=1251) (actual time=0.090..1.447 rows=563 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 1689
                                                  Buffers: shared hit=742
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1810.60 rows=123 width=1251) (actual time=0.068..0.946 rows=563 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Rows Removed by Filter: 37
                                                        Buffers: shared hit=729
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=563)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.013..0.019 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.012..0.017 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.584 ms
Execution Time: 4.026 ms
```

#### district thin (101 units, 600 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.071..1.075 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.040..0.975 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.039..0.868 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.039..0.726 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.038..0.628 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.037..0.515 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.036..0.423 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.028..0.287 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.234 ms
Execution Time: 1.216 ms
```

#### district hot (101 units, 22996 tickets) C1

```
Nested Loop Left Join  (cost=47.78..15374.25 rows=1 width=1301) (actual time=0.470..3.410 rows=51 loops=1)
  Buffers: shared hit=2487
  ->  Nested Loop  (cost=47.49..15365.91 rows=1 width=1266) (actual time=0.446..3.139 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2334
        ->  Subquery Scan on x  (cost=47.20..15357.60 rows=1 width=1259) (actual time=0.428..2.984 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2181
              ->  Limit  (cost=47.20..15356.45 rows=51 width=1259) (actual time=0.425..2.933 rows=51 loops=1)
                    Buffers: shared hit=2181
                    ->  Merge Append  (cost=47.20..49877.31 rows=166 width=1259) (actual time=0.423..2.922 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2181
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.155..0.663 rows=25 loops=1)
                                Buffers: shared hit=428
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.154..0.658 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=428
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.144..0.636 rows=25 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=428
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.026..0.033 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.025..0.031 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.022..0.025 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.026..0.028 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.022..0.022 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.133..0.139 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.133..0.138 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.088..0.120 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.085..0.116 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.063..0.067 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.047..0.055 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.036..0.037 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..46706.57 rows=51 width=1259) (actual time=0.077..2.036 rows=26 loops=1)
                                Buffers: shared hit=1717
                                ->  Result  (cost=1.14..393792.02 rows=430 width=1259) (actual time=0.077..2.030 rows=26 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1717
                                      ->  Nested Loop Anti Join  (cost=1.14..393792.02 rows=430 width=1251) (actual time=0.073..2.017 rows=26 loops=1)
                                            Buffers: shared hit=1717
                                            ->  Nested Loop Semi Join  (cost=0.84..393647.03 rows=430 width=1251) (actual time=0.058..1.854 rows=26 loops=1)
                                                  Buffers: shared hit=1665
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.013..0.839 rows=403 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 45
                                                        Buffers: shared hit=455
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.002..0.002 rows=0 loops=403)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1210
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=26)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=52
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=26)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=52
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.707 ms
Execution Time: 3.722 ms
```

#### district hot (101 units, 22996 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=0.310..3.165 rows=51 loops=1)
  Buffers: shared hit=2487
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=0.294..2.940 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2334
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=0.285..2.769 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2181
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=0.283..2.725 rows=51 loops=1)
                    Buffers: shared hit=2181
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=0.282..2.715 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2181
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.104..0.518 rows=25 loops=1)
                                Buffers: shared hit=428
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.103..0.514 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=428
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.099..0.500 rows=25 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=428
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.018..0.024 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.018..0.022 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.015..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.087..0.092 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.087..0.091 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.059..0.082 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.057..0.078 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.047 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.039 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.053..2.044 rows=26 loops=1)
                                Buffers: shared hit=1717
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.053..2.039 rows=26 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1717
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.051..2.027 rows=26 loops=1)
                                            Buffers: shared hit=1717
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.042..1.904 rows=26 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 422
                                                  Buffers: shared hit=1665
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=403)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=1210
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=403)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=1210
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=26)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=52
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=26)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=52
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.854 ms
Execution Time: 3.380 ms
```

#### district hot (101 units, 22996 tickets) W1

```
Nested Loop Left Join  (cost=47.78..15374.25 rows=1 width=1301) (actual time=0.442..4.131 rows=51 loops=1)
  Buffers: shared hit=2487
  ->  Nested Loop  (cost=47.49..15365.91 rows=1 width=1266) (actual time=0.415..3.794 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2334
        ->  Subquery Scan on x  (cost=47.20..15357.60 rows=1 width=1259) (actual time=0.399..3.596 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2181
              ->  Limit  (cost=47.20..15356.45 rows=51 width=1259) (actual time=0.397..3.528 rows=51 loops=1)
                    Buffers: shared hit=2181
                    ->  Merge Append  (cost=47.20..49877.31 rows=166 width=1259) (actual time=0.395..3.515 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2181
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.141..0.725 rows=25 loops=1)
                                Buffers: shared hit=428
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.140..0.720 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=428
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.127..0.693 rows=25 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=428
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.025..0.034 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.025..0.032 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.020..0.026 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.025..0.027 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.025..0.026 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.022..0.022 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.123..0.129 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.122..0.128 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.086..0.114 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.083..0.110 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.064..0.069 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.046..0.056 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.035..0.037 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..46706.57 rows=51 width=1259) (actual time=0.077..2.569 rows=26 loops=1)
                                Buffers: shared hit=1717
                                ->  Result  (cost=1.14..393792.02 rows=430 width=1259) (actual time=0.076..2.562 rows=26 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1717
                                      ->  Nested Loop Anti Join  (cost=1.14..393792.02 rows=430 width=1251) (actual time=0.073..2.547 rows=26 loops=1)
                                            Buffers: shared hit=1717
                                            ->  Nested Loop Semi Join  (cost=0.84..393647.03 rows=430 width=1251) (actual time=0.058..2.353 rows=26 loops=1)
                                                  Buffers: shared hit=1665
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.012..0.979 rows=403 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 45
                                                        Buffers: shared hit=455
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.003..0.003 rows=0 loops=403)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1210
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.006..0.007 rows=0 loops=26)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=52
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=26)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=52
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.397 ms
Execution Time: 4.418 ms
```

#### district hot (101 units, 22996 tickets) C2

```
Nested Loop Left Join  (cost=1909.85..2887.10 rows=1 width=1301) (actual time=60.775..61.858 rows=51 loops=1)
  Buffers: shared hit=4973
  ->  Nested Loop  (cost=1909.55..2878.76 rows=1 width=1266) (actual time=60.752..61.631 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4820
        ->  Subquery Scan on x  (cost=1909.27..2870.45 rows=1 width=1259) (actual time=60.731..61.494 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4667
              ->  Limit  (cost=1909.27..2869.30 rows=51 width=1259) (actual time=60.715..61.435 rows=51 loops=1)
                    Buffers: shared hit=4667
                    ->  Merge Append  (cost=1909.27..5034.08 rows=166 width=1259) (actual time=60.714..61.426 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4667
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.108..0.768 rows=25 loops=1)
                                Buffers: shared hit=428
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.108..0.764 rows=25 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=428
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.101..0.744 rows=25 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 396
                                            Buffers: shared hit=428
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.019..0.024 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.018..0.022 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.016..0.019 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.020 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.017..0.018 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.089..0.095 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.088..0.094 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.059..0.085 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.057..0.081 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.043..0.047 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.039 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1863.21..1863.33 rows=51 width=1259) (actual time=60.473..60.493 rows=26 loops=1)
                                Buffers: shared hit=4203
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..10.66 rows=112 width=16) (actual time=0.018..0.045 rows=101 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                        Buffers: shared hit=5
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.017..0.031 rows=101 loops=1)
                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=5
                                ->  Sort  (cost=1852.55..1852.86 rows=123 width=1259) (actual time=60.472..60.487 rows=26 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=4203
                                      ->  Result  (cost=21.92..1848.45 rows=123 width=1259) (actual time=0.144..45.899 rows=21559 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=4203
                                            ->  Nested Loop Anti Join  (cost=21.92..1848.45 rows=123 width=1251) (actual time=0.142..39.495 rows=21559 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 64678
                                                  Buffers: shared hit=4203
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1810.60 rows=123 width=1251) (actual time=0.115..16.237 rows=21560 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Rows Removed by Filter: 1436
                                                        Buffers: shared hit=4190
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=21560)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.024 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.015..0.020 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.006..0.007 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.156 ms
Execution Time: 62.170 ms
```

#### district hot (101 units, 22996 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.197..1.203 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.122..1.100 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.121..0.990 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.120..0.846 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.119..0.744 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.116..0.628 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.116..0.534 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.045..0.334 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 2.304 ms
Execution Time: 1.483 ms
```

#### region (1011 units, 37930 tickets) C1

```
Nested Loop Left Join  (cost=47.78..2585.22 rows=1 width=1301) (actual time=0.404..3.159 rows=51 loops=1)
  Buffers: shared hit=1977
  ->  Nested Loop  (cost=47.49..2576.88 rows=1 width=1266) (actual time=0.382..2.864 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1824
        ->  Subquery Scan on x  (cost=47.20..2568.57 rows=1 width=1259) (actual time=0.368..2.652 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1671
              ->  Limit  (cost=47.20..2567.42 rows=51 width=1259) (actual time=0.365..2.590 rows=51 loops=1)
                    Buffers: shared hit=1671
                    ->  Merge Append  (cost=47.20..8250.27 rows=166 width=1259) (actual time=0.364..2.578 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1671
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.133..0.686 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.132..0.681 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.126..0.663 rows=21 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.023..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.119..0.126 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.119..0.124 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.079..0.113 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.077..0.108 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.056..0.061 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.042..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..5079.52 rows=51 width=1259) (actual time=0.062..1.693 rows=31 loops=1)
                                Buffers: shared hit=1286
                                ->  Result  (cost=1.14..395019.94 rows=3967 width=1259) (actual time=0.061..1.685 rows=31 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1286
                                      ->  Nested Loop Anti Join  (cost=1.14..395019.94 rows=3967 width=1251) (actual time=0.059..1.668 rows=31 loops=1)
                                            Buffers: shared hit=1286
                                            ->  Nested Loop Semi Join  (cost=0.84..393682.40 rows=3967 width=1251) (actual time=0.047..1.483 rows=31 loops=1)
                                                  Buffers: shared hit=1224
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.010..0.509 rows=296 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=335
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.003..0.003 rows=0 loops=296)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=889
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=31)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=62
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=31)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=62
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.077 ms
Execution Time: 3.461 ms
```

#### region (1011 units, 37930 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=0.467..3.423 rows=51 loops=1)
  Buffers: shared hit=1977
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=0.444..3.133 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1824
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=0.429..2.966 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1671
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=0.427..2.913 rows=51 loops=1)
                    Buffers: shared hit=1671
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=0.425..2.901 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1671
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.164..0.686 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.163..0.681 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.151..0.660 rows=21 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.029..0.030 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.029..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.023..0.023 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.026..0.028 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.025..0.026 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.022..0.022 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.124..0.134 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.124..0.132 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.085..0.118 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.082..0.113 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.062..0.068 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.048..0.057 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.037..0.040 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.018..0.019 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.012..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.077..1.998 rows=31 loops=1)
                                Buffers: shared hit=1286
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.076..1.991 rows=31 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1286
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.074..1.976 rows=31 loops=1)
                                            Buffers: shared hit=1286
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.059..1.781 rows=31 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 298
                                                  Buffers: shared hit=1224
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.004..0.004 rows=0 loops=296)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=889
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=296)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=889
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=31)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=62
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=31)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=62
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 3.007 ms
Execution Time: 3.740 ms
```

#### region (1011 units, 37930 tickets) W1

```
Nested Loop Left Join  (cost=47.78..2585.22 rows=1 width=1301) (actual time=0.630..6.016 rows=51 loops=1)
  Buffers: shared hit=1977
  ->  Nested Loop  (cost=47.49..2576.88 rows=1 width=1266) (actual time=0.601..5.565 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1824
        ->  Subquery Scan on x  (cost=47.20..2568.57 rows=1 width=1259) (actual time=0.586..5.182 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1671
              ->  Limit  (cost=47.20..2567.42 rows=51 width=1259) (actual time=0.582..5.066 rows=51 loops=1)
                    Buffers: shared hit=1671
                    ->  Merge Append  (cost=47.20..8250.27 rows=166 width=1259) (actual time=0.579..5.047 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1671
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.236..0.944 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.235..0.936 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.196..0.880 rows=21 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.036..0.038 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.035..0.036 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.026..0.027 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.025..0.026 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.021..0.021 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.195..0.201 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.194..0.200 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.150..0.184 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.146..0.178 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.115..0.120 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.073..0.106 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.016..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..5079.52 rows=51 width=1259) (actual time=0.080..3.793 rows=31 loops=1)
                                Buffers: shared hit=1286
                                ->  Result  (cost=1.14..395019.94 rows=3967 width=1259) (actual time=0.079..3.782 rows=31 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=1286
                                      ->  Nested Loop Anti Join  (cost=1.14..395019.94 rows=3967 width=1251) (actual time=0.072..3.755 rows=31 loops=1)
                                            Buffers: shared hit=1286
                                            ->  Nested Loop Semi Join  (cost=0.84..393682.40 rows=3967 width=1251) (actual time=0.056..3.334 rows=31 loops=1)
                                                  Buffers: shared hit=1224
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.015..1.010 rows=296 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=335
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.007..0.007 rows=0 loops=296)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=889
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=31)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=62
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=31)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=62
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.004..0.004 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.669 ms
Execution Time: 6.670 ms
```

#### region (1011 units, 37930 tickets) C2

```
Nested Loop Left Join  (cost=1968.25..2945.50 rows=1 width=1301) (actual time=100.758..101.791 rows=51 loops=1)
  Buffers: shared hit=10468
  ->  Nested Loop  (cost=1967.95..2937.16 rows=1 width=1266) (actual time=100.735..101.566 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=10315
        ->  Subquery Scan on x  (cost=1967.67..2928.85 rows=1 width=1259) (actual time=100.711..101.414 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=10162
              ->  Limit  (cost=1967.67..2927.70 rows=51 width=1259) (actual time=100.695..101.326 rows=51 loops=1)
                    Buffers: shared hit=10162
                    ->  Merge Append  (cost=1967.67..5092.48 rows=166 width=1259) (actual time=100.693..101.295 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=10162
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.109..0.654 rows=21 loops=1)
                                Buffers: shared hit=350
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.109..0.650 rows=21 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=350
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.105..0.636 rows=21 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 323
                                            Buffers: shared hit=350
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.018..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.015..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.020 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.016..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.094..0.100 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.094..0.099 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.063..0.089 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.060..0.084 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.048 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.039 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1921.61..1921.73 rows=51 width=1259) (actual time=100.446..100.468 rows=31 loops=1)
                                Buffers: shared hit=9777
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..69.06 rows=1032 width=16) (actual time=0.019..0.340 rows=1011 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                        Buffers: shared hit=15
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..69.06 rows=1032 width=16) (actual time=0.018..0.212 rows=1011 loops=1)
                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=15
                                ->  Sort  (cost=1852.55..1852.86 rows=123 width=1259) (actual time=100.443..100.459 rows=31 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=9777
                                      ->  Result  (cost=21.92..1848.45 rows=123 width=1259) (actual time=0.531..76.981 rows=35610 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=9777
                                            ->  Nested Loop Anti Join  (cost=21.92..1848.45 rows=123 width=1251) (actual time=0.529..67.158 rows=35610 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 106830
                                                  Buffers: shared hit=9777
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1810.60 rows=123 width=1251) (actual time=0.498..30.950 rows=35610 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Rows Removed by Filter: 2320
                                                        Buffers: shared hit=9764
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=35610)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.020..0.026 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.017..0.022 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.969 ms
Execution Time: 102.021 ms
```

#### region (1011 units, 37930 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.317..1.330 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.087..1.216 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.086..1.096 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.085..0.940 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.084..0.824 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.082..0.658 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.082..0.555 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.062..0.397 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.885 ms
Execution Time: 1.580 ms
```

#### root (10111 units, 380048 tickets) C1

```
Nested Loop Left Join  (cost=47.78..1189.61 rows=1 width=1301) (actual time=0.493..1.507 rows=51 loops=1)
  Buffers: shared hit=713
  ->  Nested Loop  (cost=47.49..1181.27 rows=1 width=1266) (actual time=0.474..1.285 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=560
        ->  Subquery Scan on x  (cost=47.20..1172.96 rows=1 width=1259) (actual time=0.459..1.155 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=407
              ->  Limit  (cost=47.20..1171.81 rows=51 width=1259) (actual time=0.456..1.112 rows=51 loops=1)
                    Buffers: shared hit=407
                    ->  Merge Append  (cost=47.20..3707.70 rows=166 width=1259) (actual time=0.453..1.101 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=407
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.165..0.225 rows=2 loops=1)
                                Buffers: shared hit=66
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.163..0.223 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=66
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.154..0.212 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=66
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.028..0.029 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.028..0.028 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.023..0.023 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.027..0.028 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.024..0.024 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.170..0.174 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.169..0.172 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.113..0.153 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.108..0.148 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.081..0.085 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.057..0.069 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.044..0.045 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.025..0.025 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.018..0.018 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..536.95 rows=51 width=1259) (actual time=0.058..0.626 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=1.14..407094.63 rows=38748 width=1259) (actual time=0.058..0.618 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=1.14..407094.63 rows=38748 width=1251) (actual time=0.055..0.599 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Nested Loop Semi Join  (cost=0.84..394030.21 rows=38748 width=1251) (actual time=0.043..0.370 rows=50 loops=1)
                                                  Buffers: shared hit=206
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.015..0.085 rows=50 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 2
                                                        Buffers: shared hit=55
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.005..0.005 rows=1 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.940 ms
Execution Time: 1.769 ms
```

#### root (10111 units, 380048 tickets) W0

```
Nested Loop Left Join  (cost=47.37..1500.14 rows=1 width=1301) (actual time=0.461..1.766 rows=51 loops=1)
  Buffers: shared hit=713
  ->  Nested Loop  (cost=47.07..1491.81 rows=1 width=1266) (actual time=0.437..1.456 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=560
        ->  Subquery Scan on x  (cost=46.79..1483.49 rows=1 width=1259) (actual time=0.420..1.292 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=407
              ->  Limit  (cost=46.79..1482.35 rows=51 width=1259) (actual time=0.416..1.239 rows=51 loops=1)
                    Buffers: shared hit=407
                    ->  Merge Append  (cost=46.79..4719.39 rows=166 width=1259) (actual time=0.414..1.227 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=407
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.147..0.204 rows=2 loops=1)
                                Buffers: shared hit=66
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.146..0.202 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=66
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.137..0.191 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=66
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.026..0.027 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.025..0.026 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.021..0.022 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.024..0.025 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.023..0.024 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.021..0.021 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.147..0.151 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.146..0.150 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.096..0.133 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.094..0.129 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.068..0.071 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.043..0.054 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.034..0.035 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1548.65 rows=51 width=1259) (actual time=0.065..0.799 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=0.72..1250085.61 rows=41187 width=1259) (actual time=0.064..0.788 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=0.72..1250085.61 rows=41187 width=1251) (actual time=0.060..0.765 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1235800.86 rows=41187 width=1251) (actual time=0.046..0.502 rows=50 loops=1)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id I… (519 chars)
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=206
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.007..0.007 rows=1 loops=50)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          Buffers: shared hit=151
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.005..0.005 rows=1 loops=50)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.681 ms
Execution Time: 2.057 ms
```

#### root (10111 units, 380048 tickets) W1

```
Nested Loop Left Join  (cost=47.78..1189.61 rows=1 width=1301) (actual time=0.732..2.628 rows=51 loops=1)
  Buffers: shared hit=713
  ->  Nested Loop  (cost=47.49..1181.27 rows=1 width=1266) (actual time=0.704..2.318 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=560
        ->  Subquery Scan on x  (cost=47.20..1172.96 rows=1 width=1259) (actual time=0.683..1.912 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=407
              ->  Limit  (cost=47.20..1171.81 rows=51 width=1259) (actual time=0.679..1.850 rows=51 loops=1)
                    Buffers: shared hit=407
                    ->  Merge Append  (cost=47.20..3707.70 rows=166 width=1259) (actual time=0.678..1.837 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=407
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.152..0.464 rows=2 loops=1)
                                Buffers: shared hit=66
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.152..0.461 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=66
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.147..0.452 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=66
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.022..0.024 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.022..0.023 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.019..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.019..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.322..0.330 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.321..0.328 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.268..0.312 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.266..0.307 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.228..0.233 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.205..0.218 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.180..0.182 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.017 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.163..0.163 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.021..0.021 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=1.14..536.95 rows=51 width=1259) (actual time=0.158..0.972 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=1.14..407094.63 rows=38748 width=1259) (actual time=0.156..0.962 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=1.14..407094.63 rows=38748 width=1251) (actual time=0.146..0.932 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Nested Loop Semi Join  (cost=0.84..394030.21 rows=38748 width=1251) (actual time=0.119..0.620 rows=50 loops=1)
                                                  Buffers: shared hit=206
                                                  ->  Index Scan using records_list_updated on records r_4  (cost=0.42..355316.54 rows=82373 width=1267) (actual time=0.067..0.221 rows=50 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                                        Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_grou… (509 chars)
                                                        Rows Removed by Filter: 2
                                                        Buffers: shared hit=55
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..0.47 rows=1 width=16) (actual time=0.007..0.007 rows=1 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.174 ms
Execution Time: 2.858 ms
```

#### root (10111 units, 380048 tickets) C2

```
Nested Loop Left Join  (cost=2569.23..3546.48 rows=1 width=1301) (actual time=904.244..904.626 rows=51 loops=1)
  Buffers: shared hit=101573
  ->  Nested Loop  (cost=2568.93..3538.14 rows=1 width=1266) (actual time=904.221..904.445 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=101420
        ->  Subquery Scan on x  (cost=2568.65..3529.83 rows=1 width=1259) (actual time=904.199..904.332 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=101267
              ->  Limit  (cost=2568.65..3528.68 rows=51 width=1259) (actual time=904.186..904.289 rows=51 loops=1)
                    Buffers: shared hit=101267
                    ->  Merge Append  (cost=2568.65..5693.46 rows=166 width=1259) (actual time=904.185..904.281 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=101267
                          ->  Limit  (cost=0.42..2919.26 rows=51 width=1259) (actual time=0.101..0.153 rows=2 loops=1)
                                Buffers: shared hit=66
                                ->  Result  (cost=0.42..354324.45 rows=6191 width=1259) (actual time=0.100..0.152 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=66
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..354324.45 rows=6191 width=1251) (actual time=0.096..0.146 rows=2 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=66
                          ->  Limit  (cost=0.55..160.01 rows=51 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..400.76 rows=128 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=1251) (actual time=0.015..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.018..0.019 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.17 rows=12 width=1259) (actual time=0.017..0.018 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=1251) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.102..0.108 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1259) (actual time=0.102..0.107 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1259) (actual time=0.058..0.084 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.056..0.080 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.041..0.044 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.037 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1251) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=2522.59..2522.71 rows=51 width=1259) (actual time=903.940..903.958 rows=50 loops=1)
                                Buffers: shared hit=101166
                                InitPlan 1 (returns $0)
                                  ->  Result  (cost=0.41..670.03 rows=10081 width=16) (actual time=0.020..2.611 rows=10111 loops=1)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                        Buffers: shared hit=110
                                        ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..670.03 rows=10081 width=16) (actual time=0.018..1.459 rows=10111 loops=1)
                                              Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                              Heap Fetches: 0
                                              Buffers: shared hit=110
                                ->  Sort  (cost=1852.55..1852.86 rows=123 width=1259) (actual time=903.939..903.951 rows=50 loops=1)
                                      Sort Key: r_4.updated_at DESC, r_4.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=101166
                                      ->  Result  (cost=21.92..1848.45 rows=123 width=1259) (actual time=4.036..694.354 rows=356645 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                            Buffers: shared hit=101166
                                            ->  Nested Loop Anti Join  (cost=21.92..1848.45 rows=123 width=1251) (actual time=4.033..607.761 rows=356645 loops=1)
                                                  Join Filter: (acl_share_1.record_id = r_4.id)
                                                  Rows Removed by Join Filter: 1069938
                                                  Buffers: shared hit=101166
                                                  ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1810.60 rows=123 width=1251) (actual time=3.967..286.197 rows=356648 loops=1)
                                                        Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = ANY ($0)))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853… (553 chars)
                                                        Rows Removed by Filter: 23400
                                                        Buffers: shared hit=101153
                                                  ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (actual time=0.000..0.000 rows=3 loops=356648)
                                                        Buffers: shared hit=13
                                                        ->  Result  (cost=21.49..35.99 rows=1 width=32) (actual time=0.047..0.058 rows=3 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (actual time=0.038..0.047 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{… (461 chars)
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2… (402 chars)
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9… (339 chars)
                                                                                Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.927 ms
Execution Time: 904.851 ms
```

#### root (10111 units, 380048 tickets) W0 count

```
Aggregate  (cost=13628.07..13628.08 rows=1 width=8) (actual time=1.358..1.364 rows=1 loops=1)
  Buffers: shared hit=597
  ->  Limit  (cost=0.55..13615.56 rows=1001 width=4) (actual time=0.046..1.245 rows=1001 loops=1)
        Buffers: shared hit=597
        ->  Result  (cost=0.55..29148.37 rows=2143 width=4) (actual time=0.045..1.116 rows=1001 loops=1)
              Buffers: shared hit=597
              ->  Append  (cost=0.55..29126.94 rows=2143 width=0) (actual time=0.044..0.877 rows=1001 loops=1)
                    Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..82.42 rows=1001 width=0) (actual time=0.043..0.758 rows=1001 loops=1)
                          Buffers: shared hit=597
                          ->  Limit  (cost=0.55..72.41 rows=1001 width=4) (actual time=0.042..0.619 rows=1001 loops=1)
                                Buffers: shared hit=597
                                ->  Result  (cost=0.55..444.99 rows=6191 width=4) (actual time=0.042..0.509 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      Buffers: shared hit=597
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..444.99 rows=6191 width=0) (actual time=0.033..0.344 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,9… (422 chars)
                                            Heap Fetches: 0
                                            Buffers: shared hit=597
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..402.04 rows=128 width=0) (never executed)
                          ->  Limit  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                ->  Result  (cost=0.55..400.76 rows=128 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..400.76 rows=128 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..44.29 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.55..44.17 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.17 rows=12 width=0) (never executed)
                                            Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3adaa8c69168'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a… (374 chars)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b… (449 chars)
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e74… (390 chars)
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a… (327 chars)
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (547 chars)
                    ->  Subquery Scan on "*SELECT* 5"  (cost=21.92..28542.99 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=21.92..28532.98 rows=1001 width=4) (never executed)
                                ->  Result  (cost=21.92..1173133.75 rows=41187 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                      ->  Nested Loop Anti Join  (cost=21.92..1173133.75 rows=41187 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=0.42..1172479.94 rows=41187 width=32) (never executed)
                                                  Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fc6a-5b93-7080-853f-3ada… (563 chars)
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=21.49..36.00 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=21.49..35.99 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a34… (455 chars)
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d3… (396 chars)
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05… (333 chars)
Planning Time: 1.579 ms
Execution Time: 1.520 ms
```

#### area (10 units, 220 tickets) probe K=1000

```
Aggregate  (cost=22.56..22.57 rows=1 width=8) (actual time=0.350..0.352 rows=1 loops=1)
  Buffers: shared hit=37
  ->  Limit  (cost=4.92..22.23 rows=26 width=4) (actual time=0.118..0.318 rows=220 loops=1)
        Buffers: shared hit=37
        ->  Result  (cost=4.92..22.23 rows=26 width=4) (actual time=0.116..0.279 rows=220 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=37
              ->  Nested Loop  (cost=4.92..22.23 rows=26 width=0) (actual time=0.103..0.214 rows=220 loops=1)
                    Buffers: shared hit=37
                    ->  HashAggregate  (cost=4.48..4.51 rows=3 width=16) (actual time=0.050..0.054 rows=10 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.48 rows=3 width=16) (actual time=0.035..0.039 rows=10 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{d9f2a5d9-1b46-33a5-d056-ff2295902adc}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=4
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=28 width=16) (actual time=0.008..0.011 rows=22 loops=10)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=33
Planning Time: 0.545 ms
Execution Time: 0.423 ms
```

#### district (101 units, 3379 tickets) probe K=1000

```
Aggregate  (cost=595.13..595.14 rows=1 width=8) (actual time=1.343..1.346 rows=1 loops=1)
  Buffers: shared hit=114
  ->  Limit  (cost=11.21..583.60 rows=922 width=4) (actual time=0.152..1.207 rows=1001 loops=1)
        Buffers: shared hit=114
        ->  Result  (cost=11.21..583.60 rows=922 width=4) (actual time=0.150..0.965 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=114
              ->  Nested Loop  (cost=11.21..583.60 rows=922 width=0) (actual time=0.137..0.714 rows=1001 loops=1)
                    Buffers: shared hit=114
                    ->  HashAggregate  (cost=10.78..11.83 rows=105 width=16) (actual time=0.118..0.129 rows=34 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.52 rows=105 width=16) (actual time=0.042..0.066 rows=101 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{5f75d86e-bf36-1555-dd22-6537fe2338f2}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.17 rows=28 width=16) (actual time=0.006..0.011 rows=29 loops=34)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=109
Planning Time: 0.538 ms
Execution Time: 1.422 ms
```

#### district thin (101 units, 600 tickets) probe K=1000

```
Aggregate  (cost=589.08..589.09 rows=1 width=8) (actual time=0.603..0.604 rows=1 loops=1)
  Buffers: shared hit=341
  ->  Limit  (cost=11.19..577.67 rows=913 width=4) (actual time=0.082..0.567 rows=600 loops=1)
        Buffers: shared hit=341
        ->  Result  (cost=11.19..577.67 rows=913 width=4) (actual time=0.081..0.522 rows=600 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=341
              ->  Nested Loop  (cost=11.19..577.67 rows=913 width=0) (actual time=0.074..0.453 rows=600 loops=1)
                    Buffers: shared hit=341
                    ->  HashAggregate  (cost=10.75..11.79 rows=104 width=16) (actual time=0.057..0.069 rows=101 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.49 rows=104 width=16) (actual time=0.021..0.034 rows=101 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{cc244a0a-84bc-97a1-4d8d-e7bbd1a953a2}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.16 rows=28 width=16) (actual time=0.003..0.003 rows=6 loops=101)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=336
Planning Time: 0.251 ms
Execution Time: 0.646 ms
```

#### district hot (101 units, 22996 tickets) probe K=1000

```
Aggregate  (cost=629.43..629.44 rows=1 width=8) (actual time=0.587..0.589 rows=1 loops=1)
  Buffers: shared hit=137
  ->  Limit  (cost=11.37..617.15 rows=983 width=4) (actual time=0.094..0.534 rows=1001 loops=1)
        Buffers: shared hit=137
        ->  Result  (cost=11.37..617.15 rows=983 width=4) (actual time=0.093..0.467 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=137
              ->  Nested Loop  (cost=11.37..617.15 rows=983 width=0) (actual time=0.086..0.368 rows=1001 loops=1)
                    Buffers: shared hit=137
                    ->  HashAggregate  (cost=10.94..12.05 rows=112 width=16) (actual time=0.061..0.066 rows=41 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..10.66 rows=112 width=16) (actual time=0.030..0.040 rows=101 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{2674e79d-66b4-6ca7-6ff1-e1468f780293}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.12 rows=28 width=16) (actual time=0.003..0.005 rows=24 loops=41)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=132
Planning Time: 0.254 ms
Execution Time: 0.632 ms
```

#### region (1011 units, 37930 tickets) probe K=1000

```
Aggregate  (cost=383.81..383.82 rows=1 width=8) (actual time=3.844..3.847 rows=1 loops=1)
  Buffers: shared hit=120
  ->  Limit  (cost=121.27..371.30 rows=1001 width=4) (actual time=0.550..3.737 rows=1001 loops=1)
        Buffers: shared hit=120
        ->  Result  (cost=121.27..2384.24 rows=9060 width=4) (actual time=0.549..3.602 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=120
              ->  Merge Semi Join  (cost=121.27..2384.24 rows=9060 width=0) (actual time=0.541..3.407 rows=1001 loops=1)
                    Merge Cond: (r.org_unit_id = acl_ou.descendant_id)
                    Buffers: shared hit=120
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4037.03 rows=188149 width=16) (actual time=0.024..1.632 rows=12028 loops=1)
                          Index Cond: (tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
                          Heap Fetches: 0
                          Buffers: shared hit=105
                    ->  Sort  (cost=120.71..123.29 rows=1032 width=16) (actual time=0.208..0.213 rows=38 loops=1)
                          Sort Key: acl_ou.descendant_id
                          Sort Method: quicksort  Memory: 48kB
                          Buffers: shared hit=15
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..69.06 rows=1032 width=16) (actual time=0.014..0.134 rows=1011 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=15
Planning Time: 0.233 ms
Execution Time: 3.890 ms
```

#### root (10111 units, 380048 tickets) probe K=1000

```
Aggregate  (cost=872.05..872.06 rows=1 width=8) (actual time=6.058..6.060 rows=1 loops=1)
  Buffers: shared hit=233
  ->  Limit  (cost=695.67..859.54 rows=1001 width=4) (actual time=5.255..5.961 rows=1001 loops=1)
        Buffers: shared hit=233
        ->  Result  (cost=695.67..15184.43 rows=88505 width=4) (actual time=5.254..5.839 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid)
              Buffers: shared hit=233
              ->  Nested Loop  (cost=695.67..15184.43 rows=88505 width=0) (actual time=5.245..5.660 rows=1001 loops=1)
                    Buffers: shared hit=233
                    ->  HashAggregate  (cost=695.24..763.22 rows=6798 width=16) (actual time=5.198..5.209 rows=39 loops=1)
                          Group Key: acl_ou.descendant_id
                          Batches: 1  Memory Usage: 913kB
                          Buffers: shared hit=110
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..670.03 rows=10081 width=16) (actual time=0.027..1.777 rows=10111 loops=1)
                                Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                Heap Fetches: 0
                                Buffers: shared hit=110
                    ->  Index Only Scan using records_org_unit on records r  (cost=0.42..1.84 rows=28 width=16) (actual time=0.005..0.007 rows=26 loops=39)
                          Index Cond: ((tenant_id = 'e2e74aab-5c03-4232-a19a-d8abde1496c1'::uuid) AND (org_unit_id = acl_ou.descendant_id))
                          Heap Fetches: 0
                          Buffers: shared hit=123
Planning Time: 0.308 ms
Execution Time: 6.127 ms
```

## 2026-11-21T05:30:53Z phase=server secs=15

`PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit`. nava_app role settings: `plan_cache_mode=force_custom_plan`. Foreign containers at start: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.

Large tenant tickets: 400000.

### Server-side paired pass: planning + execution ms per request (count + page [+ probe]), EXPLAIN (ANALYZE, TIMING OFF), 300 pairs per variant, baseline and variant alternate

| Set | variant | pairs | base p50 | base p95 | var p50 | var p95 | Δp50 | **Δp95** | median per-pair Δ | p95 per-pair Δ |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| area (10 units, 220 tickets) | C2 | 300 | 5.90 | 8.80 | 7.75 | 11.25 | +1.85 | **+2.45** | +1.62 | +4.69 |
| area (10 units, 220 tickets) | M2 K=500 | 300 | 5.54 | 8.29 | 7.87 | 11.59 | +2.33 | **+3.30** | +2.32 | +5.60 |
| area (10 units, 220 tickets) | M2 K=1000 | 300 | 5.86 | 8.97 | 8.23 | 12.30 | +2.37 | **+3.33** | +2.29 | +5.97 |
| district (101 units, 3379 tickets) | C2 | 300 | 5.70 | 8.64 | 13.16 | 18.15 | +7.46 | **+9.51** | +7.09 | +12.45 |
| district (101 units, 3379 tickets) | M2 K=500 | 300 | 5.92 | 8.64 | 11.57 | 18.13 | +5.65 | **+9.48** | +5.69 | +11.16 |
| district (101 units, 3379 tickets) | M2 K=1000 | 300 | 5.65 | 8.11 | 11.38 | 16.64 | +5.73 | **+8.53** | +5.70 | +10.52 |
| district thin (101 units, 600 tickets) | C2 | 300 | 5.87 | 8.34 | 8.80 | 12.65 | +2.93 | **+4.31** | +2.94 | +6.49 |
| district thin (101 units, 600 tickets) | M2 K=500 | 300 | 6.23 | 8.59 | 13.99 | 28.16 | +7.76 | **+19.57** | +7.81 | +21.89 |
| district thin (101 units, 600 tickets) | M2 K=1000 | 300 | 6.39 | 11.07 | 11.16 | 22.46 | +4.77 | **+11.38** | +4.75 | +10.91 |
| district hot (101 units, 22996 tickets) | C2 | 300 | 5.99 | 8.73 | 42.99 | 59.62 | +37.01 | **+50.89** | +37.39 | +52.82 |
| district hot (101 units, 22996 tickets) | M2 K=500 | 300 | 6.20 | 8.61 | 9.04 | 13.17 | +2.84 | **+4.56** | +2.72 | +7.28 |
| district hot (101 units, 22996 tickets) | M2 K=1000 | 300 | 5.75 | 8.53 | 8.44 | 12.87 | +2.69 | **+4.34** | +2.72 | +6.99 |
| region (1011 units, 37930 tickets) | C2 | 300 | 6.38 | 8.95 | 83.44 | 131.82 | +77.06 | **+122.87** | +77.13 | +125.33 |
| region (1011 units, 37930 tickets) | M2 K=500 | 300 | 5.88 | 8.18 | 9.36 | 12.75 | +3.47 | **+4.57** | +3.27 | +6.89 |
| region (1011 units, 37930 tickets) | M2 K=1000 | 300 | 5.88 | 8.70 | 9.86 | 14.18 | +3.98 | **+5.49** | +3.88 | +8.28 |
| root (10111 units, 380048 tickets) | C2 | 300 | 6.26 | 9.82 | 765.62 | 1194.65 | +759.36 | **+1184.84** | +760.07 | +1188.74 |
| root (10111 units, 380048 tickets) | M2 K=500 | 300 | 4.96 | 7.68 | 6.68 | 10.11 | +1.72 | **+2.43** | +1.69 | +4.21 |
| root (10111 units, 380048 tickets) | M2 K=1000 | 300 | 5.05 | 8.26 | 9.90 | 15.00 | +4.86 | **+6.74** | +4.43 | +9.23 |

Foreign containers at end: `nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1`.
