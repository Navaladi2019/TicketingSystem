# SP-1f (TICK-1092) region rows

## Sets (as each row ran)

| Mode | Set | Type | \|S\| | m | N̂_e | K* | Shape |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A2 | area (\|S\| 10) | ticket | 10 | 220 | 400000 | 3613 | C3 (m <= K*) |
| A2 | district (\|S\| 101) | ticket | 101 | 3379 | 400000 | 3613 | C3 (m <= K*) |
| A2 | region (\|S\| 1,011) | ticket | 1011 | 37930 | 400000 | 3613 | W0 (m > K*) |
| A2 | root (\|S\| 10,111) | ticket | 10111 | 380048 | 400000 | 3613 | W0 (m > K*) |
| A2 | district hot | ticket | 101 | 22996 | 400000 | 3613 | W0 (m > K*) |
| A2 | district thin | ticket | 101 | 600 | 400000 | 3613 | C3 (m <= K*) |
| A2 | uniform m=1001 | ticket | 101 | 1001 | 400000 | 3613 | C3 (m <= K*) |
| A2 | uniform m=3000 | ticket | 101 | 3000 | 400000 | 3613 | C3 (m <= K*) |
| A2 | uniform m=4500 | ticket | 101 | 4500 | 400000 | 3613 | W0 (m > K*) |
| A2 | uniform m=10000 | ticket | 101 | 10000 | 400000 | 3613 | W0 (m > K*) |
| A2 | bench_rare at root | bench_rare | 10111 | 500 | 500 | 1000 | A2-5a rare: W0, no probe |
| A2 | bench_conc, regions 0-8 (\|S\| 9,099) | bench_conc | 9099 | 0 | 20000 | 1000 | C3 (m <= K*) |
| A2 | bench_conc at root (\|S\| 10,111) | bench_conc | 10111 | 20000 | 20000 | 1000 | W0 (m > K*) |
| A2 | sweep bench_conc, regions 0-0 (\|S\| 1011) | bench_conc | 1011 | 0 | 20000 | 1000 | C3 (m <= K*) |
| A2 | sweep bench_conc, regions 0-1 (\|S\| 2022) | bench_conc | 2022 | 0 | 20000 | 1000 | C3 (m <= K*) |
| A2 | sweep bench_conc, regions 0-2 (\|S\| 3033) | bench_conc | 3033 | 0 | 20000 | 1000 | C3 (m <= K*) |
| A2 | sweep bench_conc, regions 0-3 (\|S\| 4044) | bench_conc | 4044 | 0 | 20000 | 1000 | C3 (m <= K*) |
| R | area (\|S\| 10) | ticket | 10 | 220 | 400000 | 3613 | C3 (m <= K*) |
| R | district (\|S\| 101) | ticket | 101 | 3379 | 400000 | 3613 | C3 (m <= K*) |
| R | region (\|S\| 1,011) | ticket | 1011 | 37930 | 400000 | 3613 | W0 (m > K*) |
| R | root (\|S\| 10,111) | ticket | 10111 | 380048 | 400000 | 3613 | W0 (m > K*) |
| R | district hot | ticket | 101 | 22996 | 400000 | 3613 | W0 (m > K*) |
| R | district thin | ticket | 101 | 600 | 400000 | 3613 | C3 (m <= K*) |
| R | uniform m=1001 | ticket | 101 | 1001 | 400000 | 3613 | C3 (m <= K*) |
| R | uniform m=3000 | ticket | 101 | 3000 | 400000 | 3613 | C3 (m <= K*) |
| R | uniform m=4500 | ticket | 101 | 4500 | 400000 | 3613 | W0 (m > K*) |
| R | uniform m=10000 | ticket | 101 | 10000 | 400000 | 3613 | W0 (m > K*) |
| R | bench_rare at root | bench_rare | 10111 | 500 | 500 | 1000 | A2-5a rare: W0, no probe |
| R | bench_conc, regions 0-8 (\|S\| 9,099) | bench_conc | 9099 | 0 | 20000 | 1000 | R (N̂_e <= 8·|S|) |
| R | bench_conc at root (\|S\| 10,111) | bench_conc | 10111 | 20000 | 20000 | 1000 | R (N̂_e <= 8·|S|) |

## A2-6 agreement (ids in order, capped count, against the control)

| Set | Mode | Callers | Result |
| --- | --- | --- | --- |
| sweep bench_conc, regions 0-0 (\|S\| 1011) | R forced | 201 callers | 0 mismatches |
| sweep bench_conc, regions 0-1 (\|S\| 2022) | R forced | 201 callers | 0 mismatches |
| sweep bench_conc, regions 0-2 (\|S\| 3033) | R forced | 201 callers | 0 mismatches |
| sweep bench_conc, regions 0-3 (\|S\| 4044) | R forced | 201 callers | 0 mismatches |
| area (\|S\| 10) | R | 201 callers | 0 mismatches |
| district (\|S\| 101) | R | 201 callers | 0 mismatches |
| region (\|S\| 1,011) | R | 201 callers | 0 mismatches |
| root (\|S\| 10,111) | R | 201 callers | 0 mismatches |
| district hot | R | 201 callers | 0 mismatches |
| district thin | R | 201 callers | 0 mismatches |
| uniform m=1001 | R | 201 callers | 0 mismatches |
| uniform m=3000 | R | 201 callers | 0 mismatches |
| uniform m=4500 | R | 201 callers | 0 mismatches |
| uniform m=10000 | R | 201 callers | 0 mismatches |
| bench_rare at root | R | 201 callers | 0 mismatches |
| bench_conc, regions 0-8 (\|S\| 9,099) | R | 201 callers | 0 mismatches |
| bench_conc at root (\|S\| 10,111) | R | 201 callers | 0 mismatches |

## SP-1f plans (EXPLAIN ANALYZE, BUFFERS)

### n_distinct(records.org_unit_id) = analyzed (pg_stats: 3311)

#### bench_conc, regions 0-8 (|S| 9,099), region-only
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=718.94..718.95 rows=1 width=16) (actual time=3.123..3.137 rows=1 loops=1)
  Buffers: shared hit=118
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=375.17..375.18 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.38..375.08 rows=7 width=4) (never executed)
                ->  Result  (cost=1.38..375.08 rows=7 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.38..375.08 rows=7 width=0) (never executed)
                            ->  Nested Loop  (cost=0.95..343.66 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.09 rows=9 width=16) (never executed)
                                        ->  Result  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                  ->  Result  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4.47 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=343.76..343.77 rows=1 width=8) (actual time=3.118..3.124 rows=1 loops=1)
        Buffers: shared hit=118
        ->  Limit  (cost=0.96..343.67 rows=7 width=4) (actual time=0.197..2.860 rows=2500 loops=1)
              Buffers: shared hit=118
              ->  Result  (cost=0.96..343.67 rows=7 width=4) (actual time=0.195..2.399 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=118
                    ->  Nested Loop  (cost=0.96..343.67 rows=7 width=0) (actual time=0.181..1.831 rows=2500 loops=1)
                          Buffers: shared hit=118
                          ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (actual time=0.114..0.347 rows=3 loops=1)
                                Buffers: shared hit=82
                                ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (actual time=0.017..0.029 rows=3 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.09 rows=9 width=16) (actual time=0.008..0.009 rows=9 loops=1)
                                ->  Result  (cost=0.41..37.64 rows=1 width=32) (actual time=0.101..0.101 rows=0 loops=3)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=82
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..37.64 rows=1 width=32) (actual time=0.083..0.083 rows=0 loops=3)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=82
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.032..0.280 rows=833 loops=3)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=36
Planning Time: 0.929 ms
Execution Time: 3.298 ms
```

count:
```
Aggregate  (cost=10579.57..10579.58 rows=1 width=8) (actual time=60.972..60.991 rows=1 loops=1)
  Buffers: shared hit=19952
  ->  Limit  (cost=0.55..10567.06 rows=1001 width=4) (actual time=60.965..60.983 rows=0 loops=1)
        Buffers: shared hit=19952
        ->  Result  (cost=0.55..10598.73 rows=1004 width=4) (actual time=60.962..60.979 rows=0 loops=1)
              Buffers: shared hit=19952
              ->  Append  (cost=0.55..10588.69 rows=1004 width=0) (actual time=60.961..60.978 rows=0 loops=1)
                    Buffers: shared hit=19952
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.068..0.069 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.066..0.067 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.066..0.066 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.051..0.051 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..8.59 rows=1 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (actual time=0.027..0.028 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (actual time=0.027..0.027 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.029..0.035 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.028..0.034 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.028..0.033 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.026..0.030 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.025..0.029 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.022..0.024 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.020..0.022 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=610.26..10545.92 rows=1001 width=0) (actual time=60.833..60.839 rows=0 loops=1)
                          Buffers: shared hit=19940
                          ->  Limit  (cost=610.26..10535.91 rows=1001 width=4) (actual time=60.833..60.838 rows=0 loops=1)
                                Buffers: shared hit=19940
                                ->  Result  (cost=610.26..23059.49 rows=2264 width=4) (actual time=60.832..60.836 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=19940
                                      ->  Nested Loop Anti Join  (cost=610.26..23059.49 rows=2264 width=0) (actual time=60.829..60.833 rows=0 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=19940
                                            ->  Index Scan using records_org_unit on records r_3  (cost=601.64..23009.44 rows=2264 width=32) (actual time=60.826..60.827 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 20000
                                                  Buffers: shared hit=19940
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.033..5.144 rows=9099 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=127
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.029..3.221 rows=9099 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=127
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 2.814 ms
Execution Time: 61.236 ms
```

page:
```
Nested Loop Left Join  (cost=636.55..1511.65 rows=1 width=1317) (actual time=46.805..46.822 rows=0 loops=1)
  Buffers: shared hit=20327
  ->  Nested Loop  (cost=636.25..1503.31 rows=1 width=1282) (actual time=46.805..46.820 rows=0 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=20327
        ->  Subquery Scan on x  (cost=635.97..1495.00 rows=1 width=1275) (actual time=46.804..46.819 rows=0 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=20327
              ->  Limit  (cost=635.97..1493.85 rows=51 width=1275) (actual time=46.803..46.817 rows=0 loops=1)
                    Buffers: shared hit=20327
                    ->  Merge Append  (cost=635.97..1544.31 rows=54 width=1275) (actual time=46.801..46.815 rows=0 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=20327
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.042..0.044 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.041..0.042 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.033..0.034 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.018..0.019 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.018..0.019 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.016..0.016 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.026..0.031 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.025..0.030 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=4
                                      ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.021..0.025 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.019..0.022 rows=0 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.019..0.021 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.016..0.018 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                          ->  Limit  (cost=610.26..1501.74 rows=51 width=1275) (actual time=46.711..46.716 rows=0 loops=1)
                                Buffers: shared hit=20315
                                ->  Result  (cost=610.26..40184.74 rows=2264 width=1275) (actual time=46.710..46.714 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=20315
                                      ->  Nested Loop Anti Join  (cost=610.26..40184.74 rows=2264 width=1267) (actual time=46.708..46.712 rows=0 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=20315
                                            ->  Index Scan using records_list_updated on records r_3  (cost=601.65..40134.68 rows=2264 width=1267) (actual time=46.706..46.708 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 20000
                                                  Buffers: shared hit=20315
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.024..4.619 rows=9099 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=127
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.021..2.816 rows=9099 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=127
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (never executed)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (never executed)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
Planning Time: 2.815 ms
Execution Time: 47.088 ms
```

#### bench_conc, regions 0-8 (|S| 9,099), ragent-1
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=718.94..718.95 rows=1 width=16) (actual time=1.931..1.936 rows=1 loops=1)
  Buffers: shared hit=118
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=375.17..375.18 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.38..375.08 rows=7 width=4) (never executed)
                ->  Result  (cost=1.38..375.08 rows=7 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.38..375.08 rows=7 width=0) (never executed)
                            ->  Nested Loop  (cost=0.95..343.66 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.09 rows=9 width=16) (never executed)
                                        ->  Result  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                  ->  Result  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4.47 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=343.76..343.77 rows=1 width=8) (actual time=1.929..1.931 rows=1 loops=1)
        Buffers: shared hit=118
        ->  Limit  (cost=0.96..343.67 rows=7 width=4) (actual time=0.120..1.707 rows=2500 loops=1)
              Buffers: shared hit=118
              ->  Result  (cost=0.96..343.67 rows=7 width=4) (actual time=0.119..1.386 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=118
                    ->  Nested Loop  (cost=0.96..343.67 rows=7 width=0) (actual time=0.108..0.984 rows=2500 loops=1)
                          Buffers: shared hit=118
                          ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (actual time=0.099..0.187 rows=3 loops=1)
                                Buffers: shared hit=82
                                ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (actual time=0.014..0.016 rows=3 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.09 rows=9 width=16) (actual time=0.007..0.008 rows=9 loops=1)
                                ->  Result  (cost=0.41..37.64 rows=1 width=32) (actual time=0.056..0.056 rows=0 loops=3)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=82
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..37.64 rows=1 width=32) (actual time=0.052..0.052 rows=0 loops=3)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=82
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.009..0.153 rows=833 loops=3)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=36
Planning Time: 1.458 ms
Execution Time: 2.058 ms
```

count:
```
Aggregate  (cost=8311.89..8311.90 rows=1 width=8) (actual time=1.400..1.409 rows=1 loops=1)
  Buffers: shared hit=29
  ->  Limit  (cost=0.55..8299.38 rows=1001 width=4) (actual time=0.057..1.234 rows=1001 loops=1)
        Buffers: shared hit=29
        ->  Result  (cost=0.55..11051.83 rows=1333 width=4) (actual time=0.056..1.114 rows=1001 loops=1)
              Buffers: shared hit=29
              ->  Append  (cost=0.55..11038.50 rows=1333 width=0) (actual time=0.055..0.949 rows=1001 loops=1)
                    Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..34.12 rows=325 width=0) (actual time=0.055..0.820 rows=1001 loops=1)
                          Buffers: shared hit=29
                          ->  Limit  (cost=0.55..30.88 rows=325 width=4) (actual time=0.054..0.675 rows=1001 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.55..30.88 rows=325 width=4) (actual time=0.053..0.458 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=29
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..30.88 rows=325 width=0) (actual time=0.044..0.285 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..24.67 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.55..24.62 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.55..24.62 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..24.62 rows=5 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..8.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=627.44..10912.28 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=627.44..10902.27 rows=1001 width=4) (never executed)
                                ->  Result  (cost=627.44..23117.09 rows=2191 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop Anti Join  (cost=627.44..23117.09 rows=2191 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=601.64..23040.53 rows=2191 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (never executed)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 1.558 ms
Execution Time: 1.568 ms
```

page:
```
Nested Loop Left Join  (cost=1953.80..2434.70 rows=1 width=1317) (actual time=47.174..47.654 rows=51 loops=1)
  Buffers: shared hit=21814
  ->  Nested Loop  (cost=1953.50..2426.36 rows=1 width=1282) (actual time=47.147..47.454 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=21661
        ->  Subquery Scan on x  (cost=1953.22..2418.05 rows=1 width=1275) (actual time=47.088..47.204 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21457
              ->  Limit  (cost=1953.22..2416.90 rows=51 width=1275) (actual time=47.071..47.142 rows=51 loops=1)
                    Buffers: shared hit=21457
                    ->  Merge Append  (cost=1953.22..2944.23 rows=109 width=1275) (actual time=47.070..47.133 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21457
                          ->  Limit  (cost=1297.94..1298.06 rows=51 width=1275) (actual time=3.637..3.658 rows=50 loops=1)
                                Buffers: shared hit=1104
                                ->  Sort  (cost=1297.94..1298.75 rows=325 width=1275) (actual time=3.636..3.651 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=1104
                                      ->  Result  (cost=27.72..1287.09 rows=325 width=1275) (actual time=0.540..2.553 rows=1236 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=1104
                                            ->  Bitmap Heap Scan on records r  (cost=27.72..1287.09 rows=325 width=1267) (actual time=0.532..2.016 rows=1236 loops=1)
                                                  Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                  Heap Blocks: exact=1069
                                                  Buffers: shared hit=1104
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..27.62 rows=325 width=0) (actual time=0.291..0.291 rows=1236 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                        Buffers: shared hit=35
                          ->  Limit  (cost=0.55..24.62 rows=5 width=1275) (actual time=0.048..0.054 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..24.62 rows=5 width=1275) (actual time=0.046..0.052 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..24.62 rows=5 width=1267) (actual time=0.033..0.037 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.021..0.022 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.021..0.021 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.017..0.017 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.099..0.104 rows=0 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.098..0.103 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.092..0.095 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.089..0.093 rows=0 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.053..0.057 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.039..0.046 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.010..0.010 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=601.94..1558.92 rows=51 width=1275) (actual time=43.257..43.260 rows=0 loops=1)
                                Buffers: shared hit=20315
                                ->  Result  (cost=601.94..41714.30 rows=2191 width=1275) (actual time=43.256..43.258 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=20315
                                      ->  Nested Loop Anti Join  (cost=601.94..41714.30 rows=2191 width=1267) (actual time=43.253..43.255 rows=0 loops=1)
                                            Buffers: shared hit=20315
                                            ->  Index Scan using records_list_updated on records r_4  (cost=601.65..40198.00 rows=2191 width=1267) (actual time=43.251..43.252 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 20000
                                                  Buffers: shared hit=20315
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.021..3.050 rows=9099 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=127
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.020..1.752 rows=9099 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=127
                                            ->  Result  (cost=0.29..0.68 rows=1 width=32) (never executed)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.68 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.095 ms
Execution Time: 47.894 ms
```

#### bench_conc at root (|S| 10,111), region-only
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=49.82..49.83 rows=1 width=16) (actual time=1.826..1.862 rows=1 loops=1)
  Buffers: shared hit=36
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=40.61..40.62 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.28..40.52 rows=7 width=4) (never executed)
                ->  Result  (cost=1.28..40.52 rows=7 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.28..40.52 rows=7 width=0) (never executed)
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                  ->  Result  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4.47 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=1.824..1.857 rows=1 loops=1)
        Buffers: shared hit=36
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.123..1.633 rows=2500 loops=1)
              Buffers: shared hit=36
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.121..1.346 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=36
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.110..0.938 rows=2500 loops=1)
                          Buffers: shared hit=36
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.098..0.130 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.010..0.042 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.006 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.086..0.087 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.084..0.084 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.011..0.466 rows=2500 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=32
Planning Time: 0.923 ms
Execution Time: 1.971 ms
```

count:
```
Aggregate  (cost=10666.54..10666.55 rows=1 width=8) (actual time=11.151..11.169 rows=1 loops=1)
  Buffers: shared hit=1114
  ->  Limit  (cost=0.55..10654.03 rows=1001 width=4) (actual time=7.645..10.808 rows=1001 loops=1)
        Buffers: shared hit=1114
        ->  Result  (cost=0.55..10685.96 rows=1004 width=4) (actual time=7.643..10.656 rows=1001 loops=1)
              Buffers: shared hit=1114
              ->  Append  (cost=0.55..10675.92 rows=1004 width=0) (actual time=7.642..10.463 rows=1001 loops=1)
                    Buffers: shared hit=1114
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.053..0.055 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.052..0.054 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.052..0.053 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.040..0.041 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..8.59 rows=1 width=0) (actual time=0.078..0.079 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (actual time=0.077..0.078 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (actual time=0.077..0.078 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=0) (actual time=0.075..0.075 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.045..0.050 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.045..0.050 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.044..0.048 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.041..0.044 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.040..0.043 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.038..0.040 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.036..0.038 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=697.49..10633.15 rows=1001 width=0) (actual time=7.464..10.121 rows=1001 loops=1)
                          Buffers: shared hit=1102
                          ->  Limit  (cost=697.49..10623.14 rows=1001 width=4) (actual time=7.463..9.944 rows=1001 loops=1)
                                Buffers: shared hit=1102
                                ->  Result  (cost=697.49..23146.72 rows=2264 width=4) (actual time=7.462..9.827 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=1102
                                      ->  Nested Loop Anti Join  (cost=697.49..23146.72 rows=2264 width=0) (actual time=7.458..9.660 rows=1001 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=1102
                                            ->  Index Scan using records_org_unit on records r_3  (cost=688.88..23096.67 rows=2264 width=32) (actual time=7.418..9.299 rows=1001 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Buffers: shared hit=1098
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.023..3.792 rows=10111 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=116
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.021..2.208 rows=10111 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=116
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                  Buffers: shared hit=4
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (actual time=0.033..0.035 rows=0 loops=1)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (actual time=0.021..0.023 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
Planning Time: 2.346 ms
Execution Time: 11.351 ms
```

page:
```
Nested Loop Left Join  (cost=723.78..1598.88 rows=1 width=1317) (actual time=6.795..7.481 rows=51 loops=1)
  Buffers: shared hit=543
  ->  Nested Loop  (cost=723.48..1590.54 rows=1 width=1282) (actual time=6.770..7.236 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=390
        ->  Subquery Scan on x  (cost=723.20..1582.23 rows=1 width=1275) (actual time=6.746..7.027 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=186
              ->  Limit  (cost=723.20..1581.08 rows=51 width=1275) (actual time=6.742..6.978 rows=51 loops=1)
                    Buffers: shared hit=186
                    ->  Merge Append  (cost=723.20..1631.54 rows=54 width=1275) (actual time=6.739..6.966 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=186
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.090..0.092 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.090..0.090 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.073..0.074 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.044..0.045 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.043..0.044 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.039..0.039 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.049..0.054 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.049..0.053 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=4
                                      ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.038..0.043 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.036..0.040 rows=0 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.036..0.038 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.032..0.034 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.028..0.030 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                          ->  Limit  (cost=697.49..1588.97 rows=51 width=1275) (actual time=6.552..6.758 rows=51 loops=1)
                                Buffers: shared hit=174
                                ->  Result  (cost=697.49..40271.97 rows=2264 width=1275) (actual time=6.550..6.751 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=174
                                      ->  Nested Loop Anti Join  (cost=697.49..40271.97 rows=2264 width=1267) (actual time=6.547..6.730 rows=51 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=174
                                            ->  Index Scan using records_list_updated on records r_3  (cost=688.88..40221.91 rows=2264 width=1267) (actual time=6.508..6.663 rows=51 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Buffers: shared hit=170
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.059..3.443 rows=10111 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=116
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.052..2.001 rows=10111 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=116
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (actual time=0.001..0.001 rows=0 loops=51)
                                                  Buffers: shared hit=4
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (actual time=0.030..0.031 rows=0 loops=1)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (actual time=0.020..0.021 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.017..0.018 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.767 ms
Execution Time: 8.044 ms
```

#### bench_conc at root (|S| 10,111), ragent-1
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=49.82..49.83 rows=1 width=16) (actual time=2.730..2.742 rows=1 loops=1)
  Buffers: shared hit=36
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=40.61..40.62 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.28..40.52 rows=7 width=4) (never executed)
                ->  Result  (cost=1.28..40.52 rows=7 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.28..40.52 rows=7 width=0) (never executed)
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                  ->  Result  (cost=0.42..4.47 rows=1 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4.47 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=2.726..2.731 rows=1 loops=1)
        Buffers: shared hit=36
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.247..2.432 rows=2500 loops=1)
              Buffers: shared hit=36
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.244..2.037 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=36
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.120..1.407 rows=2500 loops=1)
                          Buffers: shared hit=36
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.101..0.104 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.034..0.035 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.022..0.022 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.064..0.065 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.059..0.059 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.017..0.791 rows=2500 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=32
Planning Time: 1.021 ms
Execution Time: 2.930 ms
```

count:
```
Aggregate  (cost=8377.39..8377.40 rows=1 width=8) (actual time=1.249..1.257 rows=1 loops=1)
  Buffers: shared hit=29
  ->  Limit  (cost=0.55..8364.88 rows=1001 width=4) (actual time=0.055..1.134 rows=1001 loops=1)
        Buffers: shared hit=29
        ->  Result  (cost=0.55..11139.06 rows=1333 width=4) (actual time=0.053..1.011 rows=1001 loops=1)
              Buffers: shared hit=29
              ->  Append  (cost=0.55..11125.73 rows=1333 width=0) (actual time=0.052..0.851 rows=1001 loops=1)
                    Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..34.12 rows=325 width=0) (actual time=0.052..0.688 rows=1001 loops=1)
                          Buffers: shared hit=29
                          ->  Limit  (cost=0.55..30.88 rows=325 width=4) (actual time=0.051..0.548 rows=1001 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.55..30.88 rows=325 width=4) (actual time=0.051..0.434 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=29
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..30.88 rows=325 width=0) (actual time=0.040..0.265 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..24.67 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.55..24.62 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.55..24.62 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..24.62 rows=5 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..8.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=714.67..10999.51 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=714.67..10989.50 rows=1001 width=4) (never executed)
                                ->  Result  (cost=714.67..23204.32 rows=2191 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop Anti Join  (cost=714.67..23204.32 rows=2191 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=688.88..23127.76 rows=2191 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (never executed)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 1.778 ms
Execution Time: 1.463 ms
```

page:
```
Nested Loop Left Join  (cost=2041.03..2521.93 rows=1 width=1317) (actual time=10.712..11.866 rows=51 loops=1)
  Buffers: shared hit=1766
  ->  Nested Loop  (cost=2040.73..2513.59 rows=1 width=1282) (actual time=10.686..11.566 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=1613
        ->  Subquery Scan on x  (cost=2040.45..2505.28 rows=1 width=1275) (actual time=10.663..11.317 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1409
              ->  Limit  (cost=2040.45..2504.13 rows=51 width=1275) (actual time=10.661..11.260 rows=51 loops=1)
                    Buffers: shared hit=1409
                    ->  Merge Append  (cost=2040.45..3031.46 rows=109 width=1275) (actual time=10.658..11.247 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1409
                          ->  Limit  (cost=1297.94..1298.06 rows=51 width=1275) (actual time=3.960..3.967 rows=3 loops=1)
                                Buffers: shared hit=1104
                                ->  Sort  (cost=1297.94..1298.75 rows=325 width=1275) (actual time=3.958..3.964 rows=3 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=1104
                                      ->  Result  (cost=27.72..1287.09 rows=325 width=1275) (actual time=0.546..2.744 rows=1236 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=1104
                                            ->  Bitmap Heap Scan on records r  (cost=27.72..1287.09 rows=325 width=1267) (actual time=0.539..2.151 rows=1236 loops=1)
                                                  Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                  Heap Blocks: exact=1069
                                                  Buffers: shared hit=1104
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..27.62 rows=325 width=0) (actual time=0.289..0.290 rows=1236 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                        Buffers: shared hit=35
                          ->  Limit  (cost=0.55..24.62 rows=5 width=1275) (actual time=0.055..0.056 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..24.62 rows=5 width=1275) (actual time=0.052..0.053 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..24.62 rows=5 width=1267) (actual time=0.037..0.037 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.020..0.021 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.019..0.020 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.016..0.016 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.108..0.113 rows=0 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.107..0.112 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.099..0.103 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.096..0.100 rows=0 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.057..0.062 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.043..0.050 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.011..0.011 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=689.17..1646.15 rows=51 width=1275) (actual time=6.507..7.062 rows=49 loops=1)
                                Buffers: shared hit=268
                                ->  Result  (cost=689.17..41801.53 rows=2191 width=1275) (actual time=6.506..7.050 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=268
                                      ->  Nested Loop Anti Join  (cost=689.17..41801.53 rows=2191 width=1267) (actual time=6.502..7.026 rows=49 loops=1)
                                            Buffers: shared hit=268
                                            ->  Index Scan using records_list_updated on records r_4  (cost=688.88..40285.23 rows=2191 width=1267) (actual time=6.463..6.640 rows=49 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=170
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.023..3.330 rows=10111 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=116
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.020..1.870 rows=10111 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=116
                                            ->  Result  (cost=0.29..0.68 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=49)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  Buffers: shared hit=98
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.68 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=49)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=98
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 2.100 ms
Execution Time: 12.110 ms
```

#### sweep bench_conc, regions 0-0 (|S| 1011), region-only
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=49.82..49.83 rows=1 width=16) (actual time=3.909..3.916 rows=1 loops=1)
  Buffers: shared hit=3073
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=40.61..40.62 rows=1 width=8) (actual time=3.281..3.285 rows=1 loops=1)
          Buffers: shared hit=3053
          ->  Limit  (cost=1.28..40.52 rows=7 width=4) (actual time=3.279..3.282 rows=0 loops=1)
                Buffers: shared hit=3053
                ->  Result  (cost=1.28..40.52 rows=7 width=4) (actual time=3.279..3.282 rows=0 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=3053
                      ->  Nested Loop  (cost=1.28..40.52 rows=7 width=0) (actual time=3.274..3.277 rows=0 loops=1)
                            Buffers: shared hit=3053
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.017..0.306 rows=1011 loops=1)
                                  Buffers: shared hit=20
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.013..0.016 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.003..0.004 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.007..0.008 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.003..0.171 rows=1011 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=16
                            ->  Limit  (cost=0.42..4.47 rows=1 width=4) (actual time=0.003..0.003 rows=0 loops=1011)
                                  Buffers: shared hit=3033
                                  ->  Result  (cost=0.42..4.47 rows=1 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=3033
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4.47 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1011)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=3033
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=0.624..0.627 rows=1 loops=1)
        Buffers: shared hit=20
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.097..0.548 rows=1011 loops=1)
              Buffers: shared hit=20
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.096..0.465 rows=1011 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=20
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.090..0.341 rows=1011 loops=1)
                          Buffers: shared hit=20
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.084..0.086 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.044..0.045 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.040 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.038..0.039 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.035..0.035 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.005..0.136 rows=1011 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=16
Planning Time: 0.677 ms
Execution Time: 3.997 ms
```

count:
```
Aggregate  (cost=196.01..196.02 rows=1 width=8) (actual time=10.607..10.620 rows=1 loops=1)
  Buffers: shared hit=3065
  ->  Limit  (cost=0.55..195.89 rows=10 width=4) (actual time=10.603..10.616 rows=0 loops=1)
        Buffers: shared hit=3065
        ->  Result  (cost=0.55..195.89 rows=10 width=4) (actual time=10.602..10.614 rows=0 loops=1)
              Buffers: shared hit=3065
              ->  Append  (cost=0.55..195.79 rows=10 width=0) (actual time=10.602..10.613 rows=0 loops=1)
                    Buffers: shared hit=3065
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.037..0.038 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.037..0.037 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.036..0.037 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.027..0.027 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..8.59 rows=1 width=0) (actual time=0.014..0.016 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (actual time=0.014..0.015 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (actual time=0.014..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.017..0.020 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.016..0.020 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.016..0.019 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.015..0.018 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.015..0.016 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.013..0.015 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=1.99..157.99 rows=7 width=0) (actual time=10.531..10.535 rows=0 loops=1)
                          Buffers: shared hit=3053
                          ->  Limit  (cost=1.99..157.92 rows=7 width=4) (actual time=10.531..10.535 rows=0 loops=1)
                                Buffers: shared hit=3053
                                ->  Result  (cost=1.99..157.92 rows=7 width=4) (actual time=10.530..10.534 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=3053
                                      ->  Nested Loop  (cost=1.99..157.92 rows=7 width=0) (actual time=10.529..10.533 rows=0 loops=1)
                                            Buffers: shared hit=3053
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.032..0.298 rows=1011 loops=1)
                                                  Buffers: shared hit=20
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.025..0.028 rows=1 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.008..0.009 rows=1 loops=1)
                                                              Group Key: acl_ud.id
                                                              Batches: 1  Memory Usage: 24kB
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.005 rows=1 loops=1)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.016..0.017 rows=0 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.014..0.015 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Rows Removed by Filter: 1
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.006..0.158 rows=1011 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=16
                                            ->  Limit  (cost=1.14..21.24 rows=1 width=4) (actual time=0.010..0.010 rows=0 loops=1011)
                                                  Buffers: shared hit=3033
                                                  ->  Result  (cost=1.14..21.24 rows=1 width=4) (actual time=0.010..0.010 rows=0 loops=1011)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=3033
                                                        ->  Nested Loop Anti Join  (cost=1.14..21.24 rows=1 width=0) (actual time=0.009..0.009 rows=0 loops=1011)
                                                              Buffers: shared hit=3033
                                                              ->  Nested Loop Semi Join  (cost=0.84..16.90 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=1011)
                                                                    Buffers: shared hit=3033
                                                                    ->  Index Scan using records_org_unit on records r_3  (cost=0.42..12.46 rows=1 width=48) (actual time=0.008..0.008 rows=0 loops=1011)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)))
                                                                          Buffers: shared hit=3033
                                                                    ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                          Heap Fetches: 0
                                                              ->  Result  (cost=0.29..4.31 rows=1 width=32) (never executed)
                                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                    ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..4.31 rows=1 width=32) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_3.id))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                          Heap Fetches: 0
Planning Time: 4.493 ms
Execution Time: 10.786 ms
```

page:
```
Nested Loop Left Join  (cost=215.48..247.99 rows=1 width=619) (actual time=1.737..1.747 rows=0 loops=1)
  Buffers: shared hit=3061
  ->  Nested Loop  (cost=215.18..239.66 rows=1 width=584) (actual time=1.737..1.746 rows=0 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3061
        ->  Subquery Scan on x  (cost=214.90..231.35 rows=1 width=577) (actual time=1.736..1.745 rows=0 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=3061
              ->  Limit  (cost=214.90..231.12 rows=10 width=577) (actual time=1.736..1.744 rows=0 loops=1)
                    Buffers: shared hit=3061
                    ->  Merge Append  (cost=214.90..231.12 rows=10 width=577) (actual time=1.735..1.743 rows=0 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=3061
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.023..0.024 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.023..0.024 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.021..0.021 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.012..0.012 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.012..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.011..0.011 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.017..0.019 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.016..0.018 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=4
                                      ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.014..0.016 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.013..0.014 rows=0 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                          ->  Sort  (cost=189.19..189.20 rows=7 width=278) (actual time=1.682..1.685 rows=0 loops=1)
                                Sort Key: r_3.sort_key DESC, r_3.id DESC
                                Sort Method: quicksort  Memory: 25kB
                                Buffers: shared hit=3049
                                ->  Limit  (cost=189.07..189.09 rows=7 width=278) (actual time=1.680..1.684 rows=0 loops=1)
                                      Buffers: shared hit=3049
                                      CTE acl_m
                                        ->  Result  (cost=77.27..188.83 rows=7 width=1275) (actual time=1.678..1.681 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=3049
                                              InitPlan 1 (returns $0)
                                                ->  Result  (cost=0.41..68.23 rows=991 width=16) (actual time=0.013..0.236 rows=1011 loops=1)
                                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                      Buffers: shared hit=16
                                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..68.23 rows=991 width=16) (actual time=0.012..0.139 rows=1011 loops=1)
                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])))
                                                            Heap Fetches: 0
                                                            Buffers: shared hit=16
                                              ->  Nested Loop Anti Join  (cost=9.04..120.60 rows=7 width=1267) (actual time=1.677..1.678 rows=0 loops=1)
                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                    Buffers: shared hit=3049
                                                    ->  Index Scan using records_org_unit on records r_4  (cost=0.42..104.40 rows=7 width=1267) (actual time=1.676..1.677 rows=0 loops=1)
                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                          Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)))
                                                          Buffers: shared hit=3049
                                                    ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (never executed)
                                                          ->  Result  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                                      Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                      ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (never executed)
                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                      ->  Sort  (cost=0.24..0.26 rows=7 width=278) (actual time=1.680..1.680 rows=0 loops=1)
                                            Sort Key: r_3.updated_at DESC, r_3.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=3049
                                            ->  CTE Scan on acl_m r_3  (cost=0.00..0.14 rows=7 width=278) (actual time=1.679..1.679 rows=0 loops=1)
                                                  Buffers: shared hit=3049
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (never executed)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (never executed)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
Planning Time: 1.361 ms
Execution Time: 1.891 ms
```

#### sweep bench_conc, regions 0-0 (|S| 1011), ragent-1
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=49.82..49.83 rows=1 width=16) (actual time=3.340..3.347 rows=1 loops=1)
  Buffers: shared hit=3073
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=40.61..40.62 rows=1 width=8) (actual time=2.789..2.792 rows=1 loops=1)
          Buffers: shared hit=3053
          ->  Limit  (cost=1.28..40.52 rows=7 width=4) (actual time=2.787..2.790 rows=0 loops=1)
                Buffers: shared hit=3053
                ->  Result  (cost=1.28..40.52 rows=7 width=4) (actual time=2.787..2.789 rows=0 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=3053
                      ->  Nested Loop  (cost=1.28..40.52 rows=7 width=0) (actual time=2.786..2.788 rows=0 loops=1)
                            Buffers: shared hit=3053
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.012..0.255 rows=1011 loops=1)
                                  Buffers: shared hit=20
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.008..0.010 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.003..0.004 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.002..0.002 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.003..0.004 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.003..0.133 rows=1011 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=16
                            ->  Limit  (cost=0.42..4.47 rows=1 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                  Buffers: shared hit=3033
                                  ->  Result  (cost=0.42..4.47 rows=1 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=3033
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..4.47 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1011)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=3033
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=0.548..0.551 rows=1 loops=1)
        Buffers: shared hit=20
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.052..0.484 rows=1011 loops=1)
              Buffers: shared hit=20
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.051..0.400 rows=1011 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=20
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.044..0.276 rows=1011 loops=1)
                          Buffers: shared hit=20
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.037..0.038 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.007..0.008 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.028..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.027..0.027 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.006..0.138 rows=1011 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=16
Planning Time: 0.724 ms
Execution Time: 3.433 ms
```

count:
```
Aggregate  (cost=287.00..287.01 rows=1 width=8) (actual time=1.172..1.175 rows=1 loops=1)
  Buffers: shared hit=29
  ->  Limit  (cost=0.55..282.76 rows=339 width=4) (actual time=0.215..1.066 rows=1001 loops=1)
        Buffers: shared hit=29
        ->  Result  (cost=0.55..282.76 rows=339 width=4) (actual time=0.213..0.986 rows=1001 loops=1)
              Buffers: shared hit=29
              ->  Append  (cost=0.55..279.37 rows=339 width=0) (actual time=0.212..0.852 rows=1001 loops=1)
                    Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..34.12 rows=325 width=0) (actual time=0.208..0.760 rows=1001 loops=1)
                          Buffers: shared hit=29
                          ->  Limit  (cost=0.55..30.88 rows=325 width=4) (actual time=0.203..0.660 rows=1001 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.55..30.88 rows=325 width=4) (actual time=0.203..0.581 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=29
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..30.88 rows=325 width=0) (actual time=0.167..0.435 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..24.67 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.55..24.62 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.55..24.62 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..24.62 rows=5 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..8.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=1.99..158.12 rows=7 width=0) (never executed)
                          ->  Limit  (cost=1.99..158.05 rows=7 width=4) (never executed)
                                ->  Result  (cost=1.99..158.05 rows=7 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=1.99..158.05 rows=7 width=0) (never executed)
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                                              Group Key: acl_ud.id
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Heap Fetches: 0
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                            ->  Limit  (cost=1.14..21.26 rows=1 width=4) (never executed)
                                                  ->  Result  (cost=1.14..21.26 rows=1 width=4) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Nested Loop Anti Join  (cost=1.14..21.26 rows=1 width=0) (never executed)
                                                              ->  Nested Loop Semi Join  (cost=0.84..16.92 rows=1 width=32) (never executed)
                                                                    ->  Index Scan using records_org_unit on records r_4  (cost=0.42..12.47 rows=1 width=48) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                    ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                          Heap Fetches: 0
                                                              ->  Result  (cost=0.29..4.32 rows=1 width=32) (never executed)
                                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                    ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..4.32 rows=1 width=32) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_4.id))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                          Heap Fetches: 0
Planning Time: 1.841 ms
Execution Time: 1.593 ms
```

page:
```
Nested Loop Left Join  (cost=1550.85..1594.61 rows=1 width=1223) (actual time=4.413..4.729 rows=51 loops=1)
  Buffers: shared hit=4548
  ->  Nested Loop  (cost=1550.55..1586.28 rows=1 width=1188) (actual time=4.398..4.594 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=4395
        ->  Subquery Scan on x  (cost=1550.27..1577.96 rows=1 width=1181) (actual time=4.382..4.452 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4191
              ->  Limit  (cost=1550.27..1576.82 rows=51 width=1181) (actual time=4.378..4.418 rows=51 loops=1)
                    Buffers: shared hit=4191
                    ->  Merge Append  (cost=1550.27..1583.58 rows=64 width=1181) (actual time=4.377..4.412 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4191
                          ->  Limit  (cost=1297.94..1298.06 rows=51 width=1275) (actual time=2.592..2.604 rows=50 loops=1)
                                Buffers: shared hit=1104
                                ->  Sort  (cost=1297.94..1298.75 rows=325 width=1275) (actual time=2.591..2.598 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=1104
                                      ->  Result  (cost=27.72..1287.09 rows=325 width=1275) (actual time=0.351..1.826 rows=1236 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=1104
                                            ->  Bitmap Heap Scan on records r  (cost=27.72..1287.09 rows=325 width=1267) (actual time=0.347..1.452 rows=1236 loops=1)
                                                  Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                  Heap Blocks: exact=1069
                                                  Buffers: shared hit=1104
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..27.62 rows=325 width=0) (actual time=0.202..0.202 rows=1236 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                        Buffers: shared hit=35
                          ->  Limit  (cost=0.55..24.62 rows=5 width=1275) (actual time=0.026..0.031 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..24.62 rows=5 width=1275) (actual time=0.025..0.029 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..24.62 rows=5 width=1267) (actual time=0.018..0.021 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.014..0.014 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.013..0.014 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.011..0.011 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.069..0.071 rows=0 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.068..0.070 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.064..0.066 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.063..0.064 rows=0 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.036..0.037 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.027..0.031 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.008..0.008 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
                          ->  Sort  (cost=199.00..199.02 rows=6 width=278) (actual time=1.672..1.674 rows=0 loops=1)
                                Sort Key: r_4.sort_key DESC, r_4.id DESC
                                Sort Method: quicksort  Memory: 25kB
                                Buffers: shared hit=3049
                                ->  Limit  (cost=198.91..198.92 rows=6 width=278) (actual time=1.669..1.672 rows=0 loops=1)
                                      Buffers: shared hit=3049
                                      CTE acl_m
                                        ->  Result  (cost=68.95..198.71 rows=6 width=1275) (actual time=1.666..1.668 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=3049
                                              InitPlan 1 (returns $0)
                                                ->  Result  (cost=0.41..68.23 rows=991 width=16) (actual time=0.014..0.235 rows=1011 loops=1)
                                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                      Buffers: shared hit=16
                                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..68.23 rows=991 width=16) (actual time=0.012..0.136 rows=1011 loops=1)
                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])))
                                                            Heap Fetches: 0
                                                            Buffers: shared hit=16
                                              ->  Nested Loop Anti Join  (cost=0.72..130.48 rows=6 width=1267) (actual time=1.665..1.665 rows=0 loops=1)
                                                    Buffers: shared hit=3049
                                                    ->  Index Scan using records_org_unit on records r_5  (cost=0.42..104.49 rows=6 width=1267) (actual time=1.664..1.664 rows=0 loops=1)
                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                          Buffers: shared hit=3049
                                                    ->  Result  (cost=0.29..4.32 rows=1 width=32) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..4.32 rows=1 width=32) (never executed)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_5.id))
                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                Heap Fetches: 0
                                      ->  Sort  (cost=0.20..0.21 rows=6 width=278) (actual time=1.669..1.669 rows=0 loops=1)
                                            Sort Key: r_4.updated_at DESC, r_4.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=3049
                                            ->  CTE Scan on acl_m r_4  (cost=0.00..0.12 rows=6 width=278) (actual time=1.667..1.667 rows=0 loops=1)
                                                  Buffers: shared hit=3049
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.486 ms
Execution Time: 4.884 ms
```

#### root (|S| 10,111), region-only
N̂_e 400000, K* 3613, alpha 8.0, cap ceil(N̂_e/alpha) = 50000

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=57.57..57.59 rows=1 width=16) (actual time=8.403..8.410 rows=1 loops=1)
  Buffers: shared hit=586
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=48.36..48.37 rows=1 width=8) (actual time=2.865..2.869 rows=1 loops=1)
          Buffers: shared hit=466
          ->  Limit  (cost=1.28..47.05 rows=105 width=4) (actual time=0.248..2.640 rows=3614 loops=1)
                Buffers: shared hit=466
                ->  Result  (cost=1.28..47.05 rows=105 width=4) (actual time=0.246..2.348 rows=3614 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=466
                      ->  Nested Loop  (cost=1.28..47.05 rows=105 width=0) (actual time=0.179..1.883 rows=3614 loops=1)
                            Buffers: shared hit=466
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.149..0.188 rows=139 loops=1)
                                  Buffers: shared hit=10
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.132..0.134 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.085..0.086 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.068..0.069 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.034..0.034 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.025..0.025 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.012..0.034 rows=139 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=6
                            ->  Limit  (cost=0.42..5.12 rows=15 width=4) (actual time=0.002..0.009 rows=26 loops=139)
                                  Buffers: shared hit=456
                                  ->  Result  (cost=0.42..5.12 rows=15 width=4) (actual time=0.002..0.007 rows=26 loops=139)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=456
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.12 rows=15 width=0) (actual time=0.002..0.003 rows=26 loops=139)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=456
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=5.518..5.521 rows=1 loops=1)
        Buffers: shared hit=120
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.042..4.808 rows=10111 loops=1)
              Buffers: shared hit=120
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.041..3.959 rows=10111 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=120
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.036..2.811 rows=10111 loops=1)
                          Buffers: shared hit=120
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.028..0.043 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.007..0.017 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.021..0.021 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.007..1.758 rows=10111 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=116
Planning Time: 0.600 ms
Execution Time: 8.490 ms
```

count:
```
Aggregate  (cost=1148.95..1148.96 rows=1 width=8) (actual time=2.399..2.412 rows=1 loops=1)
  Buffers: shared hit=417
  ->  Limit  (cost=0.55..1147.34 rows=129 width=4) (actual time=0.146..2.314 rows=1001 loops=1)
        Buffers: shared hit=417
        ->  Result  (cost=0.55..1147.34 rows=129 width=4) (actual time=0.145..2.216 rows=1001 loops=1)
              Buffers: shared hit=417
              ->  Append  (cost=0.55..1146.05 rows=129 width=0) (actual time=0.144..2.083 rows=1001 loops=1)
                    Buffers: shared hit=417
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..5.14 rows=17 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.97 rows=17 width=4) (actual time=0.022..0.023 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.97 rows=17 width=4) (actual time=0.022..0.022 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.97 rows=17 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..56.78 rows=13 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.65 rows=13 width=4) (actual time=0.012..0.012 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.65 rows=13 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..56.65 rows=13 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.014..0.016 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.013..0.016 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.013..0.015 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.012..0.014 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=10.31..1058.91 rows=98 width=0) (actual time=0.096..1.927 rows=1001 loops=1)
                          Buffers: shared hit=405
                          ->  Limit  (cost=10.31..1057.93 rows=98 width=4) (actual time=0.095..1.804 rows=1001 loops=1)
                                Buffers: shared hit=405
                                ->  Result  (cost=10.31..1057.93 rows=98 width=4) (actual time=0.095..1.706 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=405
                                      ->  Nested Loop  (cost=10.31..1057.93 rows=98 width=0) (actual time=0.094..1.574 rows=1001 loops=1)
                                            Buffers: shared hit=405
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.024..0.038 rows=25 loops=1)
                                                  Buffers: shared hit=9
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.017..0.019 rows=1 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.005..0.005 rows=1 loops=1)
                                                              Group Key: acl_ud.id
                                                              Batches: 1  Memory Usage: 24kB
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.003..0.003 rows=1 loops=1)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Rows Removed by Filter: 1
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.006..0.015 rows=25 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Limit  (cost=9.45..149.55 rows=14 width=4) (actual time=0.009..0.057 rows=40 loops=25)
                                                  Buffers: shared hit=396
                                                  ->  Result  (cost=9.45..149.55 rows=14 width=4) (actual time=0.009..0.053 rows=40 loops=25)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=396
                                                        ->  Nested Loop Anti Join  (cost=9.45..149.55 rows=14 width=0) (actual time=0.007..0.046 rows=40 loops=25)
                                                              Join Filter: (acl_share_1.record_id = r_3.id)
                                                              Buffers: shared hit=396
                                                              ->  Nested Loop Semi Join  (cost=0.84..133.25 rows=14 width=32) (actual time=0.006..0.037 rows=40 loops=25)
                                                                    Buffers: shared hit=392
                                                                    ->  Index Scan using records_org_unit on records r_3  (cost=0.42..128.63 rows=14 width=48) (actual time=0.004..0.026 rows=40 loops=25)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)))
                                                                          Buffers: shared hit=355
                                                                    ->  Materialize  (cost=0.41..4.44 rows=1 width=16) (actual time=0.000..0.000 rows=1 loops=1001)
                                                                          Buffers: shared hit=37
                                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (actual time=0.003..0.003 rows=1 loops=12)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                                Heap Fetches: 0
                                                                                Buffers: shared hit=37
                                                              ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=4
                                                                    ->  Result  (cost=8.60..16.08 rows=1 width=32) (actual time=0.006..0.008 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                          Buffers: shared hit=4
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (actual time=0.004..0.006 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Buffers: shared hit=4
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.004..0.005 rows=0 loops=1)
                                                                                      Buffers: shared hit=4
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                                            Buffers: shared hit=2
Planning Time: 1.103 ms
Execution Time: 2.653 ms
```

page:
```
Nested Loop Left Join  (cost=27.01..1094.56 rows=1 width=1317) (actual time=0.115..1.109 rows=51 loops=1)
  Buffers: shared hit=629
  ->  Nested Loop  (cost=26.71..1086.22 rows=1 width=1282) (actual time=0.100..0.921 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=476
        ->  Subquery Scan on x  (cost=26.43..1077.91 rows=1 width=1275) (actual time=0.092..0.724 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=323
              ->  Limit  (cost=26.43..1076.76 rows=51 width=1275) (actual time=0.090..0.688 rows=51 loops=1)
                    Buffers: shared hit=323
                    ->  Merge Append  (cost=26.43..1715.20 rows=82 width=1275) (actual time=0.090..0.680 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=323
                          ->  Limit  (cost=0.55..72.63 rows=17 width=1275) (actual time=0.022..0.023 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..72.63 rows=17 width=1275) (actual time=0.022..0.023 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..72.63 rows=17 width=1267) (actual time=0.019..0.019 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.65 rows=13 width=1275) (actual time=0.011..0.012 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.65 rows=13 width=1275) (actual time=0.011..0.011 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..56.65 rows=13 width=1267) (actual time=0.010..0.010 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.015..0.017 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.014..0.017 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=4
                                      ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.012..0.014 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.012..0.013 rows=0 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.011..0.013 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                          ->  Limit  (cost=0.72..1560.08 rows=51 width=1275) (actual time=0.041..0.619 rows=51 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.72..1425558.47 rows=46624 width=1275) (actual time=0.040..0.611 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Anti Join  (cost=0.72..1425558.47 rows=46624 width=1267) (actual time=0.039..0.595 rows=51 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_updated on records r_3  (cost=0.42..1409667.93 rows=46624 width=1267) (actual time=0.032..0.339 rows=51 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (SubPlan 1))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=209
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.004..0.004 rows=1 loops=51)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=154
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=1 loops=51)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_3.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=154
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=51)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  Buffers: shared hit=102
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=51)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_3.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=102
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.203 ms
Execution Time: 1.243 ms
```

#### root (|S| 10,111), ragent-1
N̂_e 400000, K* 3613, alpha 8.0, cap ceil(N̂_e/alpha) = 50000

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=57.57..57.59 rows=1 width=16) (actual time=7.489..7.496 rows=1 loops=1)
  Buffers: shared hit=586
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=48.36..48.37 rows=1 width=8) (actual time=2.708..2.711 rows=1 loops=1)
          Buffers: shared hit=466
          ->  Limit  (cost=1.28..47.05 rows=105 width=4) (actual time=0.073..2.485 rows=3614 loops=1)
                Buffers: shared hit=466
                ->  Result  (cost=1.28..47.05 rows=105 width=4) (actual time=0.071..2.194 rows=3614 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=466
                      ->  Nested Loop  (cost=1.28..47.05 rows=105 width=0) (actual time=0.066..1.793 rows=3614 loops=1)
                            Buffers: shared hit=466
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.045..0.083 rows=139 loops=1)
                                  Buffers: shared hit=10
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.037..0.039 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.023..0.024 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.009..0.010 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.006..0.028 rows=139 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=6
                            ->  Limit  (cost=0.42..5.12 rows=15 width=4) (actual time=0.002..0.010 rows=26 loops=139)
                                  Buffers: shared hit=456
                                  ->  Result  (cost=0.42..5.12 rows=15 width=4) (actual time=0.002..0.007 rows=26 loops=139)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=456
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.12 rows=15 width=0) (actual time=0.001..0.003 rows=26 loops=139)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=456
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=4.778..4.780 rows=1 loops=1)
        Buffers: shared hit=120
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.041..4.162 rows=10111 loops=1)
              Buffers: shared hit=120
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.040..3.357 rows=10111 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=120
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.035..2.244 rows=10111 loops=1)
                          Buffers: shared hit=120
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.028..0.031 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.007..0.008 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.020..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.006..1.252 rows=10111 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=116
Planning Time: 0.597 ms
Execution Time: 7.570 ms
```

count:
```
Aggregate  (cost=1589.60..1589.61 rows=1 width=8) (actual time=0.935..0.938 rows=1 loops=1)
  Buffers: shared hit=567
  ->  Limit  (cost=0.55..1577.09 rows=1001 width=4) (actual time=0.037..0.856 rows=1001 loops=1)
        Buffers: shared hit=567
        ->  Result  (cost=0.55..1912.55 rows=1214 width=4) (actual time=0.036..0.765 rows=1001 loops=1)
              Buffers: shared hit=567
              ->  Append  (cost=0.55..1900.41 rows=1214 width=0) (actual time=0.036..0.613 rows=1001 loops=1)
                    Buffers: shared hit=567
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..83.74 rows=1001 width=0) (actual time=0.035..0.529 rows=1001 loops=1)
                          Buffers: shared hit=567
                          ->  Limit  (cost=0.55..73.73 rows=1001 width=4) (actual time=0.035..0.432 rows=1001 loops=1)
                                Buffers: shared hit=567
                                ->  Result  (cost=0.55..490.15 rows=6697 width=4) (actual time=0.035..0.353 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=567
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..490.15 rows=6697 width=0) (actual time=0.028..0.236 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=567
                    ->  Subquery Scan on "*SELECT* 2"  (cost=10.25..448.48 rows=108 width=0) (never executed)
                          ->  Limit  (cost=10.25..447.40 rows=108 width=4) (never executed)
                                ->  Result  (cost=10.25..447.40 rows=108 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Bitmap Heap Scan on records r_1  (cost=10.25..447.40 rows=108 width=0) (never executed)
                                            Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Bitmap Index Scan on records_list_requester  (cost=0.00..10.21 rows=111 width=0) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..56.86 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.55..56.73 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.55..56.73 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.73 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=27.49..1253.08 rows=91 width=0) (never executed)
                          ->  Limit  (cost=27.49..1252.17 rows=91 width=4) (never executed)
                                ->  Result  (cost=27.49..1252.17 rows=91 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=27.49..1252.17 rows=91 width=0) (never executed)
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                                              Group Key: acl_ud.id
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Heap Fetches: 0
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                            ->  Limit  (cost=26.63..177.32 rows=13 width=4) (never executed)
                                                  ->  Result  (cost=26.63..177.32 rows=13 width=4) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Nested Loop Anti Join  (cost=26.63..177.32 rows=13 width=0) (never executed)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              ->  Nested Loop Semi Join  (cost=0.84..133.43 rows=13 width=32) (never executed)
                                                                    ->  Index Scan using records_org_unit on records r_4  (cost=0.42..128.82 rows=13 width=48) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                    ->  Materialize  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                                Heap Fetches: 0
                                                              ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                                    ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                                Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 1.471 ms
Execution Time: 1.103 ms
```

page:
```
Nested Loop Left Join  (cost=55.06..1774.43 rows=1 width=1317) (actual time=0.250..1.040 rows=51 loops=1)
  Buffers: shared hit=714
  ->  Nested Loop  (cost=54.76..1766.10 rows=1 width=1282) (actual time=0.235..0.861 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=561
        ->  Subquery Scan on x  (cost=54.48..1757.78 rows=1 width=1275) (actual time=0.227..0.748 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=408
              ->  Limit  (cost=54.48..1756.64 rows=51 width=1275) (actual time=0.226..0.715 rows=51 loops=1)
                    Buffers: shared hit=408
                    ->  Merge Append  (cost=54.48..5628.20 rows=167 width=1275) (actual time=0.225..0.708 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=408
                          ->  Limit  (cost=0.42..3688.24 rows=51 width=1275) (actual time=0.090..0.136 rows=2 loops=1)
                                Buffers: shared hit=65
                                ->  Result  (cost=0.42..484261.75 rows=6697 width=1275) (actual time=0.090..0.136 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=65
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..484261.75 rows=6697 width=1267) (actual time=0.086..0.132 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=65
                          ->  Limit  (cost=0.55..214.52 rows=51 width=1275) (actual time=0.017..0.017 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..453.65 rows=108 width=1275) (actual time=0.016..0.017 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..453.65 rows=108 width=1267) (actual time=0.015..0.015 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..56.73 rows=13 width=1275) (actual time=0.015..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..56.73 rows=13 width=1275) (actual time=0.014..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.73 rows=13 width=1267) (actual time=0.013..0.013 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.073..0.075 rows=1 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.073..0.074 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.049..0.066 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.048..0.064 rows=3 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.035..0.037 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.027..0.032 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1613.70 rows=51 width=1275) (actual time=0.028..0.452 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=0.72..1426596.63 rows=45107 width=1275) (actual time=0.028..0.445 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=0.72..1426596.63 rows=45107 width=1267) (actual time=0.027..0.430 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1410971.69 rows=45107 width=1267) (actual time=0.019..0.265 rows=50 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND (SubPlan 1))
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=206
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.004..0.004 rows=1 loops=50)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=151
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=1 loops=50)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.430 ms
Execution Time: 1.223 ms
```

### n_distinct(records.org_unit_id) = 100 (pg_stats: 100)

#### bench_conc, regions 0-8 (|S| 9,099), region-only
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=732.66..732.67 rows=1 width=16) (actual time=1.172..1.175 rows=1 loops=1)
  Buffers: shared hit=118
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=388.89..388.90 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.38..386.62 rows=182 width=4) (never executed)
                ->  Result  (cost=1.38..386.62 rows=182 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.38..386.62 rows=182 width=0) (never executed)
                            ->  Nested Loop  (cost=0.95..343.66 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.09 rows=9 width=16) (never executed)
                                        ->  Result  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                  ->  Result  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=26 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=343.76..343.77 rows=1 width=8) (actual time=1.171..1.172 rows=1 loops=1)
        Buffers: shared hit=118
        ->  Limit  (cost=0.96..343.67 rows=7 width=4) (actual time=0.067..1.035 rows=2500 loops=1)
              Buffers: shared hit=118
              ->  Result  (cost=0.96..343.67 rows=7 width=4) (actual time=0.066..0.858 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=118
                    ->  Nested Loop  (cost=0.96..343.67 rows=7 width=0) (actual time=0.061..0.589 rows=2500 loops=1)
                          Buffers: shared hit=118
                          ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (actual time=0.055..0.097 rows=3 loops=1)
                                Buffers: shared hit=82
                                ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (actual time=0.009..0.010 rows=3 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.09 rows=9 width=16) (actual time=0.004..0.004 rows=9 loops=1)
                                ->  Result  (cost=0.41..37.64 rows=1 width=32) (actual time=0.028..0.028 rows=0 loops=3)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=82
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..37.64 rows=1 width=32) (actual time=0.026..0.026 rows=0 loops=3)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=82
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.005..0.093 rows=833 loops=3)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=36
Planning:
  Buffers: shared hit=29
Planning Time: 0.694 ms
Execution Time: 1.239 ms
```

count:
```
Aggregate  (cost=10527.03..10527.04 rows=1 width=8) (actual time=17.590..17.600 rows=1 loops=1)
  Buffers: shared hit=19952
  ->  Limit  (cost=0.55..10514.52 rows=1001 width=4) (actual time=17.587..17.596 rows=0 loops=1)
        Buffers: shared hit=19952
        ->  Result  (cost=0.55..10546.03 rows=1004 width=4) (actual time=17.586..17.595 rows=0 loops=1)
              Buffers: shared hit=19952
              ->  Append  (cost=0.55..10535.99 rows=1004 width=0) (actual time=17.585..17.595 rows=0 loops=1)
                    Buffers: shared hit=19952
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.016..0.017 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.016..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..8.59 rows=1 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (actual time=0.010..0.010 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.011..0.015 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.011..0.015 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.011..0.015 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.010..0.013 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.010..0.013 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.009..0.012 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.008..0.010 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.004 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=610.26..10493.23 rows=1001 width=0) (actual time=17.545..17.548 rows=0 loops=1)
                          Buffers: shared hit=19940
                          ->  Limit  (cost=610.26..10483.22 rows=1001 width=4) (actual time=17.545..17.547 rows=0 loops=1)
                                Buffers: shared hit=19940
                                ->  Result  (cost=610.26..24173.19 rows=2389 width=4) (actual time=17.544..17.546 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=19940
                                      ->  Nested Loop Anti Join  (cost=610.26..24173.19 rows=2389 width=0) (actual time=17.543..17.545 rows=0 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=19940
                                            ->  Index Scan using records_org_unit on records r_3  (cost=601.64..24121.26 rows=2389 width=32) (actual time=17.543..17.543 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 20000
                                                  Buffers: shared hit=19940
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.011..1.909 rows=9099 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=127
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.009..1.066 rows=9099 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=127
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning:
  Buffers: shared hit=27
Planning Time: 0.805 ms
Execution Time: 17.686 ms
```

page:
```
Nested Loop Left Join  (cost=636.55..1514.53 rows=1 width=1317) (actual time=18.108..18.118 rows=0 loops=1)
  Buffers: shared hit=20327
  ->  Nested Loop  (cost=636.25..1506.19 rows=1 width=1282) (actual time=18.108..18.117 rows=0 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=20327
        ->  Subquery Scan on x  (cost=635.97..1497.88 rows=1 width=1275) (actual time=18.108..18.117 rows=0 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=20327
              ->  Limit  (cost=635.97..1496.73 rows=51 width=1275) (actual time=18.107..18.115 rows=0 loops=1)
                    Buffers: shared hit=20327
                    ->  Merge Append  (cost=635.97..1547.37 rows=54 width=1275) (actual time=18.106..18.114 rows=0 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=20327
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.019..0.020 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.019..0.019 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.015..0.015 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.011..0.012 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.011..0.011 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.010..0.010 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.015..0.018 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.015..0.018 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=4
                                      ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.012..0.015 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.012..0.014 rows=0 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.011..0.013 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                          ->  Limit  (cost=610.26..1504.79 rows=51 width=1275) (actual time=18.059..18.061 rows=0 loops=1)
                                Buffers: shared hit=20315
                                ->  Result  (cost=610.26..42512.75 rows=2389 width=1275) (actual time=18.058..18.060 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=20315
                                      ->  Nested Loop Anti Join  (cost=610.26..42512.75 rows=2389 width=1267) (actual time=18.057..18.059 rows=0 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=20315
                                            ->  Index Scan using records_list_updated on records r_3  (cost=601.65..42460.82 rows=2389 width=1267) (actual time=18.056..18.057 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 20000
                                                  Buffers: shared hit=20315
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.015..1.981 rows=9099 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=127
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.014..1.162 rows=9099 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=127
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (never executed)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (never executed)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
Planning:
  Buffers: shared hit=55
Planning Time: 1.344 ms
Execution Time: 18.255 ms
```

#### bench_conc, regions 0-8 (|S| 9,099), ragent-1
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=732.66..732.67 rows=1 width=16) (actual time=1.204..1.208 rows=1 loops=1)
  Buffers: shared hit=118
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=388.89..388.90 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.38..386.62 rows=182 width=4) (never executed)
                ->  Result  (cost=1.38..386.62 rows=182 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.38..386.62 rows=182 width=0) (never executed)
                            ->  Nested Loop  (cost=0.95..343.66 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.09 rows=9 width=16) (never executed)
                                        ->  Result  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..37.64 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                  ->  Result  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=26 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=343.76..343.77 rows=1 width=8) (actual time=1.203..1.205 rows=1 loops=1)
        Buffers: shared hit=118
        ->  Limit  (cost=0.96..343.67 rows=7 width=4) (actual time=0.068..1.066 rows=2500 loops=1)
              Buffers: shared hit=118
              ->  Result  (cost=0.96..343.67 rows=7 width=4) (actual time=0.067..0.889 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=118
                    ->  Nested Loop  (cost=0.96..343.67 rows=7 width=0) (actual time=0.063..0.641 rows=2500 loops=1)
                          Buffers: shared hit=118
                          ->  Nested Loop Anti Join  (cost=0.53..339.03 rows=1 width=16) (actual time=0.057..0.106 rows=3 loops=1)
                                Buffers: shared hit=82
                                ->  HashAggregate  (cost=0.11..0.20 rows=9 width=16) (actual time=0.010..0.012 rows=3 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.09 rows=9 width=16) (actual time=0.005..0.005 rows=9 loops=1)
                                ->  Result  (cost=0.41..37.64 rows=1 width=32) (actual time=0.030..0.030 rows=0 loops=3)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=82
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..37.64 rows=1 width=32) (actual time=0.028..0.028 rows=0 loops=3)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=82
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.005..0.108 rows=833 loops=3)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=36
Planning Time: 0.556 ms
Execution Time: 1.274 ms
```

count:
```
Aggregate  (cost=8176.20..8176.21 rows=1 width=8) (actual time=0.769..0.772 rows=1 loops=1)
  Buffers: shared hit=29
  ->  Limit  (cost=0.55..8163.68 rows=1001 width=4) (actual time=0.051..0.698 rows=1001 loops=1)
        Buffers: shared hit=29
        ->  Result  (cost=0.55..11017.93 rows=1351 width=4) (actual time=0.050..0.626 rows=1001 loops=1)
              Buffers: shared hit=29
              ->  Append  (cost=0.55..11004.42 rows=1351 width=0) (actual time=0.049..0.510 rows=1001 loops=1)
                    Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..34.61 rows=338 width=0) (actual time=0.049..0.435 rows=1001 loops=1)
                          Buffers: shared hit=29
                          ->  Limit  (cost=0.55..31.23 rows=338 width=4) (actual time=0.048..0.349 rows=1001 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.55..31.23 rows=338 width=4) (actual time=0.048..0.280 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=29
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..31.23 rows=338 width=0) (actual time=0.043..0.178 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..44.82 rows=10 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.72 rows=10 width=4) (never executed)
                                ->  Result  (cost=0.55..44.72 rows=10 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.72 rows=10 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..8.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=627.44..10857.46 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=627.44..10847.45 rows=1001 width=4) (never executed)
                                ->  Result  (cost=627.44..24232.49 rows=2312 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop Anti Join  (cost=627.44..24232.49 rows=2312 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=601.64..24154.11 rows=2312 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (never executed)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 0.923 ms
Execution Time: 0.862 ms
```

page:
```
Nested Loop Left Join  (cost=2003.85..2474.43 rows=1 width=1317) (actual time=21.700..21.993 rows=51 loops=1)
  Buffers: shared hit=21814
  ->  Nested Loop  (cost=2003.56..2466.10 rows=1 width=1282) (actual time=21.684..21.869 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=21661
        ->  Subquery Scan on x  (cost=2003.27..2457.78 rows=1 width=1275) (actual time=21.666..21.736 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21457
              ->  Limit  (cost=2003.27..2456.64 rows=51 width=1275) (actual time=21.657..21.701 rows=51 loops=1)
                    Buffers: shared hit=21457
                    ->  Merge Append  (cost=2003.27..3016.67 rows=114 width=1275) (actual time=21.656..21.695 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21457
                          ->  Limit  (cost=1347.99..1348.12 rows=51 width=1275) (actual time=2.277..2.291 rows=50 loops=1)
                                Buffers: shared hit=1104
                                ->  Sort  (cost=1347.99..1348.84 rows=338 width=1275) (actual time=2.277..2.287 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=1104
                                      ->  Result  (cost=27.95..1336.72 rows=338 width=1275) (actual time=0.305..1.589 rows=1236 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=1104
                                            ->  Bitmap Heap Scan on records r  (cost=27.95..1336.72 rows=338 width=1267) (actual time=0.301..1.243 rows=1236 loops=1)
                                                  Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                  Heap Blocks: exact=1069
                                                  Buffers: shared hit=1104
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..27.85 rows=338 width=0) (actual time=0.172..0.172 rows=1236 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                        Buffers: shared hit=35
                          ->  Limit  (cost=0.55..44.72 rows=10 width=1275) (actual time=0.022..0.026 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..44.72 rows=10 width=1275) (actual time=0.021..0.025 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.72 rows=10 width=1267) (actual time=0.015..0.018 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.012..0.013 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.012..0.013 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.010..0.010 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.059..0.062 rows=0 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.059..0.062 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.056..0.058 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.054..0.056 rows=0 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.032..0.034 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.024..0.028 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.007..0.007 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=601.94..1561.11 rows=51 width=1275) (actual time=19.280..19.283 rows=0 loops=1)
                                Buffers: shared hit=20315
                                ->  Result  (cost=601.94..44083.98 rows=2312 width=1275) (actual time=19.280..19.282 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=20315
                                      ->  Nested Loop Anti Join  (cost=601.94..44083.98 rows=2312 width=1267) (actual time=19.278..19.280 rows=0 loops=1)
                                            Buffers: shared hit=20315
                                            ->  Index Scan using records_list_updated on records r_4  (cost=601.65..42527.90 rows=2312 width=1267) (actual time=19.277..19.278 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 20000
                                                  Buffers: shared hit=20315
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.012..1.849 rows=9099 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=127
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..578.83 rows=8955 width=16) (actual time=0.011..1.034 rows=9099 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3,112fcbac-557b-8737-d4eb-8030eddd3ece,305e12c7-73a6-cb5f-9d00-f8694ffe89eb,66a12f84-9258-e1db-6da5-5387ca4e13fa,afe7459c-e607-d799-1bdf-3c29f8e53298,c2393ab8-341c-6223-a04f-99e00675db6d,caea35dd-96ae-9c54-6198-902dee710dc3,e4959634-06fc-be78-f8c9-225e7a040bad,f35233cd-75aa-8b69-4d97-5c42d607b7d3}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=127
                                            ->  Result  (cost=0.29..0.66 rows=1 width=32) (never executed)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.66 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.247 ms
Execution Time: 22.139 ms
```

#### bench_conc at root (|S| 10,111), region-only
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=63.54..63.55 rows=1 width=16) (actual time=1.135..1.139 rows=1 loops=1)
  Buffers: shared hit=36
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=54.33..54.34 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.28..52.06 rows=182 width=4) (never executed)
                ->  Result  (cost=1.28..52.06 rows=182 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.28..52.06 rows=182 width=0) (never executed)
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                  ->  Result  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=26 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=1.134..1.135 rows=1 loops=1)
        Buffers: shared hit=36
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.052..0.994 rows=2500 loops=1)
              Buffers: shared hit=36
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.051..0.813 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=36
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.043..0.559 rows=2500 loops=1)
                          Buffers: shared hit=36
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.035..0.036 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.007..0.008 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.005..0.005 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.027..0.027 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.025..0.025 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.007..0.313 rows=2500 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=32
Planning Time: 0.987 ms
Execution Time: 1.249 ms
```

count:
```
Aggregate  (cost=10614.00..10614.01 rows=1 width=8) (actual time=7.318..7.328 rows=1 loops=1)
  Buffers: shared hit=1114
  ->  Limit  (cost=0.55..10601.49 rows=1001 width=4) (actual time=5.263..7.233 rows=1001 loops=1)
        Buffers: shared hit=1114
        ->  Result  (cost=0.55..10633.26 rows=1004 width=4) (actual time=5.262..7.128 rows=1001 loops=1)
              Buffers: shared hit=1114
              ->  Append  (cost=0.55..10623.22 rows=1004 width=0) (actual time=5.261..7.005 rows=1001 loops=1)
                    Buffers: shared hit=1114
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.029..0.029 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.029..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..8.59 rows=1 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (actual time=0.013..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.014..0.017 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.014..0.017 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.014..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.013..0.015 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.004 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=697.49..10580.46 rows=1001 width=0) (actual time=5.204..6.834 rows=1001 loops=1)
                          Buffers: shared hit=1102
                          ->  Limit  (cost=697.49..10570.45 rows=1001 width=4) (actual time=5.203..6.725 rows=1001 loops=1)
                                Buffers: shared hit=1102
                                ->  Result  (cost=697.49..24260.42 rows=2389 width=4) (actual time=5.203..6.652 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=1102
                                      ->  Nested Loop Anti Join  (cost=697.49..24260.42 rows=2389 width=0) (actual time=5.202..6.529 rows=1001 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=1102
                                            ->  Index Scan using records_org_unit on records r_3  (cost=688.88..24208.49 rows=2389 width=32) (actual time=5.183..6.298 rows=1001 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Buffers: shared hit=1098
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.015..2.732 rows=10111 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=116
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.013..1.592 rows=10111 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=116
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                  Buffers: shared hit=4
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (actual time=0.014..0.015 rows=0 loops=1)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
Planning Time: 0.955 ms
Execution Time: 7.470 ms
```

page:
```
Nested Loop Left Join  (cost=723.78..1601.76 rows=1 width=1317) (actual time=4.095..4.586 rows=51 loops=1)
  Buffers: shared hit=543
  ->  Nested Loop  (cost=723.48..1593.42 rows=1 width=1282) (actual time=4.081..4.426 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=390
        ->  Subquery Scan on x  (cost=723.20..1585.11 rows=1 width=1275) (actual time=4.070..4.293 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=186
              ->  Limit  (cost=723.20..1583.96 rows=51 width=1275) (actual time=4.068..4.263 rows=51 loops=1)
                    Buffers: shared hit=186
                    ->  Merge Append  (cost=723.20..1634.60 rows=54 width=1275) (actual time=4.067..4.256 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=186
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.037..0.038 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.037..0.037 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.034..0.034 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.012..0.013 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.012..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.010..0.010 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.013..0.016 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.013..0.015 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=4
                                      ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.010..0.012 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.010..0.011 rows=0 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.009..0.011 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.008..0.009 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.003 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                          ->  Limit  (cost=697.49..1592.02 rows=51 width=1275) (actual time=4.004..4.181 rows=51 loops=1)
                                Buffers: shared hit=174
                                ->  Result  (cost=697.49..42599.98 rows=2389 width=1275) (actual time=4.004..4.176 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=174
                                      ->  Nested Loop Anti Join  (cost=697.49..42599.98 rows=2389 width=1267) (actual time=4.002..4.164 rows=51 loops=1)
                                            Join Filter: (acl_share_1.record_id = r_3.id)
                                            Buffers: shared hit=174
                                            ->  Index Scan using records_list_updated on records r_3  (cost=688.88..42548.05 rows=2389 width=1267) (actual time=3.988..4.088 rows=51 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((hashed SubPlan 1) IS TRUE))
                                                  Buffers: shared hit=170
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.012..2.112 rows=10111 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=116
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.011..1.158 rows=10111 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=116
                                            ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=51)
                                                  Buffers: shared hit=4
                                                  ->  Result  (cost=8.60..16.08 rows=1 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (actual time=0.007..0.008 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.226 ms
Execution Time: 4.717 ms
```

#### bench_conc at root (|S| 10,111), ragent-1
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=63.54..63.55 rows=1 width=16) (actual time=1.165..1.167 rows=1 loops=1)
  Buffers: shared hit=36
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=54.33..54.34 rows=1 width=8) (never executed)
          ->  Limit  (cost=1.28..52.06 rows=182 width=4) (never executed)
                ->  Result  (cost=1.28..52.06 rows=182 width=4) (never executed)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      ->  Nested Loop  (cost=1.28..52.06 rows=182 width=0) (never executed)
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                              Group Key: acl_ud_1.id
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Heap Fetches: 0
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                            ->  Limit  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                  ->  Result  (cost=0.42..5.62 rows=26 width=4) (never executed)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=26 width=0) (never executed)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=1.163..1.165 rows=1 loops=1)
        Buffers: shared hit=36
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.068..1.017 rows=2500 loops=1)
              Buffers: shared hit=36
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.067..0.840 rows=2500 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=36
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.036..0.554 rows=2500 loops=1)
                          Buffers: shared hit=36
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.029..0.029 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.007..0.008 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.021..0.021 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.006..0.304 rows=2500 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=32
Planning Time: 0.595 ms
Execution Time: 1.231 ms
```

count:
```
Aggregate  (cost=8240.83..8240.84 rows=1 width=8) (actual time=0.749..0.752 rows=1 loops=1)
  Buffers: shared hit=29
  ->  Limit  (cost=0.55..8228.31 rows=1001 width=4) (actual time=0.027..0.679 rows=1001 loops=1)
        Buffers: shared hit=29
        ->  Result  (cost=0.55..11105.16 rows=1351 width=4) (actual time=0.026..0.608 rows=1001 loops=1)
              Buffers: shared hit=29
              ->  Append  (cost=0.55..11091.65 rows=1351 width=0) (actual time=0.026..0.486 rows=1001 loops=1)
                    Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..34.61 rows=338 width=0) (actual time=0.025..0.412 rows=1001 loops=1)
                          Buffers: shared hit=29
                          ->  Limit  (cost=0.55..31.23 rows=338 width=4) (actual time=0.025..0.327 rows=1001 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.55..31.23 rows=338 width=4) (actual time=0.024..0.257 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=29
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..31.23 rows=338 width=0) (actual time=0.021..0.157 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..44.82 rows=10 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.72 rows=10 width=4) (never executed)
                                ->  Result  (cost=0.55..44.72 rows=10 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.72 rows=10 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..8.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=714.67..10944.69 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=714.67..10934.68 rows=1001 width=4) (never executed)
                                ->  Result  (cost=714.67..24319.72 rows=2312 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop Anti Join  (cost=714.67..24319.72 rows=2312 width=0) (never executed)
                                            Join Filter: (acl_share_1.record_id = r_4.id)
                                            ->  Index Scan using records_org_unit on records r_4  (cost=688.88..24241.34 rows=2312 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (never executed)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                            ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                  ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 0.864 ms
Execution Time: 0.834 ms
```

page:
```
Nested Loop Left Join  (cost=2091.08..2561.66 rows=1 width=1317) (actual time=6.598..7.135 rows=51 loops=1)
  Buffers: shared hit=1766
  ->  Nested Loop  (cost=2090.79..2553.33 rows=1 width=1282) (actual time=6.583..6.980 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=1613
        ->  Subquery Scan on x  (cost=2090.50..2545.01 rows=1 width=1275) (actual time=6.570..6.846 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1409
              ->  Limit  (cost=2090.50..2543.87 rows=51 width=1275) (actual time=6.569..6.817 rows=51 loops=1)
                    Buffers: shared hit=1409
                    ->  Merge Append  (cost=2090.50..3103.90 rows=114 width=1275) (actual time=6.567..6.809 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1409
                          ->  Limit  (cost=1347.99..1348.12 rows=51 width=1275) (actual time=2.497..2.500 rows=3 loops=1)
                                Buffers: shared hit=1104
                                ->  Sort  (cost=1347.99..1348.84 rows=338 width=1275) (actual time=2.497..2.498 rows=3 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=1104
                                      ->  Result  (cost=27.95..1336.72 rows=338 width=1275) (actual time=0.329..1.745 rows=1236 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=1104
                                            ->  Bitmap Heap Scan on records r  (cost=27.95..1336.72 rows=338 width=1267) (actual time=0.325..1.384 rows=1236 loops=1)
                                                  Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                  Heap Blocks: exact=1069
                                                  Buffers: shared hit=1104
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..27.85 rows=338 width=0) (actual time=0.195..0.195 rows=1236 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                        Buffers: shared hit=35
                          ->  Limit  (cost=0.55..44.72 rows=10 width=1275) (actual time=0.030..0.031 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..44.72 rows=10 width=1275) (actual time=0.029..0.029 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.72 rows=10 width=1267) (actual time=0.020..0.020 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.014..0.014 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.013..0.014 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.011..0.011 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.066..0.070 rows=0 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.066..0.069 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.062..0.065 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.061..0.064 rows=0 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.036..0.038 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.026..0.030 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.007..0.007 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=689.17..1648.34 rows=51 width=1275) (actual time=3.955..4.180 rows=49 loops=1)
                                Buffers: shared hit=268
                                ->  Result  (cost=689.17..44171.21 rows=2312 width=1275) (actual time=3.954..4.174 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=268
                                      ->  Nested Loop Anti Join  (cost=689.17..44171.21 rows=2312 width=1267) (actual time=3.952..4.160 rows=49 loops=1)
                                            Buffers: shared hit=268
                                            ->  Index Scan using records_list_updated on records r_4  (cost=688.88..42615.13 rows=2312 width=1267) (actual time=3.939..4.019 rows=49 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND ((hashed SubPlan 1) IS TRUE))
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=170
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.013..1.998 rows=10111 loops=1)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=116
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..663.12 rows=10135 width=16) (actual time=0.012..1.124 rows=10111 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=116
                                            ->  Result  (cost=0.29..0.66 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=49)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  Buffers: shared hit=98
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.66 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=49)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=98
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.249 ms
Execution Time: 7.289 ms
```

#### sweep bench_conc, regions 0-0 (|S| 1011), region-only
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=63.54..63.55 rows=1 width=16) (actual time=2.923..2.928 rows=1 loops=1)
  Buffers: shared hit=3073
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=54.33..54.34 rows=1 width=8) (actual time=2.450..2.453 rows=1 loops=1)
          Buffers: shared hit=3053
          ->  Limit  (cost=1.28..52.06 rows=182 width=4) (actual time=2.448..2.451 rows=0 loops=1)
                Buffers: shared hit=3053
                ->  Result  (cost=1.28..52.06 rows=182 width=4) (actual time=2.448..2.450 rows=0 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=3053
                      ->  Nested Loop  (cost=1.28..52.06 rows=182 width=0) (actual time=2.447..2.449 rows=0 loops=1)
                            Buffers: shared hit=3053
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.011..0.224 rows=1011 loops=1)
                                  Buffers: shared hit=20
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.008..0.010 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.002..0.004 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.001..0.002 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.002..0.120 rows=1011 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=16
                            ->  Limit  (cost=0.42..5.62 rows=26 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                  Buffers: shared hit=3033
                                  ->  Result  (cost=0.42..5.62 rows=26 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=3033
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=26 width=0) (actual time=0.001..0.001 rows=0 loops=1011)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=3033
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=0.470..0.472 rows=1 loops=1)
        Buffers: shared hit=20
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.039..0.414 rows=1011 loops=1)
              Buffers: shared hit=20
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.038..0.341 rows=1011 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=20
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.033..0.237 rows=1011 loops=1)
                          Buffers: shared hit=20
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.027..0.029 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.020..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.005..0.121 rows=1011 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=16
Planning Time: 0.587 ms
Execution Time: 2.997 ms
```

count:
```
Aggregate  (cost=1724.01..1724.02 rows=1 width=8) (actual time=2.793..2.802 rows=1 loops=1)
  Buffers: shared hit=3065
  ->  Limit  (cost=0.55..1721.96 rows=164 width=4) (actual time=2.791..2.800 rows=0 loops=1)
        Buffers: shared hit=3065
        ->  Result  (cost=0.55..1721.96 rows=164 width=4) (actual time=2.790..2.799 rows=0 loops=1)
              Buffers: shared hit=3065
              ->  Append  (cost=0.55..1720.32 rows=164 width=0) (actual time=2.790..2.799 rows=0 loops=1)
                    Buffers: shared hit=3065
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..4.58 rows=1 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.57 rows=1 width=4) (actual time=0.020..0.020 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.57 rows=1 width=4) (actual time=0.019..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.57 rows=1 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..8.59 rows=1 width=0) (actual time=0.012..0.014 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (actual time=0.012..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.011..0.013 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.010..0.012 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.010..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.009..0.010 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=10.31..1681.76 rows=161 width=0) (actual time=2.746..2.750 rows=0 loops=1)
                          Buffers: shared hit=3053
                          ->  Limit  (cost=10.31..1680.15 rows=161 width=4) (actual time=2.745..2.749 rows=0 loops=1)
                                Buffers: shared hit=3053
                                ->  Result  (cost=10.31..1680.15 rows=161 width=4) (actual time=2.745..2.749 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=3053
                                      ->  Nested Loop  (cost=10.31..1680.15 rows=161 width=0) (actual time=2.745..2.748 rows=0 loops=1)
                                            Buffers: shared hit=3053
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.020..0.234 rows=1011 loops=1)
                                                  Buffers: shared hit=20
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.015..0.017 rows=1 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
                                                              Group Key: acl_ud.id
                                                              Batches: 1  Memory Usage: 24kB
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.003..0.003 rows=1 loops=1)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.009..0.010 rows=0 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Rows Removed by Filter: 1
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.004..0.124 rows=1011 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=16
                                            ->  Limit  (cost=9.45..238.26 rows=23 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                                  Buffers: shared hit=3033
                                                  ->  Result  (cost=9.45..238.26 rows=23 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=3033
                                                        ->  Nested Loop Anti Join  (cost=9.45..238.26 rows=23 width=0) (actual time=0.002..0.002 rows=0 loops=1011)
                                                              Join Filter: (acl_share_1.record_id = r_3.id)
                                                              Buffers: shared hit=3033
                                                              ->  Nested Loop Semi Join  (cost=0.84..221.82 rows=23 width=32) (actual time=0.001..0.001 rows=0 loops=1011)
                                                                    Buffers: shared hit=3033
                                                                    ->  Index Scan using records_org_unit on records r_3  (cost=0.42..217.09 rows=23 width=48) (actual time=0.001..0.001 rows=0 loops=1011)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)))
                                                                          Buffers: shared hit=3033
                                                                    ->  Materialize  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                                Heap Fetches: 0
                                                              ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (never executed)
                                                                    ->  Result  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                                                Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (never executed)
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 1.025 ms
Execution Time: 2.900 ms
```

page:
```
Nested Loop Left Join  (cost=2173.07..2190.41 rows=1 width=376) (actual time=1.546..1.554 rows=0 loops=1)
  Buffers: shared hit=3061
  ->  Nested Loop  (cost=2172.77..2182.08 rows=1 width=341) (actual time=1.546..1.553 rows=0 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3061
        ->  Subquery Scan on x  (cost=2172.49..2173.76 rows=1 width=334) (actual time=1.546..1.553 rows=0 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=3061
              ->  Limit  (cost=2172.49..2172.62 rows=51 width=334) (actual time=1.545..1.552 rows=0 loops=1)
                    Buffers: shared hit=3061
                    ->  Sort  (cost=2172.49..2172.62 rows=54 width=334) (actual time=1.544..1.551 rows=0 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Sort Method: quicksort  Memory: 25kB
                          Buffers: shared hit=3061
                          ->  Append  (cost=0.55..2170.93 rows=54 width=333) (actual time=1.541..1.548 rows=0 loops=1)
                                Buffers: shared hit=3061
                                ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.022..0.023 rows=0 loops=1)
                                      Buffers: shared hit=4
                                      ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.022..0.022 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.020..0.020 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                  Buffers: shared hit=4
                                ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.010..0.011 rows=0 loops=1)
                                      Buffers: shared hit=4
                                      ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.010..0.010 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.009..0.009 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                  Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                  Buffers: shared hit=4
                                ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.013..0.015 rows=0 loops=1)
                                      Buffers: shared hit=4
                                      ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.013..0.014 rows=0 loops=1)
                                            Sort Key: r_2.updated_at DESC, r_2.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=4
                                            ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.011..0.013 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  Buffers: shared hit=4
                                                  ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.011..0.012 rows=0 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.009..0.010 rows=0 loops=1)
                                                                    Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                    Buffers: shared hit=4
                                                                    ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.008..0.009 rows=0 loops=1)
                                                                          Buffers: shared hit=4
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                                Buffers: shared hit=2
                                                        ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                ->  Limit  (cost=2128.81..2128.94 rows=51 width=278) (actual time=1.495..1.498 rows=0 loops=1)
                                      Buffers: shared hit=3049
                                      CTE acl_m
                                        ->  Result  (cost=127.51..2116.59 rows=229 width=1275) (actual time=1.492..1.494 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=3049
                                              InitPlan 1 (returns $0)
                                                ->  Result  (cost=0.41..68.23 rows=991 width=16) (actual time=0.011..0.209 rows=1011 loops=1)
                                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                      Buffers: shared hit=16
                                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..68.23 rows=991 width=16) (actual time=0.010..0.122 rows=1011 loops=1)
                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])))
                                                            Heap Fetches: 0
                                                            Buffers: shared hit=16
                                              ->  Nested Loop Anti Join  (cost=59.27..2048.36 rows=229 width=1267) (actual time=1.491..1.493 rows=0 loops=1)
                                                    Join Filter: (acl_share_1.record_id = r_4.id)
                                                    Buffers: shared hit=3049
                                                    ->  Bitmap Heap Scan on records r_4  (cost=50.66..2028.83 rows=229 width=1267) (actual time=1.491..1.491 rows=0 loops=1)
                                                          Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                          Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)))
                                                          Buffers: shared hit=3049
                                                          ->  Bitmap Index Scan on records_org_unit  (cost=0.00..50.60 rows=514 width=0) (actual time=1.490..1.490 rows=0 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                                Buffers: shared hit=3049
                                                    ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (never executed)
                                                          ->  Result  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (never executed)
                                                                      Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                      ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (never executed)
                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                      ->  Sort  (cost=12.22..12.79 rows=229 width=278) (actual time=1.494..1.495 rows=0 loops=1)
                                            Sort Key: r_3.updated_at DESC, r_3.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=3049
                                            ->  CTE Scan on acl_m r_3  (cost=0.00..4.58 rows=229 width=278) (actual time=1.493..1.494 rows=0 loops=1)
                                                  Buffers: shared hit=3049
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (never executed)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (never executed)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
Planning Time: 1.148 ms
Execution Time: 1.677 ms
```

#### sweep bench_conc, regions 0-0 (|S| 1011), ragent-1
N̂_e 20000, K* 1000, alpha 8.0, cap ceil(N̂_e/alpha) = 2500

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=63.54..63.55 rows=1 width=16) (actual time=2.919..2.925 rows=1 loops=1)
  Buffers: shared hit=3073
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=54.33..54.34 rows=1 width=8) (actual time=2.457..2.460 rows=1 loops=1)
          Buffers: shared hit=3053
          ->  Limit  (cost=1.28..52.06 rows=182 width=4) (actual time=2.455..2.458 rows=0 loops=1)
                Buffers: shared hit=3053
                ->  Result  (cost=1.28..52.06 rows=182 width=4) (actual time=2.455..2.458 rows=0 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=3053
                      ->  Nested Loop  (cost=1.28..52.06 rows=182 width=0) (actual time=2.453..2.456 rows=0 loops=1)
                            Buffers: shared hit=3053
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.010..0.218 rows=1011 loops=1)
                                  Buffers: shared hit=20
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.007..0.009 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.002..0.003 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.001..0.002 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.004..0.005 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.002..0.115 rows=1011 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=16
                            ->  Limit  (cost=0.42..5.62 rows=26 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                  Buffers: shared hit=3033
                                  ->  Result  (cost=0.42..5.62 rows=26 width=4) (actual time=0.002..0.002 rows=0 loops=1011)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=3033
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..5.62 rows=26 width=0) (actual time=0.001..0.001 rows=0 loops=1011)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=3033
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=0.460..0.462 rows=1 loops=1)
        Buffers: shared hit=20
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.037..0.406 rows=1011 loops=1)
              Buffers: shared hit=20
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.036..0.335 rows=1011 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=20
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.031..0.230 rows=1011 loops=1)
                          Buffers: shared hit=20
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.025..0.027 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.003..0.003 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.016..0.016 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.005..0.118 rows=1011 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=16
Planning Time: 0.553 ms
Execution Time: 2.991 ms
```

count:
```
Aggregate  (cost=2031.55..2031.56 rows=1 width=8) (actual time=0.789..0.796 rows=1 loops=1)
  Buffers: shared hit=29
  ->  Limit  (cost=0.55..2025.16 rows=511 width=4) (actual time=0.028..0.718 rows=1001 loops=1)
        Buffers: shared hit=29
        ->  Result  (cost=0.55..2025.16 rows=511 width=4) (actual time=0.027..0.643 rows=1001 loops=1)
              Buffers: shared hit=29
              ->  Append  (cost=0.55..2020.05 rows=511 width=0) (actual time=0.027..0.541 rows=1001 loops=1)
                    Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..34.61 rows=338 width=0) (actual time=0.026..0.461 rows=1001 loops=1)
                          Buffers: shared hit=29
                          ->  Limit  (cost=0.55..31.23 rows=338 width=4) (actual time=0.026..0.351 rows=1001 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.55..31.23 rows=338 width=4) (actual time=0.026..0.279 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=29
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..31.23 rows=338 width=0) (actual time=0.021..0.173 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=29
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..44.82 rows=10 width=0) (never executed)
                          ->  Limit  (cost=0.55..44.72 rows=10 width=4) (never executed)
                                ->  Result  (cost=0.55..44.72 rows=10 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.72 rows=10 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..8.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.55..8.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=27.49..1877.29 rows=161 width=0) (never executed)
                          ->  Limit  (cost=27.49..1875.68 rows=161 width=4) (never executed)
                                ->  Result  (cost=27.49..1875.68 rows=161 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=27.49..1875.68 rows=161 width=0) (never executed)
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                                              Group Key: acl_ud.id
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Heap Fetches: 0
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                            ->  Limit  (cost=26.63..266.19 rows=23 width=4) (never executed)
                                                  ->  Result  (cost=26.63..266.19 rows=23 width=4) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Nested Loop Anti Join  (cost=26.63..266.19 rows=23 width=0) (never executed)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              ->  Nested Loop Semi Join  (cost=0.84..222.15 rows=23 width=32) (never executed)
                                                                    ->  Index Scan using records_org_unit on records r_4  (cost=0.42..217.42 rows=23 width=48) (never executed)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                    ->  Materialize  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                                Heap Fetches: 0
                                                              ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                                    ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                                Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 1.191 ms
Execution Time: 0.900 ms
```

page:
```
Nested Loop Left Join  (cost=3562.63..3604.16 rows=1 width=870) (actual time=4.010..4.294 rows=51 loops=1)
  Buffers: shared hit=4548
  ->  Nested Loop  (cost=3562.33..3595.83 rows=1 width=835) (actual time=3.996..4.175 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Rows Removed by Join Filter: 102
        Buffers: shared hit=4395
        ->  Subquery Scan on x  (cost=3562.05..3587.52 rows=1 width=828) (actual time=3.984..4.048 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4191
              ->  Limit  (cost=3562.05..3586.37 rows=51 width=828) (actual time=3.980..4.017 rows=51 loops=1)
                    Buffers: shared hit=4191
                    ->  Merge Append  (cost=3562.05..3616.41 rows=114 width=828) (actual time=3.979..4.012 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4191
                          ->  Limit  (cost=1347.99..1348.12 rows=51 width=1275) (actual time=2.377..2.388 rows=50 loops=1)
                                Buffers: shared hit=1104
                                ->  Sort  (cost=1347.99..1348.84 rows=338 width=1275) (actual time=2.377..2.383 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=1104
                                      ->  Result  (cost=27.95..1336.72 rows=338 width=1275) (actual time=0.343..1.665 rows=1236 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=1104
                                            ->  Bitmap Heap Scan on records r  (cost=27.95..1336.72 rows=338 width=1267) (actual time=0.338..1.326 rows=1236 loops=1)
                                                  Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                  Heap Blocks: exact=1069
                                                  Buffers: shared hit=1104
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..27.85 rows=338 width=0) (actual time=0.186..0.186 rows=1236 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                        Buffers: shared hit=35
                          ->  Limit  (cost=0.55..44.72 rows=10 width=1275) (actual time=0.026..0.030 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..44.72 rows=10 width=1275) (actual time=0.024..0.028 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.72 rows=10 width=1267) (actual time=0.019..0.022 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.013..0.014 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..8.58 rows=1 width=1275) (actual time=0.013..0.013 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..8.58 rows=1 width=1267) (actual time=0.011..0.011 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.061..0.063 rows=0 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.061..0.063 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.058..0.060 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.056..0.058 rows=0 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.033..0.034 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.024..0.028 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.007..0.007 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
                          ->  Sort  (cost=2160.72..2160.85 rows=51 width=278) (actual time=1.498..1.501 rows=0 loops=1)
                                Sort Key: r_4.sort_key DESC, r_4.id DESC
                                Sort Method: quicksort  Memory: 25kB
                                Buffers: shared hit=3049
                                ->  Limit  (cost=2159.15..2159.27 rows=51 width=278) (actual time=1.497..1.500 rows=0 loops=1)
                                      Buffers: shared hit=3049
                                      CTE acl_m
                                        ->  Result  (cost=144.69..2147.30 rows=222 width=1275) (actual time=1.494..1.496 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=3049
                                              InitPlan 1 (returns $0)
                                                ->  Result  (cost=0.41..68.23 rows=991 width=16) (actual time=0.011..0.207 rows=1011 loops=1)
                                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                      Buffers: shared hit=16
                                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..68.23 rows=991 width=16) (actual time=0.010..0.120 rows=1011 loops=1)
                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{03f6e003-695a-a286-6105-1eed022a02f3}'::uuid[])))
                                                            Heap Fetches: 0
                                                            Buffers: shared hit=16
                                              ->  Nested Loop Anti Join  (cost=76.45..2079.06 rows=222 width=1267) (actual time=1.493..1.494 rows=0 loops=1)
                                                    Join Filter: (acl_share_1.record_id = r_5.id)
                                                    Buffers: shared hit=3049
                                                    ->  Bitmap Heap Scan on records r_5  (cost=50.66..2032.04 rows=222 width=1267) (actual time=1.492..1.492 rows=0 loops=1)
                                                          Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                          Buffers: shared hit=3049
                                                          ->  Bitmap Index Scan on records_org_unit  (cost=0.00..50.60 rows=514 width=0) (actual time=1.491..1.491 rows=0 loops=1)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = ANY ($0)) AND (entity_definition_id = '01a202cf-ab2a-7ef3-9873-0c51fbc8dbe6'::uuid))
                                                                Buffers: shared hit=3049
                                                    ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                          ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                      Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                      Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                      ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                      ->  Sort  (cost=11.85..12.40 rows=222 width=278) (actual time=1.496..1.497 rows=0 loops=1)
                                            Sort Key: r_4.updated_at DESC, r_4.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=3049
                                            ->  CTE Scan on acl_m r_4  (cost=0.00..4.44 rows=222 width=278) (actual time=1.495..1.495 rows=0 loops=1)
                                                  Buffers: shared hit=3049
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=3 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=204
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.338 ms
Execution Time: 4.448 ms
```

#### root (|S| 10,111), region-only
N̂_e 400000, K* 3613, alpha 8.0, cap ceil(N̂_e/alpha) = 50000

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=323.84..323.85 rows=1 width=16) (actual time=6.954..6.960 rows=1 loops=1)
  Buffers: shared hit=586
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=314.63..314.64 rows=1 width=8) (actual time=2.467..2.469 rows=1 loops=1)
          Buffers: shared hit=466
          ->  Limit  (cost=1.28..270.88 rows=3500 width=4) (actual time=0.045..2.268 rows=3614 loops=1)
                Buffers: shared hit=466
                ->  Result  (cost=1.28..270.88 rows=3500 width=4) (actual time=0.045..1.983 rows=3614 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=466
                      ->  Nested Loop  (cost=1.28..270.88 rows=3500 width=0) (actual time=0.041..1.609 rows=3614 loops=1)
                            Buffers: shared hit=466
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.022..0.058 rows=139 loops=1)
                                  Buffers: shared hit=10
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.017..0.018 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.008..0.008 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.005 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.006..0.007 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.004..0.024 rows=139 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=6
                            ->  Limit  (cost=0.42..27.40 rows=500 width=4) (actual time=0.002..0.009 rows=26 loops=139)
                                  Buffers: shared hit=456
                                  ->  Result  (cost=0.42..27.40 rows=500 width=4) (actual time=0.002..0.007 rows=26 loops=139)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=456
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..27.40 rows=500 width=0) (actual time=0.001..0.003 rows=26 loops=139)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=456
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=4.485..4.487 rows=1 loops=1)
        Buffers: shared hit=120
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.042..3.918 rows=10111 loops=1)
              Buffers: shared hit=120
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.041..3.143 rows=10111 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=120
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.036..2.090 rows=10111 loops=1)
                          Buffers: shared hit=120
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.028..0.030 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.007..0.008 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.020..0.021 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.007..1.142 rows=10111 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=116
Planning Time: 0.598 ms
Execution Time: 7.025 ms
```

count:
```
Aggregate  (cost=8444.98..8444.99 rows=1 width=8) (actual time=2.255..2.266 rows=1 loops=1)
  Buffers: shared hit=417
  ->  Limit  (cost=0.55..8432.46 rows=1001 width=4) (actual time=0.203..2.181 rows=1001 loops=1)
        Buffers: shared hit=417
        ->  Result  (cost=0.55..8693.59 rows=1032 width=4) (actual time=0.202..2.085 rows=1001 loops=1)
              Buffers: shared hit=417
              ->  Append  (cost=0.55..8683.27 rows=1032 width=0) (actual time=0.201..1.964 rows=1001 loops=1)
                    Buffers: shared hit=417
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..5.14 rows=17 width=0) (actual time=0.040..0.041 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..4.97 rows=17 width=4) (actual time=0.039..0.039 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..4.97 rows=17 width=4) (actual time=0.038..0.039 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using records_list_requester on records r  (cost=0.55..4.97 rows=17 width=0) (actual time=0.030..0.030 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..56.86 rows=13 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.73 rows=13 width=4) (actual time=0.018..0.019 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.73 rows=13 width=4) (actual time=0.018..0.018 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..56.73 rows=13 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=16.51..24.57 rows=1 width=0) (actual time=0.022..0.025 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=16.51..24.56 rows=1 width=4) (actual time=0.022..0.025 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=16.51..24.56 rows=1 width=4) (actual time=0.022..0.024 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop  (cost=16.51..24.56 rows=1 width=0) (actual time=0.020..0.022 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.020..0.021 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.018..0.019 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=4
                                                        ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                              Buffers: shared hit=4
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.006 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                    Buffers: shared hit=2
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=27.19..8591.54 rows=1001 width=0) (actual time=0.119..1.781 rows=1001 loops=1)
                          Buffers: shared hit=405
                          ->  Limit  (cost=27.19..8581.53 rows=1001 width=4) (actual time=0.119..1.668 rows=1001 loops=1)
                                Buffers: shared hit=405
                                ->  Result  (cost=27.19..27245.56 rows=3185 width=4) (actual time=0.119..1.558 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=405
                                      ->  Nested Loop  (cost=27.19..27245.56 rows=3185 width=0) (actual time=0.117..1.448 rows=1001 loops=1)
                                            Buffers: shared hit=405
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.042..0.054 rows=25 loops=1)
                                                  Buffers: shared hit=9
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.030..0.031 rows=1 loops=1)
                                                        Buffers: shared hit=4
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.010..0.010 rows=1 loops=1)
                                                              Group Key: acl_ud.id
                                                              Batches: 1  Memory Usage: 24kB
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.006..0.006 rows=1 loops=1)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=1)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.017..0.017 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Rows Removed by Filter: 1
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (actual time=0.010..0.018 rows=25 loops=1)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Limit  (cost=26.33..3881.82 rows=455 width=4) (actual time=0.011..0.052 rows=40 loops=25)
                                                  Buffers: shared hit=396
                                                  ->  Result  (cost=26.33..3881.82 rows=455 width=4) (actual time=0.011..0.048 rows=40 loops=25)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        Buffers: shared hit=396
                                                        ->  Nested Loop Anti Join  (cost=26.33..3881.82 rows=455 width=0) (actual time=0.010..0.043 rows=40 loops=25)
                                                              Join Filter: (acl_share_1.record_id = r_3.id)
                                                              Buffers: shared hit=396
                                                              ->  Nested Loop Semi Join  (cost=17.71..3858.90 rows=455 width=32) (actual time=0.009..0.035 rows=40 loops=25)
                                                                    Buffers: shared hit=392
                                                                    ->  Bitmap Heap Scan on records r_3  (cost=17.30..3848.77 rows=455 width=48) (actual time=0.006..0.023 rows=40 loops=25)
                                                                          Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)))
                                                                          Heap Blocks: exact=280
                                                                          Buffers: shared hit=355
                                                                          ->  Bitmap Index Scan on records_org_unit  (cost=0.00..17.19 rows=1021 width=0) (actual time=0.003..0.003 rows=44 loops=25)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                                                Buffers: shared hit=75
                                                                    ->  Materialize  (cost=0.41..4.44 rows=1 width=16) (actual time=0.000..0.000 rows=1 loops=1001)
                                                                          Buffers: shared hit=37
                                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=12)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                                Heap Fetches: 0
                                                                                Buffers: shared hit=37
                                                              ->  Materialize  (cost=8.60..16.09 rows=1 width=32) (actual time=0.000..0.000 rows=0 loops=1001)
                                                                    Buffers: shared hit=4
                                                                    ->  Result  (cost=8.60..16.08 rows=1 width=32) (actual time=0.009..0.010 rows=0 loops=1)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                          Buffers: shared hit=4
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=8.60..16.08 rows=1 width=32) (actual time=0.007..0.008 rows=0 loops=1)
                                                                                Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Buffers: shared hit=4
                                                                                ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                      Buffers: shared hit=4
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                                            Buffers: shared hit=2
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                                            Buffers: shared hit=2
Planning Time: 1.078 ms
Execution Time: 2.448 ms
```

page:
```
Nested Loop Left Join  (cost=27.01..1099.18 rows=1 width=1317) (actual time=0.143..0.833 rows=51 loops=1)
  Buffers: shared hit=629
  ->  Nested Loop  (cost=26.71..1090.84 rows=1 width=1282) (actual time=0.128..0.680 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=476
        ->  Subquery Scan on x  (cost=26.43..1082.53 rows=1 width=1275) (actual time=0.120..0.585 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=323
              ->  Limit  (cost=26.43..1081.38 rows=51 width=1275) (actual time=0.119..0.557 rows=51 loops=1)
                    Buffers: shared hit=323
                    ->  Merge Append  (cost=26.43..1722.62 rows=82 width=1275) (actual time=0.118..0.551 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=323
                          ->  Limit  (cost=0.55..72.73 rows=17 width=1275) (actual time=0.049..0.050 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..72.73 rows=17 width=1275) (actual time=0.049..0.049 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..72.73 rows=17 width=1267) (actual time=0.019..0.019 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.73 rows=13 width=1275) (actual time=0.012..0.013 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.73 rows=13 width=1275) (actual time=0.012..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..56.73 rows=13 width=1267) (actual time=0.010..0.010 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Filter: ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.016..0.019 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Sort  (cost=24.57..24.57 rows=1 width=1275) (actual time=0.016..0.018 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=4
                                      ->  Result  (cost=16.51..24.56 rows=1 width=1275) (actual time=0.013..0.015 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=4
                                            ->  Nested Loop  (cost=16.51..24.56 rows=1 width=1267) (actual time=0.012..0.014 rows=0 loops=1)
                                                  Buffers: shared hit=4
                                                  ->  HashAggregate  (cost=16.07..16.08 rows=1 width=32) (actual time=0.012..0.014 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=4
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.59..16.07 rows=1 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Buffers: shared hit=4
                                                              ->  BitmapOr  (cost=8.59..8.59 rows=2 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                    Buffers: shared hit=4
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1267) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                          ->  Limit  (cost=0.72..1567.32 rows=51 width=1275) (actual time=0.040..0.462 rows=51 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.72..1425670.08 rows=46412 width=1275) (actual time=0.040..0.456 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Anti Join  (cost=0.72..1425670.08 rows=46412 width=1267) (actual time=0.038..0.442 rows=51 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_updated on records r_3  (cost=0.42..1409848.18 rows=46412 width=1267) (actual time=0.031..0.291 rows=51 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) AND (SubPlan 1))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=209
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.003..0.003 rows=1 loops=51)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=154
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.003..0.003 rows=1 loops=51)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_3.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=154
                                            ->  Result  (cost=0.29..0.33 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=51)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  Buffers: shared hit=102
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.33 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=51)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_3.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e64f7b63-4ab5-31f5-8c76-1084aee640a7'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=102
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.053 ms
Execution Time: 0.956 ms
```

#### root (|S| 10,111), ragent-1
N̂_e 400000, K* 3613, alpha 8.0, cap ceil(N̂_e/alpha) = 50000

folded probe (|S| capped at N̂_e/alpha, P3 in CASE ELSE):
```
Subquery Scan on acl_s  (cost=323.84..323.85 rows=1 width=16) (actual time=6.599..6.603 rows=1 loops=1)
  Buffers: shared hit=586
  InitPlan 1 (returns $3)
    ->  Aggregate  (cost=314.63..314.64 rows=1 width=8) (actual time=2.325..2.327 rows=1 loops=1)
          Buffers: shared hit=466
          ->  Limit  (cost=1.28..270.88 rows=3500 width=4) (actual time=0.044..2.128 rows=3614 loops=1)
                Buffers: shared hit=466
                ->  Result  (cost=1.28..270.88 rows=3500 width=4) (actual time=0.043..1.860 rows=3614 loops=1)
                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                      Buffers: shared hit=466
                      ->  Nested Loop  (cost=1.28..270.88 rows=3500 width=0) (actual time=0.040..1.500 rows=3614 loops=1)
                            Buffers: shared hit=466
                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (actual time=0.020..0.053 rows=139 loops=1)
                                  Buffers: shared hit=10
                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.014..0.016 rows=1 loops=1)
                                        Buffers: shared hit=4
                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                                              Group Key: acl_ud_1.id
                                              Batches: 1  Memory Usage: 24kB
                                              ->  Function Scan on unnest acl_ud_1  (cost=0.00..0.01 rows=1 width=16) (actual time=0.003..0.004 rows=1 loops=1)
                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=1)
                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                              Buffers: shared hit=4
                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up_1  (cost=0.41..4.44 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud_1.id))
                                                    Filter: (ancestor_id <> acl_ud_1.id)
                                                    Rows Removed by Filter: 1
                                                    Heap Fetches: 0
                                                    Buffers: shared hit=4
                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.56 rows=7 width=32) (actual time=0.005..0.022 rows=139 loops=1)
                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud_1.id))
                                        Heap Fetches: 0
                                        Buffers: shared hit=6
                            ->  Limit  (cost=0.42..27.40 rows=500 width=4) (actual time=0.002..0.008 rows=26 loops=139)
                                  Buffers: shared hit=456
                                  ->  Result  (cost=0.42..27.40 rows=500 width=4) (actual time=0.002..0.006 rows=26 loops=139)
                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                        Buffers: shared hit=456
                                        ->  Index Only Scan using records_org_unit on records r  (cost=0.42..27.40 rows=500 width=0) (actual time=0.001..0.003 rows=26 loops=139)
                                              Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou_1.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                              Heap Fetches: 0
                                              Buffers: shared hit=456
  ->  Aggregate  (cost=9.20..9.21 rows=1 width=8) (actual time=4.271..4.273 rows=1 loops=1)
        Buffers: shared hit=120
        ->  Limit  (cost=0.86..9.11 rows=7 width=4) (actual time=0.040..3.704 rows=10111 loops=1)
              Buffers: shared hit=120
              ->  Result  (cost=0.86..9.11 rows=7 width=4) (actual time=0.039..3.000 rows=10111 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                    Buffers: shared hit=120
                    ->  Nested Loop  (cost=0.86..9.11 rows=7 width=0) (actual time=0.034..1.967 rows=10111 loops=1)
                          Buffers: shared hit=120
                          ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (actual time=0.026..0.028 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (actual time=0.007..0.008 rows=1 loops=1)
                                      Group Key: acl_ud.id
                                      Batches: 1  Memory Usage: 24kB
                                      ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=1)
                                ->  Result  (cost=0.41..4.44 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (actual time=0.017..0.017 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                            Filter: (ancestor_id <> acl_ud.id)
                                            Rows Removed by Filter: 1
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=16) (actual time=0.007..1.099 rows=10111 loops=1)
                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                Heap Fetches: 0
                                Buffers: shared hit=116
Planning Time: 0.547 ms
Execution Time: 6.668 ms
```

count:
```
Aggregate  (cost=4544.23..4544.24 rows=1 width=8) (actual time=0.805..0.809 rows=1 loops=1)
  Buffers: shared hit=567
  ->  Limit  (cost=0.55..4531.71 rows=1001 width=4) (actual time=0.031..0.734 rows=1001 loops=1)
        Buffers: shared hit=567
        ->  Result  (cost=0.55..9977.26 rows=2204 width=4) (actual time=0.030..0.653 rows=1001 loops=1)
              Buffers: shared hit=567
              ->  Append  (cost=0.55..9955.22 rows=2204 width=0) (actual time=0.030..0.544 rows=1001 loops=1)
                    Buffers: shared hit=567
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..84.68 rows=1001 width=0) (actual time=0.029..0.468 rows=1001 loops=1)
                          Buffers: shared hit=567
                          ->  Limit  (cost=0.55..74.67 rows=1001 width=4) (actual time=0.028..0.381 rows=1001 loops=1)
                                Buffers: shared hit=567
                                ->  Result  (cost=0.55..486.99 rows=6569 width=4) (actual time=0.028..0.311 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=567
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..486.99 rows=6569 width=0) (actual time=0.024..0.209 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=567
                    ->  Subquery Scan on "*SELECT* 2"  (cost=15.53..779.96 rows=188 width=0) (never executed)
                          ->  Limit  (cost=15.53..778.08 rows=188 width=4) (never executed)
                                ->  Result  (cost=15.53..778.08 rows=188 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Bitmap Heap Scan on records r_1  (cost=15.53..778.08 rows=188 width=0) (never executed)
                                            Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.48 rows=195 width=0) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..56.94 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.55..56.81 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.55..56.81 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.81 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=44.11..52.18 rows=1 width=0) (never executed)
                          ->  Limit  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                ->  Result  (cost=44.11..52.17 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.11..52.17 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 5"  (cost=44.36..8970.44 rows=1001 width=0) (never executed)
                          ->  Limit  (cost=44.36..8960.43 rows=1001 width=4) (never executed)
                                ->  Result  (cost=44.36..27478.43 rows=3080 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      ->  Nested Loop  (cost=44.36..27478.43 rows=3080 width=0) (never executed)
                                            ->  Nested Loop  (cost=0.84..9.10 rows=7 width=16) (never executed)
                                                  ->  Nested Loop Anti Join  (cost=0.43..4.47 rows=1 width=16) (never executed)
                                                        ->  HashAggregate  (cost=0.02..0.03 rows=1 width=16) (never executed)
                                                              Group Key: acl_ud.id
                                                              ->  Function Scan on unnest acl_ud  (cost=0.00..0.01 rows=1 width=16) (never executed)
                                                        ->  Result  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                              ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_up  (cost=0.41..4.44 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ud.id))
                                                                    Filter: (ancestor_id <> acl_ud.id)
                                                                    Heap Fetches: 0
                                                  ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.56 rows=7 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = acl_ud.id))
                                                        Heap Fetches: 0
                                            ->  Limit  (cost=43.51..3915.39 rows=440 width=4) (never executed)
                                                  ->  Result  (cost=43.51..3915.39 rows=440 width=4) (never executed)
                                                        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                        ->  Nested Loop Anti Join  (cost=43.51..3915.39 rows=440 width=0) (never executed)
                                                              Join Filter: (acl_share_1.record_id = r_4.id)
                                                              ->  Nested Loop Semi Join  (cost=17.71..3865.09 rows=440 width=32) (never executed)
                                                                    ->  Bitmap Heap Scan on records r_4  (cost=17.30..3855.15 rows=440 width=48) (never executed)
                                                                          Recheck Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                                          Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                          ->  Bitmap Index Scan on records_org_unit  (cost=0.00..17.19 rows=1021 width=0) (never executed)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (org_unit_id = acl_ou.descendant_id) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                                    ->  Materialize  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou_1  (cost=0.41..4.44 rows=1 width=16) (never executed)
                                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = acl_ou.descendant_id))
                                                                                Heap Fetches: 0
                                                              ->  Materialize  (cost=25.78..43.69 rows=1 width=32) (never executed)
                                                                    ->  Result  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                                          ->  Bitmap Heap Scan on record_shares acl_share_1  (cost=25.78..43.68 rows=1 width=32) (never executed)
                                                                                Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                                                ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (never executed)
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (never executed)
                                                                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
Planning Time: 1.149 ms
Execution Time: 0.938 ms
```

page:
```
Nested Loop Left Join  (cost=55.06..1800.53 rows=1 width=1317) (actual time=0.228..0.939 rows=51 loops=1)
  Buffers: shared hit=714
  ->  Nested Loop  (cost=54.76..1792.19 rows=1 width=1282) (actual time=0.215..0.784 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=561
        ->  Subquery Scan on x  (cost=54.48..1783.88 rows=1 width=1275) (actual time=0.207..0.685 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=408
              ->  Limit  (cost=54.48..1782.73 rows=51 width=1275) (actual time=0.206..0.656 rows=51 loops=1)
                    Buffers: shared hit=408
                    ->  Merge Append  (cost=54.48..5713.65 rows=167 width=1275) (actual time=0.205..0.650 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=408
                          ->  Limit  (cost=0.42..3764.74 rows=51 width=1275) (actual time=0.080..0.122 rows=2 loops=1)
                                Buffers: shared hit=65
                                ->  Result  (cost=0.42..484859.00 rows=6569 width=1275) (actual time=0.080..0.121 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=65
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..484859.00 rows=6569 width=1267) (actual time=0.076..0.117 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Rows Removed by Filter: 60
                                            Buffers: shared hit=65
                          ->  Limit  (cost=0.55..216.26 rows=51 width=1275) (actual time=0.015..0.015 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..795.71 rows=188 width=1275) (actual time=0.015..0.015 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..795.71 rows=188 width=1267) (actual time=0.013..0.013 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=0.55..56.81 rows=13 width=1275) (actual time=0.014..0.014 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..56.81 rows=13 width=1275) (actual time=0.013..0.014 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.81 rows=13 width=1267) (actual time=0.012..0.012 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.070..0.071 rows=1 loops=1)
                                Buffers: shared hit=27
                                ->  Sort  (cost=52.18..52.18 rows=1 width=1275) (actual time=0.070..0.071 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=27
                                      ->  Result  (cost=44.11..52.17 rows=1 width=1275) (actual time=0.048..0.064 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                            Buffers: shared hit=27
                                            ->  Nested Loop  (cost=44.11..52.17 rows=1 width=1267) (actual time=0.047..0.062 rows=3 loops=1)
                                                  Buffers: shared hit=27
                                                  ->  HashAggregate  (cost=43.68..43.69 rows=1 width=32) (actual time=0.035..0.037 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=15
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=25.77..43.67 rows=1 width=32) (actual time=0.027..0.032 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=15
                                                              ->  BitmapOr  (cost=25.77..25.77 rows=5 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                    Buffers: shared hit=12
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..21.48 rows=4 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[])))
                                                                          Buffers: shared hit=10
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Buffers: shared hit=12
                          ->  Limit  (cost=0.72..1620.83 rows=51 width=1275) (actual time=0.025..0.416 rows=50 loops=1)
                                Buffers: shared hit=306
                                ->  Result  (cost=0.72..1426712.18 rows=44912 width=1275) (actual time=0.025..0.410 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                      Buffers: shared hit=306
                                      ->  Nested Loop Anti Join  (cost=0.72..1426712.18 rows=44912 width=1267) (actual time=0.023..0.397 rows=50 loops=1)
                                            Buffers: shared hit=306
                                            ->  Index Scan using records_list_updated on records r_4  (cost=0.42..1411151.35 rows=44912 width=1267) (actual time=0.017..0.251 rows=50 loops=1)
                                                  Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (entity_definition_id = '01a202cd-46ce-7817-8c1e-d58dadbee392'::uuid))
                                                  Filter: ((org_unit_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))) AND (SubPlan 1))
                                                  Rows Removed by Filter: 2
                                                  Buffers: shared hit=206
                                                  SubPlan 1
                                                    ->  Result  (cost=0.41..4.44 rows=1 width=4) (actual time=0.004..0.004 rows=1 loops=50)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                          Buffers: shared hit=151
                                                          ->  Index Only Scan using org_unit_closure_pkey on org_unit_closure acl_ou  (cost=0.41..4.44 rows=1 width=0) (actual time=0.002..0.002 rows=1 loops=50)
                                                                Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (ancestor_id = ANY ('{6e2fac92-2d82-4b0b-0c75-2f7f2f495e8b}'::uuid[])) AND (descendant_id = r_4.org_unit_id))
                                                                Heap Fetches: 0
                                                                Buffers: shared hit=151
                                            ->  Result  (cost=0.29..0.34 rows=1 width=32) (actual time=0.002..0.003 rows=0 loops=50)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid)
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share_1  (cost=0.29..0.34 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=50)
                                                        Index Cond: ((tenant_id = '76922db5-dfa2-42e6-9fed-25e0595a66c6'::uuid) AND (record_id = r_4.id))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,1c0b5c58-47af-3e0f-9a58-aa49ede43a78}'::uuid[]))))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_pkey on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Planning Time: 1.228 ms
Execution Time: 1.063 ms
```

## Rows: handler p95 against 150 ms, paired base beside it (1 client)

| Row | Attempt | n | p50 | **p95** | Base n | Base p50 | Base p95 | Δp95 vs base | Floor p50 before / after | Gate | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| region: [A2] area (\|S\| 10), agents | 1 | 1386 | 25.35 | **35.16** | 1385 | 15.46 | 22.32 | +12.84 | 1.63 / 0.68 |  | pass |
| region: [A2] area (\|S\| 10), count-only | 1 | 1208 | 28.36 | **41.03** | 1207 | 18.69 | 27.58 | +13.45 | 1.09 / 1.09 |  | pass |
| region: [A2] district (\|S\| 101), agents | 1 | 992 | 40.14 | **56.16** | 991 | 17.71 | 26.65 | +29.52 | 1.07 / 0.53 |  | pass |
| region: [A2] district (\|S\| 101), count-only | 1 | 1072 | 37.29 | **52.02** | 1071 | 15.86 | 23.55 | +28.47 | 0.69 / 0.55 |  | pass |
| region: [A2] region (\|S\| 1,011), agents | 1 | 1287 | 25.90 | **48.89** | 1287 | 15.66 | 29.13 | +19.75 | 0.54 / 0.67 |  | pass |
| region: [A2] region (\|S\| 1,011), count-only | 1 | 1210 | 30.09 | **39.38** | 1209 | 17.61 | 23.83 | +15.55 | 0.87 / 0.67 |  | pass |
| region: [A2] root (\|S\| 10,111), agents | 1 | 1262 | 27.55 | **37.82** | 1261 | 17.36 | 24.34 | +13.47 | 0.90 / 1.07 |  | pass |
| region: [A2] root (\|S\| 10,111), count-only | 1 | 1307 | 27.07 | **36.14** | 1306 | 16.93 | 23.46 | +12.68 | 1.03 / 0.65 |  | pass |
| region: [A2] district hot, agents | 1 | 1263 | 27.82 | **39.82** | 1263 | 16.66 | 24.97 | +14.85 | 0.66 / 0.58 |  | pass |
| region: [A2] district hot, count-only | 1 | 1180 | 30.79 | **43.50** | 1180 | 17.40 | 25.41 | +18.09 | 0.54 / 0.62 |  | pass |
| region: [A2] district thin, agents | 1 | 1296 | 27.48 | **40.27** | 1295 | 16.02 | 24.17 | +16.10 | 0.55 / 1.06 |  | pass |
| region: [A2] district thin, count-only | 1 | 1419 | 24.96 | **35.58** | 1419 | 14.97 | 22.33 | +13.25 | 0.56 / 1.09 |  | pass |
| region: [A2] uniform m=1001, agents | 1 | 1244 | 28.42 | **45.21** | 1244 | 16.06 | 27.07 | +18.14 | 0.56 / 0.56 |  | pass |
| region: [A2] uniform m=1001, count-only | 1 | 1232 | 28.68 | **45.02** | 1231 | 16.44 | 26.67 | +18.35 | 0.54 / 0.55 |  | pass |
| region: [A2] uniform m=3000, agents | 1 | 1106 | 35.27 | **49.62** | 1105 | 15.80 | 23.97 | +25.64 | 0.57 / 0.60 |  | pass |
| region: [A2] uniform m=3000, count-only | 1 | 1093 | 35.89 | **51.65** | 1092 | 15.92 | 24.26 | +27.39 | 0.55 / 0.86 |  | pass |
| region: [A2] uniform m=4500, agents | 1 | 1275 | 28.21 | **41.38** | 1274 | 15.94 | 23.73 | +17.65 | 0.63 / 1.05 |  | pass |
| region: [A2] uniform m=4500, count-only | 1 | 990 | 38.08 | **66.05** | 990 | 15.80 | 28.37 | +37.68 | 1.08 / 1.09 |  | pass |
| region: [A2] uniform m=10000, agents | 1 | 1290 | 27.57 | **39.60** | 1289 | 16.03 | 23.98 | +15.62 | 0.61 / 0.56 |  | pass |
| region: [A2] uniform m=10000, count-only | 1 | 1204 | 31.92 | **43.14** | 1204 | 15.64 | 22.12 | +21.01 | 0.54 / 0.56 |  | pass |
| region: [A2] bench_rare at root, agents | 1 | 1494 | 24.07 | **34.36** | 1494 | 13.60 | 20.67 | +13.69 | 1.06 / 0.55 |  | pass |
| region: [A2] bench_rare at root, count-only | 1 | 1429 | 24.99 | **34.37** | 1428 | 14.92 | 21.44 | +12.94 | 0.55 / 1.08 |  | pass |
| region: [A2] bench_conc, regions 0-8 (\|S\| 9,099), agents | 1 | 722 | 60.86 | **79.58** | 722 | 18.12 | 26.06 | +53.52 | 1.09 / 0.55 |  | pass |
| region: [A2] bench_conc, regions 0-8 (\|S\| 9,099), count-only | 1 | 623 | 74.84 | **99.26** | 622 | 17.46 | 26.20 | +73.06 | 0.55 / 1.09 |  | pass |
| region: [A2] bench_conc at root (\|S\| 10,111), agents | 1 | 1259 | 27.26 | **41.86** | 1258 | 17.12 | 26.91 | +14.95 | 1.01 / 0.55 |  | pass |
| region: [A2] bench_conc at root (\|S\| 10,111), count-only | 1 | 1166 | 30.27 | **46.52** | 1166 | 17.73 | 28.70 | +17.82 | 0.53 / 1.10 |  | pass |
| region: [A2] sweep bench_conc, regions 0-0 (\|S\| 1011), agents | 1 | 1032 | 34.44 | **52.99** | 1031 | 19.22 | 31.29 | +21.71 | 1.12 / 0.76 |  | pass |
| region: [A2] sweep bench_conc, regions 0-0 (\|S\| 1011), count-only | 1 | 1332 | 26.32 | **38.17** | 1331 | 16.42 | 24.51 | +13.66 | 1.08 / 0.55 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-0 (\|S\| 1011), agents | 1 | 905 | 45.70 | **64.27** | 904 | 16.49 | 24.56 | +39.71 | 0.53 / 1.06 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-0 (\|S\| 1011), count-only | 1 | 707 | 60.59 | **91.49** | 706 | 18.43 | 29.00 | +62.49 | 0.57 / 0.54 |  | pass |
| region: [A2] sweep bench_conc, regions 0-1 (\|S\| 2022), agents | 1 | 1171 | 32.01 | **47.74** | 1170 | 15.70 | 24.94 | +22.80 | 0.53 / 0.54 |  | pass |
| region: [A2] sweep bench_conc, regions 0-1 (\|S\| 2022), count-only | 1 | 1277 | 29.74 | **40.42** | 1276 | 15.20 | 21.00 | +19.42 | 0.90 / 0.55 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-1 (\|S\| 2022), agents | 1 | 941 | 45.22 | **65.56** | 940 | 14.96 | 22.27 | +43.28 | 0.56 / 0.55 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-1 (\|S\| 2022), count-only | 1 | 874 | 51.63 | **66.54** | 873 | 14.89 | 19.84 | +46.70 | 0.54 / 0.54 |  | pass |
| region: [A2] sweep bench_conc, regions 0-2 (\|S\| 3033), agents | 1 | 851 | 44.90 | **72.12** | 851 | 20.06 | 34.78 | +37.34 | 0.56 / 2.25 |  | pass |
| region: [A2] sweep bench_conc, regions 0-2 (\|S\| 3033), count-only | 1 | 843 | 42.15 | **80.94** | 843 | 18.81 | 38.09 | +42.85 | 2.29 / 1.09 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-2 (\|S\| 3033), agents | 1 | 529 | 81.50 | **131.68** | 529 | 29.62 | 47.24 | +84.44 | 0.56 / 1.10 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-2 (\|S\| 3033), count-only | 1 | 534 | 81.45 | **121.80** | 533 | 24.05 | 36.80 | +84.99 | 1.08 / 0.56 |  | pass |
| region: [A2] sweep bench_conc, regions 0-3 (\|S\| 4044), agents | 1 | 756 | 52.18 | **81.45** | 756 | 22.24 | 35.63 | +45.82 | 0.55 / 0.64 |  | pass |
| region: [A2] sweep bench_conc, regions 0-3 (\|S\| 4044), count-only | 1 | 917 | 46.26 | **57.08** | 916 | 17.38 | 22.43 | +34.66 | 0.56 / 0.54 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-3 (\|S\| 4044), agents | 1 | 746 | 55.91 | **78.69** | 745 | 20.91 | 31.15 | +47.54 | 1.63 / 0.56 |  | pass |
| region: [R forced] sweep bench_conc, regions 0-3 (\|S\| 4044), count-only | 1 | 753 | 59.61 | **81.07** | 753 | 18.03 | 25.01 | +56.05 | 0.55 / 0.53 |  | pass |
| region: [R] area (\|S\| 10), agents | 1 | 1453 | 24.05 | **34.90** | 1452 | 14.49 | 21.68 | +13.21 | 0.56 / 0.54 |  | pass |
| region: [R] area (\|S\| 10), count-only | 1 | 1508 | 22.55 | **32.17** | 1508 | 14.46 | 21.80 | +10.37 | 0.55 / 0.55 |  | pass |
| region: [R] district (\|S\| 101), agents | 1 | 968 | 39.37 | **64.76** | 968 | 17.09 | 29.38 | +35.39 | 0.57 / 0.75 |  | pass |
| region: [R] district (\|S\| 101), count-only | 1 | 1083 | 36.05 | **53.88** | 1082 | 15.42 | 24.11 | +29.77 | 1.05 / 0.53 |  | pass |
| region: [R] region (\|S\| 1,011), agents | 1 | 1078 | 30.69 | **61.21** | 1078 | 18.20 | 37.57 | +23.64 | 0.56 / 0.55 |  | pass |
| region: [R] region (\|S\| 1,011), count-only | 1 | 1001 | 35.58 | **55.16** | 1000 | 20.51 | 34.01 | +21.15 | 0.56 / 1.09 |  | pass |
| region: [R] root (\|S\| 10,111), agents | 1 | 1043 | 35.11 | **47.85** | 1043 | 20.16 | 28.89 | +18.97 | 1.08 / 0.55 |  | pass |
| region: [R] root (\|S\| 10,111), count-only | 1 | 1251 | 28.12 | **43.97** | 1250 | 16.07 | 27.41 | +16.56 | 0.55 / 0.56 |  | pass |
| region: [R] district hot, agents | 1 | 1265 | 28.44 | **41.43** | 1265 | 16.38 | 25.29 | +16.14 | 0.53 / 0.54 |  | pass |
| region: [R] district hot, count-only | 1 | 1056 | 33.55 | **56.24** | 1056 | 18.80 | 32.26 | +23.98 | 0.54 / 0.87 |  | pass |
| region: [R] district thin, agents | 1 | 955 | 36.14 | **62.44** | 954 | 21.00 | 37.42 | +25.02 | 0.53 / 1.20 |  | pass |
| region: [R] district thin, count-only | 1 | 1111 | 30.67 | **54.90** | 1111 | 18.10 | 33.34 | +21.56 | 1.64 / 0.55 |  | pass |
| region: [R] uniform m=1001, agents | 1 | 1281 | 28.72 | **40.66** | 1280 | 15.56 | 23.17 | +17.49 | 0.57 / 0.84 |  | pass |
| region: [R] uniform m=1001, count-only | 1 | 974 | 35.89 | **63.46** | 973 | 19.88 | 35.78 | +27.68 | 0.87 / 0.55 |  | pass |
| region: [R] uniform m=3000, agents | 1 | 964 | 39.99 | **64.04** | 963 | 16.86 | 28.76 | +35.28 | 0.87 / 0.64 |  | pass |
| region: [R] uniform m=3000, count-only | 1 | 908 | 43.07 | **64.86** | 907 | 18.91 | 30.34 | +34.53 | 1.12 / 1.02 |  | pass |
| region: [R] uniform m=4500, agents | 1 | 1219 | 29.25 | **46.94** | 1218 | 16.26 | 26.14 | +20.80 | 1.05 / 0.68 |  | pass |
| region: [R] uniform m=4500, count-only | 1 | 988 | 39.75 | **62.81** | 987 | 16.38 | 27.40 | +35.41 | 0.74 / 0.56 |  | pass |
| region: [R] uniform m=10000, agents | 1 | 1448 | 25.16 | **34.75** | 1447 | 14.23 | 19.73 | +15.02 | 0.55 / 0.55 |  | pass |
| region: [R] uniform m=10000, count-only | 1 | 1262 | 30.68 | **41.25** | 1261 | 14.95 | 20.58 | +20.67 | 0.57 / 0.56 |  | pass |
| region: [R] bench_rare at root, agents | 1 | 1528 | 23.96 | **34.23** | 1527 | 13.56 | 19.75 | +14.48 | 1.08 / 1.05 |  | pass |
| region: [R] bench_rare at root, count-only | 1 | 1456 | 24.94 | **35.59** | 1455 | 14.61 | 22.00 | +13.59 | 1.08 / 0.56 |  | pass |
| region: [R] bench_conc, regions 0-8 (\|S\| 9,099), agents | 1 | 928 | 45.13 | **67.66** | 927 | 15.80 | 22.58 | +45.08 | 0.54 / 1.06 |  | pass |
| region: [R] bench_conc, regions 0-8 (\|S\| 9,099), count-only | 1 | 880 | 51.49 | **64.40** | 879 | 14.98 | 19.77 | +44.63 | 0.53 / 0.54 |  | pass |
| region: [R] bench_conc at root (\|S\| 10,111), agents | 1 | 1343 | 27.17 | **36.43** | 1342 | 15.44 | 21.82 | +14.61 | 0.55 / 0.54 |  | pass |
| region: [R] bench_conc at root (\|S\| 10,111), count-only | 1 | 825 | 43.61 | **75.85** | 824 | 24.80 | 42.95 | +32.90 | 0.55 / 2.85 |  | pass |
