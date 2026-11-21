# TICK-1022 A2-11 spikes: the count and skew plans kept (full capture 1.3 MB, regenerate with the harness)

### none before 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 1073741824) c
```

```
Aggregate  (cost=9937.24..9937.25 rows=1 width=8) (actual time=100.680..100.687 rows=1 loops=1)
  Buffers: shared hit=4983
  ->  Limit  (cost=9673.70..9717.63 rows=17569 width=28) (actual time=85.395..95.770 rows=26593 loops=1)
        Buffers: shared hit=4983
        ->  Sort  (cost=9673.70..9717.63 rows=17569 width=28) (actual time=85.392..90.062 rows=26593 loops=1)
              Sort Key: v.record_id
              Sort Method: quicksort  Memory: 2638kB
              Buffers: shared hit=4983
              ->  Result  (cost=1308.57..8435.02 rows=17569 width=28) (actual time=14.666..67.862 rows=26593 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                    Buffers: shared hit=4983
                    ->  Bitmap Heap Scan on rfv_num v  (cost=1308.57..8435.02 rows=17569 width=24) (actual time=14.640..47.719 rows=26593 loops=1)
                          Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                          Heap Blocks: exact=4537
                          Buffers: shared hit=4983
                          ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..1304.16 rows=17569 width=0) (actual time=9.683..9.684 rows=26593 loops=1)
                                Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                                Buffers: shared hit=446
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.689 ms
Execution Time: 100.881 ms
```

### none before 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9762.21..9762.22 rows=1 width=8) (actual time=111.416..111.434 rows=1 loops=1)
  Buffers: shared hit=18046
  ->  Limit  (cost=8655.19..9760.56 rows=132 width=4) (actual time=111.409..111.427 rows=0 loops=1)
        Buffers: shared hit=18046
        ->  Result  (cost=8655.19..9760.56 rows=132 width=4) (actual time=111.406..111.423 rows=0 loops=1)
              Buffers: shared hit=18046
              ->  Append  (cost=8655.19..9759.24 rows=132 width=0) (actual time=111.405..111.422 rows=0 loops=1)
                    Buffers: shared hit=18046
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8655.19..9364.55 rows=129 width=0) (actual time=111.241..111.246 rows=0 loops=1)
                          Buffers: shared hit=18032
                          ->  Limit  (cost=8655.19..9363.26 rows=129 width=4) (actual time=111.238..111.243 rows=0 loops=1)
                                Buffers: shared hit=18032
                                ->  Result  (cost=8655.19..9363.26 rows=129 width=4) (actual time=111.236..111.241 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=18032
                                      ->  Hash Semi Join  (cost=8655.19..9363.26 rows=129 width=0) (actual time=111.220..111.224 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=18032
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.065..25.795 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=8435.01..8435.01 rows=17569 width=32) (actual time=67.124..67.126 rows=26593 loops=1)
                                                  Buckets: 32768  Batches: 1  Memory Usage: 1919kB
                                                  Buffers: shared hit=4983
                                                  ->  Bitmap Heap Scan on rfv_num v  (cost=1308.55..8435.01 rows=17569 width=32) (actual time=20.150..50.244 rows=26593 loops=1)
                                                        Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                                                        Heap Blocks: exact=4537
                                                        Buffers: shared hit=4983
                                                        ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..1304.16 rows=17569 width=0) (actual time=17.670..17.670 rows=26593 loops=1)
                                                              Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                                                              Buffers: shared hit=446
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..194.80 rows=1 width=0) (actual time=0.074..0.077 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..194.79 rows=1 width=4) (actual time=0.072..0.075 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..194.79 rows=1 width=4) (actual time=0.069..0.071 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..194.79 rows=1 width=0) (actual time=0.046..0.047 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.045..0.045 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '10'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..166.06 rows=1 width=0) (actual time=0.035..0.037 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..166.05 rows=1 width=4) (actual time=0.034..0.035 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..166.05 rows=1 width=4) (actual time=0.033..0.034 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..166.05 rows=1 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.028..0.029 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '10'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.049..0.053 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.048..0.052 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.047..0.051 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.043..0.046 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.042..0.045 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.042..0.044 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.037..0.038 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '10'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.843 ms
Execution Time: 114.784 ms
```

### none A2 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=9937.24..9937.25 rows=1 width=8) (actual time=82.403..82.409 rows=1 loops=1)
  Buffers: shared hit=4983
  ->  Limit  (cost=9673.70..9717.63 rows=17569 width=28) (actual time=70.362..78.005 rows=20000 loops=1)
        Buffers: shared hit=4983
        ->  Sort  (cost=9673.70..9717.63 rows=17569 width=28) (actual time=70.358..73.304 rows=20000 loops=1)
              Sort Key: v.record_id
              Sort Method: quicksort  Memory: 2638kB
              Buffers: shared hit=4983
              ->  Result  (cost=1308.57..8435.02 rows=17569 width=28) (actual time=8.183..56.078 rows=26593 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                    Buffers: shared hit=4983
                    ->  Bitmap Heap Scan on rfv_num v  (cost=1308.57..8435.02 rows=17569 width=24) (actual time=8.164..35.079 rows=26593 loops=1)
                          Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                          Heap Blocks: exact=4537
                          Buffers: shared hit=4983
                          ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..1304.16 rows=17569 width=0) (actual time=6.373..6.374 rows=26593 loops=1)
                                Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                                Buffers: shared hit=446
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.348 ms
Execution Time: 82.557 ms
```

### none A2 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 30001)) u LIMIT 30001) c
```

```
Aggregate  (cost=1069.24..1069.25 rows=1 width=8) (actual time=86.454..86.496 rows=1 loops=1)
  Buffers: shared hit=13063
  ->  Limit  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.107..80.609 rows=24055 loops=1)
        Buffers: shared hit=13063
        ->  Result  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.104..73.663 rows=24055 loops=1)
              Buffers: shared hit=13063
              ->  Append  (cost=0.55..936.58 rows=5896 width=0) (actual time=0.103..50.834 rows=24055 loops=1)
                    Buffers: shared hit=13063
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..750.46 rows=5867 width=0) (actual time=0.101..44.760 rows=24055 loops=1)
                          Buffers: shared hit=13049
                          ->  Limit  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.099..37.715 rows=24055 loops=1)
                                Buffers: shared hit=13049
                                ->  Result  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.098..31.948 rows=24055 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13049
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=0) (actual time=0.080..24.366 rows=24055 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=13049
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..68.03 rows=15 width=0) (actual time=0.077..0.079 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..67.88 rows=15 width=4) (actual time=0.076..0.077 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..67.88 rows=15 width=4) (actual time=0.073..0.073 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=0) (actual time=0.048..0.049 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..56.19 rows=13 width=0) (actual time=0.032..0.060 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.06 rows=13 width=4) (actual time=0.031..0.059 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.06 rows=13 width=4) (actual time=0.030..0.057 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=0) (actual time=0.025..0.025 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..32.42 rows=1 width=0) (actual time=0.049..0.054 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..32.41 rows=1 width=4) (actual time=0.048..0.052 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..32.41 rows=1 width=4) (actual time=0.048..0.051 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..32.41 rows=1 width=0) (actual time=0.044..0.047 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.043..0.045 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.038..0.039 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.035..0.036 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.033 ms
Execution Time: 86.743 ms
```

### none A2 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17343.54..17343.55 rows=1 width=8) (actual time=464.571..464.589 rows=1 loops=1)
  Buffers: shared hit=109283
  ->  Limit  (cost=0.55..17331.03 rows=1001 width=4) (actual time=464.565..464.583 rows=0 loops=1)
        Buffers: shared hit=109283
        ->  Result  (cost=0.55..17590.73 rows=1016 width=4) (actual time=464.560..464.577 rows=0 loops=1)
              Buffers: shared hit=109283
              ->  Append  (cost=0.55..17580.57 rows=1016 width=0) (actual time=464.557..464.573 rows=0 loops=1)
                    Buffers: shared hit=109283
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..17165.41 rows=1001 width=0) (actual time=464.429..464.432 rows=0 loops=1)
                          Buffers: shared hit=109269
                          ->  Limit  (cost=0.55..17155.40 rows=1001 width=4) (actual time=464.426..464.429 rows=0 loops=1)
                                Buffers: shared hit=109269
                                ->  Result  (cost=0.55..50282.61 rows=2934 width=4) (actual time=464.425..464.427 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=109269
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..50282.61 rows=2934 width=0) (actual time=464.404..464.405 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Filter: (SubPlan 1)
                                            Rows Removed by Filter: 24055
                                            Heap Fetches: 0
                                            Buffers: shared hit=109269
                                            SubPlan 1
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (actual time=0.017..0.017 rows=0 loops=24055)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r.tenant_id)
                                                    Buffers: shared hit=96220
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..8.45 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=24055)
                                                          Index Cond: ((tenant_id = r.tenant_id) AND (record_id = r.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '10'::double precision)
                                                          Rows Removed by Filter: 1
                                                          Buffers: shared hit=96220
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..203.20 rows=8 width=0) (actual time=0.046..0.049 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..203.12 rows=8 width=4) (actual time=0.045..0.048 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..203.12 rows=8 width=4) (actual time=0.041..0.044 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..203.12 rows=8 width=0) (actual time=0.036..0.038 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND (SubPlan 2))
                                            Buffers: shared hit=4
                                            SubPlan 2
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_1.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_1.tenant_id) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '10'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..166.00 rows=6 width=0) (actual time=0.028..0.031 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..165.94 rows=6 width=4) (actual time=0.027..0.029 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..165.94 rows=6 width=4) (actual time=0.026..0.028 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..165.94 rows=6 width=0) (actual time=0.022..0.024 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (SubPlan 3))
                                            Buffers: shared hit=4
                                            SubPlan 3
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_2.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_2.tenant_id) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '10'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..40.87 rows=1 width=0) (actual time=0.048..0.053 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..40.86 rows=1 width=4) (actual time=0.047..0.051 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..40.86 rows=1 width=4) (actual time=0.046..0.050 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..40.86 rows=1 width=0) (actual time=0.043..0.047 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.041..0.043 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.036..0.037 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.032..0.033 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..16.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (SubPlan 4))
                                                  SubPlan 4
                                                    ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_3.tenant_id)
                                                          ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = r_3.tenant_id) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                                Filter: ((value + '0'::double precision) = '10'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.013 ms
Execution Time: 465.002 ms
```

### none before 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 1073741824) c
```

```
Aggregate  (cost=9209.52..9209.53 rows=1 width=8) (actual time=44.208..44.213 rows=1 loops=1)
  Buffers: shared hit=1054
  ->  Limit  (cost=9007.13..9040.86 rows=13493 width=28) (actual time=32.917..41.091 rows=19999 loops=1)
        Buffers: shared hit=1054
        ->  Sort  (cost=9007.13..9040.86 rows=13493 width=28) (actual time=32.911..35.761 rows=19999 loops=1)
              Sort Key: v.record_id
              Sort Method: quicksort  Memory: 2175kB
              Buffers: shared hit=1054
              ->  Result  (cost=1008.60..8081.51 rows=13493 width=28) (actual time=4.280..22.706 rows=19999 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                    Buffers: shared hit=1054
                    ->  Bitmap Heap Scan on rfv_num v  (cost=1008.60..8081.51 rows=13493 width=24) (actual time=4.256..14.558 rows=19999 loops=1)
                          Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                          Heap Blocks: exact=721
                          Buffers: shared hit=1054
                          ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..1005.21 rows=13493 width=0) (actual time=4.036..4.037 rows=19999 loops=1)
                                Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                Buffers: shared hit=333
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.452 ms
Execution Time: 45.764 ms
```

### none before 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9356.29..9356.30 rows=1 width=8) (actual time=61.385..61.405 rows=1 loops=1)
  Buffers: shared hit=14117
  ->  Limit  (cost=8250.72..9355.01 rows=102 width=4) (actual time=61.379..61.398 rows=0 loops=1)
        Buffers: shared hit=14117
        ->  Result  (cost=8250.72..9355.01 rows=102 width=4) (actual time=61.376..61.395 rows=0 loops=1)
              Buffers: shared hit=14117
              ->  Append  (cost=8250.72..9353.99 rows=102 width=0) (actual time=61.375..61.393 rows=0 loops=1)
                    Buffers: shared hit=14117
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8250.72..8959.46 rows=99 width=0) (actual time=61.223..61.229 rows=0 loops=1)
                          Buffers: shared hit=14103
                          ->  Limit  (cost=8250.72..8958.47 rows=99 width=4) (actual time=61.221..61.226 rows=0 loops=1)
                                Buffers: shared hit=14103
                                ->  Result  (cost=8250.72..8958.47 rows=99 width=4) (actual time=61.219..61.224 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=14103
                                      ->  Hash Semi Join  (cost=8250.72..8958.47 rows=99 width=0) (actual time=61.201..61.205 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=14103
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.069..17.046 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=8081.50..8081.50 rows=13493 width=32) (actual time=30.279..30.281 rows=19999 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1506kB
                                                  Buffers: shared hit=1054
                                                  ->  Bitmap Heap Scan on rfv_num v  (cost=1008.59..8081.50 rows=13493 width=32) (actual time=3.989..19.409 rows=19999 loops=1)
                                                        Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                                        Heap Blocks: exact=721
                                                        Buffers: shared hit=1054
                                                        ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..1005.21 rows=13493 width=0) (actual time=3.750..3.750 rows=19999 loops=1)
                                                              Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                                              Buffers: shared hit=333
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..194.80 rows=1 width=0) (actual time=0.072..0.075 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..194.79 rows=1 width=4) (actual time=0.070..0.072 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..194.79 rows=1 width=4) (actual time=0.067..0.069 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..194.79 rows=1 width=0) (actual time=0.046..0.047 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.044..0.045 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..166.06 rows=1 width=0) (actual time=0.029..0.031 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..166.05 rows=1 width=4) (actual time=0.028..0.029 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..166.05 rows=1 width=4) (actual time=0.027..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..166.05 rows=1 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.023..0.023 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.046..0.051 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.045..0.050 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.044..0.049 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.040..0.044 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.039..0.042 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.038..0.040 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.034..0.035 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.948 ms
Execution Time: 61.753 ms
```

### none A2 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=9209.52..9209.53 rows=1 width=8) (actual time=72.480..72.485 rows=1 loops=1)
  Buffers: shared hit=1054
  ->  Limit  (cost=9007.13..9040.86 rows=13493 width=28) (actual time=42.708..70.058 rows=19999 loops=1)
        Buffers: shared hit=1054
        ->  Sort  (cost=9007.13..9040.86 rows=13493 width=28) (actual time=42.704..54.361 rows=19999 loops=1)
              Sort Key: v.record_id
              Sort Method: quicksort  Memory: 2175kB
              Buffers: shared hit=1054
              ->  Result  (cost=1008.60..8081.51 rows=13493 width=28) (actual time=4.519..24.432 rows=19999 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                    Buffers: shared hit=1054
                    ->  Bitmap Heap Scan on rfv_num v  (cost=1008.60..8081.51 rows=13493 width=24) (actual time=4.501..16.062 rows=19999 loops=1)
                          Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                          Heap Blocks: exact=721
                          Buffers: shared hit=1054
                          ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..1005.21 rows=13493 width=0) (actual time=4.326..4.326 rows=19999 loops=1)
                                Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                Buffers: shared hit=333
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.319 ms
Execution Time: 72.577 ms
```

### none A2 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9356.29..9356.30 rows=1 width=8) (actual time=80.368..80.387 rows=1 loops=1)
  Buffers: shared hit=14117
  ->  Limit  (cost=8250.72..9355.01 rows=102 width=4) (actual time=80.361..80.379 rows=0 loops=1)
        Buffers: shared hit=14117
        ->  Result  (cost=8250.72..9355.01 rows=102 width=4) (actual time=80.358..80.375 rows=0 loops=1)
              Buffers: shared hit=14117
              ->  Append  (cost=8250.72..9353.99 rows=102 width=0) (actual time=80.357..80.373 rows=0 loops=1)
                    Buffers: shared hit=14117
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8250.72..8959.46 rows=99 width=0) (actual time=80.198..80.204 rows=0 loops=1)
                          Buffers: shared hit=14103
                          ->  Limit  (cost=8250.72..8958.47 rows=99 width=4) (actual time=80.196..80.202 rows=0 loops=1)
                                Buffers: shared hit=14103
                                ->  Result  (cost=8250.72..8958.47 rows=99 width=4) (actual time=80.194..80.199 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=14103
                                      ->  Hash Semi Join  (cost=8250.72..8958.47 rows=99 width=0) (actual time=80.178..80.182 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=14103
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.067..21.988 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=8081.50..8081.50 rows=13493 width=32) (actual time=38.627..38.629 rows=19999 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1506kB
                                                  Buffers: shared hit=1054
                                                  ->  Bitmap Heap Scan on rfv_num v  (cost=1008.59..8081.50 rows=13493 width=32) (actual time=3.810..14.867 rows=19999 loops=1)
                                                        Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                                        Heap Blocks: exact=721
                                                        Buffers: shared hit=1054
                                                        ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..1005.21 rows=13493 width=0) (actual time=3.599..3.599 rows=19999 loops=1)
                                                              Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                                              Buffers: shared hit=333
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..194.80 rows=1 width=0) (actual time=0.074..0.077 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..194.79 rows=1 width=4) (actual time=0.072..0.074 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..194.79 rows=1 width=4) (actual time=0.069..0.071 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..194.79 rows=1 width=0) (actual time=0.046..0.048 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.045..0.046 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..166.06 rows=1 width=0) (actual time=0.031..0.033 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..166.05 rows=1 width=4) (actual time=0.029..0.030 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..166.05 rows=1 width=4) (actual time=0.028..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..166.05 rows=1 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.023..0.023 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.048..0.052 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.047..0.051 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.046..0.050 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.042..0.045 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.041..0.044 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.040..0.042 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.035..0.037 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.031..0.033 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.269 ms
Execution Time: 80.786 ms
```

### none before 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 1073741824) c
```

```
Aggregate  (cost=9185.73..9185.74 rows=1 width=8) (actual time=49.029..49.035 rows=1 loops=1)
  Buffers: shared hit=1055
  ->  Limit  (cost=8988.93..9021.73 rows=13120 width=28) (actual time=38.678..45.617 rows=20000 loops=1)
        Buffers: shared hit=1055
        ->  Sort  (cost=8988.93..9021.73 rows=13120 width=28) (actual time=38.675..41.464 rows=20000 loops=1)
              Sort Key: v.record_id
              Sort Method: quicksort  Memory: 2175kB
              Buffers: shared hit=1055
              ->  Result  (cost=979.84..8091.56 rows=13120 width=28) (actual time=4.388..27.842 rows=20000 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                    Buffers: shared hit=1055
                    ->  Bitmap Heap Scan on rfv_num v  (cost=979.84..8091.56 rows=13120 width=24) (actual time=4.369..14.803 rows=20000 loops=1)
                          Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                          Heap Blocks: exact=719
                          Buffers: shared hit=1055
                          ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..976.55 rows=13120 width=0) (actual time=4.098..4.101 rows=20000 loops=1)
                                Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                                Buffers: shared hit=336
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.438 ms
Execution Time: 49.128 ms
```

### none before 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9361.52..9361.53 rows=1 width=8) (actual time=137.199..137.219 rows=1 loops=1)
  Buffers: shared hit=14118
  ->  Limit  (cost=8256.11..9360.29 rows=99 width=4) (actual time=137.193..137.212 rows=0 loops=1)
        Buffers: shared hit=14118
        ->  Result  (cost=8256.11..9360.29 rows=99 width=4) (actual time=137.190..137.208 rows=0 loops=1)
              Buffers: shared hit=14118
              ->  Append  (cost=8256.11..9359.30 rows=99 width=0) (actual time=137.188..137.207 rows=0 loops=1)
                    Buffers: shared hit=14118
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8256.11..8964.78 rows=96 width=0) (actual time=137.040..137.047 rows=0 loops=1)
                          Buffers: shared hit=14104
                          ->  Limit  (cost=8256.11..8963.82 rows=96 width=4) (actual time=137.038..137.044 rows=0 loops=1)
                                Buffers: shared hit=14104
                                ->  Result  (cost=8256.11..8963.82 rows=96 width=4) (actual time=137.036..137.042 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=14104
                                      ->  Hash Semi Join  (cost=8256.11..8963.82 rows=96 width=0) (actual time=137.020..137.025 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=14104
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.068..21.358 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=8091.55..8091.55 rows=13120 width=32) (actual time=40.031..40.033 rows=20000 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1506kB
                                                  Buffers: shared hit=1055
                                                  ->  Bitmap Heap Scan on rfv_num v  (cost=979.83..8091.55 rows=13120 width=32) (actual time=3.895..22.691 rows=20000 loops=1)
                                                        Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                                                        Heap Blocks: exact=719
                                                        Buffers: shared hit=1055
                                                        ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..976.55 rows=13120 width=0) (actual time=3.660..3.660 rows=20000 loops=1)
                                                              Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                                                              Buffers: shared hit=336
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..194.80 rows=1 width=0) (actual time=0.070..0.072 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..194.79 rows=1 width=4) (actual time=0.068..0.070 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..194.79 rows=1 width=4) (actual time=0.066..0.067 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..194.79 rows=1 width=0) (actual time=0.043..0.045 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.042..0.042 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '31'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..166.06 rows=1 width=0) (actual time=0.028..0.030 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.99..166.05 rows=1 width=4) (actual time=0.027..0.029 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.99..166.05 rows=1 width=4) (actual time=0.026..0.027 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=0.99..166.05 rows=1 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.022..0.022 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '31'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.045..0.049 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.044..0.048 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.043..0.047 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.040..0.043 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.039..0.042 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.038..0.040 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.034..0.036 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '31'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.796 ms
Execution Time: 137.539 ms
```

### none A2 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=9185.73..9185.74 rows=1 width=8) (actual time=64.952..64.956 rows=1 loops=1)
  Buffers: shared hit=1055
  ->  Limit  (cost=8988.93..9021.73 rows=13120 width=28) (actual time=50.566..61.146 rows=20000 loops=1)
        Buffers: shared hit=1055
        ->  Sort  (cost=8988.93..9021.73 rows=13120 width=28) (actual time=50.562..55.265 rows=20000 loops=1)
              Sort Key: v.record_id
              Sort Method: quicksort  Memory: 2175kB
              Buffers: shared hit=1055
              ->  Result  (cost=979.84..8091.56 rows=13120 width=28) (actual time=14.899..38.658 rows=20000 loops=1)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                    Buffers: shared hit=1055
                    ->  Bitmap Heap Scan on rfv_num v  (cost=979.84..8091.56 rows=13120 width=24) (actual time=3.712..17.516 rows=20000 loops=1)
                          Recheck Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                          Heap Blocks: exact=719
                          Buffers: shared hit=1055
                          ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..976.55 rows=13120 width=0) (actual time=3.480..3.481 rows=20000 loops=1)
                                Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                                Buffers: shared hit=336
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.424 ms
Execution Time: 65.055 ms
```

### none A2 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 30001)) u LIMIT 30001) c
```

```
Aggregate  (cost=1069.24..1069.25 rows=1 width=8) (actual time=107.146..107.159 rows=1 loops=1)
  Buffers: shared hit=13063
  ->  Limit  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.088..99.513 rows=24055 loops=1)
        Buffers: shared hit=13063
        ->  Result  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.086..90.813 rows=24055 loops=1)
              Buffers: shared hit=13063
              ->  Append  (cost=0.55..936.58 rows=5896 width=0) (actual time=0.084..68.031 rows=24055 loops=1)
                    Buffers: shared hit=13063
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..750.46 rows=5867 width=0) (actual time=0.082..61.116 rows=24055 loops=1)
                          Buffers: shared hit=13049
                          ->  Limit  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.080..52.890 rows=24055 loops=1)
                                Buffers: shared hit=13049
                                ->  Result  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.080..46.236 rows=24055 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13049
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=0) (actual time=0.065..36.799 rows=24055 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=13049
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..68.03 rows=15 width=0) (actual time=0.126..0.127 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..67.88 rows=15 width=4) (actual time=0.125..0.126 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..67.88 rows=15 width=4) (actual time=0.122..0.123 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=0) (actual time=0.102..0.103 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..56.19 rows=13 width=0) (actual time=0.036..0.037 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.06 rows=13 width=4) (actual time=0.035..0.036 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.06 rows=13 width=4) (actual time=0.033..0.034 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=0) (actual time=0.027..0.027 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..32.42 rows=1 width=0) (actual time=0.046..0.051 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..32.41 rows=1 width=4) (actual time=0.045..0.049 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..32.41 rows=1 width=4) (actual time=0.044..0.048 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..32.41 rows=1 width=0) (actual time=0.041..0.045 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.040..0.042 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.035..0.037 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.032..0.033 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.450 ms
Execution Time: 107.426 ms
```

### none A2 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17343.54..17343.55 rows=1 width=8) (actual time=429.860..429.878 rows=1 loops=1)
  Buffers: shared hit=109283
  ->  Limit  (cost=0.55..17331.03 rows=1001 width=4) (actual time=429.853..429.870 rows=0 loops=1)
        Buffers: shared hit=109283
        ->  Result  (cost=0.55..17590.73 rows=1016 width=4) (actual time=429.850..429.867 rows=0 loops=1)
              Buffers: shared hit=109283
              ->  Append  (cost=0.55..17580.57 rows=1016 width=0) (actual time=429.849..429.865 rows=0 loops=1)
                    Buffers: shared hit=109283
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..17165.41 rows=1001 width=0) (actual time=429.704..429.707 rows=0 loops=1)
                          Buffers: shared hit=109269
                          ->  Limit  (cost=0.55..17155.40 rows=1001 width=4) (actual time=429.703..429.705 rows=0 loops=1)
                                Buffers: shared hit=109269
                                ->  Result  (cost=0.55..50282.61 rows=2934 width=4) (actual time=429.700..429.702 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=109269
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..50282.61 rows=2934 width=0) (actual time=429.687..429.688 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Filter: (SubPlan 1)
                                            Rows Removed by Filter: 24055
                                            Heap Fetches: 0
                                            Buffers: shared hit=109269
                                            SubPlan 1
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (actual time=0.016..0.016 rows=0 loops=24055)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r.tenant_id)
                                                    Buffers: shared hit=96220
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..8.45 rows=1 width=0) (actual time=0.013..0.013 rows=0 loops=24055)
                                                          Index Cond: ((tenant_id = r.tenant_id) AND (record_id = r.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '31'::double precision)
                                                          Rows Removed by Filter: 1
                                                          Buffers: shared hit=96220
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..203.20 rows=8 width=0) (actual time=0.054..0.057 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..203.12 rows=8 width=4) (actual time=0.052..0.054 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..203.12 rows=8 width=4) (actual time=0.047..0.049 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..203.12 rows=8 width=0) (actual time=0.041..0.043 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND (SubPlan 2))
                                            Buffers: shared hit=4
                                            SubPlan 2
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_1.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_1.tenant_id) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '31'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..166.00 rows=6 width=0) (actual time=0.036..0.039 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..165.94 rows=6 width=4) (actual time=0.035..0.037 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..165.94 rows=6 width=4) (actual time=0.034..0.036 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..165.94 rows=6 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (SubPlan 3))
                                            Buffers: shared hit=4
                                            SubPlan 3
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_2.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_2.tenant_id) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '31'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..40.87 rows=1 width=0) (actual time=0.046..0.052 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..40.86 rows=1 width=4) (actual time=0.045..0.051 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..40.86 rows=1 width=4) (actual time=0.045..0.049 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..40.86 rows=1 width=0) (actual time=0.041..0.044 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.039..0.041 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.032..0.034 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..16.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (SubPlan 4))
                                                  SubPlan 4
                                                    ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_3.tenant_id)
                                                          ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = r_3.tenant_id) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                                Filter: ((value + '0'::double precision) = '31'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.310 ms
Execution Time: 430.186 ms
```

### none S-A2-skew pre-count

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=3850.46..3850.47 rows=1 width=8) (actual time=13.804..13.806 rows=1 loops=1)
  Buffers: shared hit=43
  ->  Limit  (cost=0.55..3600.46 rows=20000 width=28) (actual time=0.342..13.781 rows=50 loops=1)
        Buffers: shared hit=43
        ->  Result  (cost=0.55..24033.56 rows=133520 width=28) (actual time=0.339..13.764 rows=50 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=43
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..24033.56 rows=133520 width=24) (actual time=0.322..13.726 rows=50 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '77'::double precision))
                    Heap Fetches: 50
                    Buffers: shared hit=43
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.742 ms
Execution Time: 13.937 ms
```

### vacuum before 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 1073741824) c
```

```
Aggregate  (cost=1767.12..1767.13 rows=1 width=8) (actual time=96.351..96.354 rows=1 loops=1)
  Buffers: shared hit=447
  ->  Limit  (cost=0.55..1539.06 rows=18245 width=28) (actual time=0.090..80.423 rows=26593 loops=1)
        Buffers: shared hit=447
        ->  Result  (cost=0.55..1539.06 rows=18245 width=28) (actual time=0.087..60.868 rows=26593 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=447
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1539.06 rows=18245 width=24) (actual time=0.071..24.650 rows=26593 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                    Heap Fetches: 0
                    Buffers: shared hit=447
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.405 ms
Execution Time: 96.441 ms
```

### vacuum before 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2766.45..2766.46 rows=1 width=8) (actual time=130.826..130.845 rows=1 loops=1)
  Buffers: shared hit=13510
  ->  Limit  (cost=1767.69..2764.74 rows=137 width=4) (actual time=130.819..130.837 rows=0 loops=1)
        Buffers: shared hit=13510
        ->  Result  (cost=1767.69..2764.74 rows=137 width=4) (actual time=130.817..130.834 rows=0 loops=1)
              Buffers: shared hit=13510
              ->  Append  (cost=1767.69..2763.37 rows=137 width=0) (actual time=130.815..130.832 rows=0 loops=1)
                    Buffers: shared hit=13510
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1767.69..2477.16 rows=134 width=0) (actual time=130.659..130.665 rows=0 loops=1)
                          Buffers: shared hit=13496
                          ->  Limit  (cost=1767.69..2475.82 rows=134 width=4) (actual time=130.609..130.614 rows=0 loops=1)
                                Buffers: shared hit=13496
                                ->  Result  (cost=1767.69..2475.82 rows=134 width=4) (actual time=130.607..130.612 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13496
                                      ->  Hash Semi Join  (cost=1767.69..2475.82 rows=134 width=0) (actual time=130.593..130.597 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=13496
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.061..50.427 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=1539.06..1539.06 rows=18245 width=32) (actual time=24.107..24.109 rows=26593 loops=1)
                                                  Buckets: 32768  Batches: 1  Memory Usage: 1919kB
                                                  Buffers: shared hit=447
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1539.06 rows=18245 width=32) (actual time=0.037..12.829 rows=26593 loops=1)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=447
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..136.67 rows=1 width=0) (actual time=0.069..0.072 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..136.66 rows=1 width=4) (actual time=0.068..0.069 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..136.66 rows=1 width=4) (actual time=0.065..0.066 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..136.66 rows=1 width=0) (actual time=0.044..0.046 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.043..0.043 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..115.69 rows=1 width=0) (actual time=0.033..0.035 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..115.68 rows=1 width=4) (actual time=0.031..0.033 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..115.68 rows=1 width=4) (actual time=0.030..0.032 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..115.68 rows=1 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.024..0.025 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.047..0.052 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.047..0.051 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.046..0.050 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.043..0.046 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.042..0.045 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.041..0.043 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.035..0.037 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.032..0.034 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '10'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.919 ms
Execution Time: 131.411 ms
```

### vacuum A2 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=1767.12..1767.13 rows=1 width=8) (actual time=36.450..36.453 rows=1 loops=1)
  Buffers: shared hit=338
  ->  Limit  (cost=0.55..1539.06 rows=18245 width=28) (actual time=0.084..32.713 rows=20000 loops=1)
        Buffers: shared hit=338
        ->  Result  (cost=0.55..1539.06 rows=18245 width=28) (actual time=0.081..17.864 rows=20000 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=338
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1539.06 rows=18245 width=24) (actual time=0.065..10.216 rows=20000 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '10'::double precision))
                    Heap Fetches: 0
                    Buffers: shared hit=338
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.601 ms
Execution Time: 36.548 ms
```

### vacuum A2 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 30001)) u LIMIT 30001) c
```

```
Aggregate  (cost=1069.24..1069.25 rows=1 width=8) (actual time=70.191..70.203 rows=1 loops=1)
  Buffers: shared hit=13063
  ->  Limit  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.097..56.165 rows=24055 loops=1)
        Buffers: shared hit=13063
        ->  Result  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.095..49.996 rows=24055 loops=1)
              Buffers: shared hit=13063
              ->  Append  (cost=0.55..936.58 rows=5896 width=0) (actual time=0.093..42.676 rows=24055 loops=1)
                    Buffers: shared hit=13063
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..750.46 rows=5867 width=0) (actual time=0.092..36.872 rows=24055 loops=1)
                          Buffers: shared hit=13049
                          ->  Limit  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.090..30.331 rows=24055 loops=1)
                                Buffers: shared hit=13049
                                ->  Result  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.090..25.209 rows=24055 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13049
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=0) (actual time=0.073..18.515 rows=24055 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=13049
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..68.03 rows=15 width=0) (actual time=0.069..0.071 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..67.88 rows=15 width=4) (actual time=0.067..0.068 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..67.88 rows=15 width=4) (actual time=0.065..0.066 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=0) (actual time=0.044..0.044 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..56.19 rows=13 width=0) (actual time=0.030..0.032 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.06 rows=13 width=4) (actual time=0.029..0.030 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.06 rows=13 width=4) (actual time=0.028..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..32.42 rows=1 width=0) (actual time=0.046..0.050 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..32.41 rows=1 width=4) (actual time=0.045..0.048 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..32.41 rows=1 width=4) (actual time=0.044..0.047 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..32.41 rows=1 width=0) (actual time=0.040..0.043 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.039..0.041 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.035..0.036 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.914 ms
Execution Time: 70.441 ms
```

### vacuum A2 6%, value on 26k invisible records

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17343.54..17343.55 rows=1 width=8) (actual time=364.386..364.404 rows=1 loops=1)
  Buffers: shared hit=109283
  ->  Limit  (cost=0.55..17331.03 rows=1001 width=4) (actual time=364.380..364.397 rows=0 loops=1)
        Buffers: shared hit=109283
        ->  Result  (cost=0.55..17590.73 rows=1016 width=4) (actual time=364.377..364.394 rows=0 loops=1)
              Buffers: shared hit=109283
              ->  Append  (cost=0.55..17580.57 rows=1016 width=0) (actual time=364.375..364.391 rows=0 loops=1)
                    Buffers: shared hit=109283
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..17165.41 rows=1001 width=0) (actual time=364.256..364.259 rows=0 loops=1)
                          Buffers: shared hit=109269
                          ->  Limit  (cost=0.55..17155.40 rows=1001 width=4) (actual time=364.254..364.257 rows=0 loops=1)
                                Buffers: shared hit=109269
                                ->  Result  (cost=0.55..50282.61 rows=2934 width=4) (actual time=364.253..364.255 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=109269
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..50282.61 rows=2934 width=0) (actual time=364.240..364.242 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Filter: (SubPlan 1)
                                            Rows Removed by Filter: 24055
                                            Heap Fetches: 0
                                            Buffers: shared hit=109269
                                            SubPlan 1
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (actual time=0.014..0.014 rows=0 loops=24055)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r.tenant_id)
                                                    Buffers: shared hit=96220
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..8.45 rows=1 width=0) (actual time=0.012..0.012 rows=0 loops=24055)
                                                          Index Cond: ((tenant_id = r.tenant_id) AND (record_id = r.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '10'::double precision)
                                                          Rows Removed by Filter: 1
                                                          Buffers: shared hit=96220
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..203.20 rows=8 width=0) (actual time=0.043..0.047 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..203.12 rows=8 width=4) (actual time=0.042..0.045 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..203.12 rows=8 width=4) (actual time=0.039..0.041 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..203.12 rows=8 width=0) (actual time=0.033..0.035 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND (SubPlan 2))
                                            Buffers: shared hit=4
                                            SubPlan 2
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_1.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_1.tenant_id) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '10'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..166.00 rows=6 width=0) (actual time=0.029..0.032 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..165.94 rows=6 width=4) (actual time=0.028..0.030 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..165.94 rows=6 width=4) (actual time=0.027..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..165.94 rows=6 width=0) (actual time=0.023..0.025 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (SubPlan 3))
                                            Buffers: shared hit=4
                                            SubPlan 3
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_2.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_2.tenant_id) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '10'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..40.87 rows=1 width=0) (actual time=0.041..0.046 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..40.86 rows=1 width=4) (actual time=0.040..0.045 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..40.86 rows=1 width=4) (actual time=0.039..0.044 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..40.86 rows=1 width=0) (actual time=0.036..0.040 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.034..0.037 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.029..0.031 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..16.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (SubPlan 4))
                                                  SubPlan 4
                                                    ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_3.tenant_id)
                                                          ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = r_3.tenant_id) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                                Filter: ((value + '0'::double precision) = '10'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.280 ms
Execution Time: 364.691 ms
```

### vacuum before 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 1073741824) c
```

```
Aggregate  (cost=1315.99..1316.00 rows=1 width=8) (actual time=11.822..11.824 rows=1 loops=1)
  Buffers: shared hit=334
  ->  Limit  (cost=0.55..1146.19 rows=13584 width=28) (actual time=0.040..10.007 rows=19999 loops=1)
        Buffers: shared hit=334
        ->  Result  (cost=0.55..1146.19 rows=13584 width=28) (actual time=0.039..7.787 rows=19999 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=334
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1146.19 rows=13584 width=24) (actual time=0.031..4.396 rows=19999 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                    Heap Fetches: 0
                    Buffers: shared hit=334
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.162 ms
Execution Time: 11.863 ms
```

### vacuum before 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2313.61..2313.62 rows=1 width=8) (actual time=26.580..26.596 rows=1 loops=1)
  Buffers: shared hit=13397
  ->  Limit  (cost=1316.55..2312.34 rows=102 width=4) (actual time=26.576..26.591 rows=0 loops=1)
        Buffers: shared hit=13397
        ->  Result  (cost=1316.55..2312.34 rows=102 width=4) (actual time=26.575..26.590 rows=0 loops=1)
              Buffers: shared hit=13397
              ->  Append  (cost=1316.55..2311.32 rows=102 width=0) (actual time=26.574..26.588 rows=0 loops=1)
                    Buffers: shared hit=13397
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1316.55..2025.28 rows=99 width=0) (actual time=26.454..26.459 rows=0 loops=1)
                          Buffers: shared hit=13383
                          ->  Limit  (cost=1316.55..2024.29 rows=99 width=4) (actual time=26.452..26.456 rows=0 loops=1)
                                Buffers: shared hit=13383
                                ->  Result  (cost=1316.55..2024.29 rows=99 width=4) (actual time=26.451..26.455 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13383
                                      ->  Hash Semi Join  (cost=1316.55..2024.29 rows=99 width=0) (actual time=26.441..26.444 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=13383
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.040..9.644 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=1146.19..1146.19 rows=13584 width=32) (actual time=9.855..9.857 rows=19999 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1506kB
                                                  Buffers: shared hit=334
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1146.19 rows=13584 width=32) (actual time=0.020..5.284 rows=19999 loops=1)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=334
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..136.67 rows=1 width=0) (actual time=0.055..0.057 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..136.66 rows=1 width=4) (actual time=0.053..0.055 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..136.66 rows=1 width=4) (actual time=0.052..0.053 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..136.66 rows=1 width=0) (actual time=0.034..0.035 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.033..0.033 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..115.69 rows=1 width=0) (actual time=0.023..0.025 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..115.68 rows=1 width=4) (actual time=0.022..0.024 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..115.68 rows=1 width=4) (actual time=0.022..0.023 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..115.68 rows=1 width=0) (actual time=0.019..0.019 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.037..0.042 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.036..0.041 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.036..0.040 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.033..0.036 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.032..0.035 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.032..0.034 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.028..0.029 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.024..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.049 ms
Execution Time: 26.824 ms
```

### vacuum A2 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=1315.99..1316.00 rows=1 width=8) (actual time=9.466..9.469 rows=1 loops=1)
  Buffers: shared hit=334
  ->  Limit  (cost=0.55..1146.19 rows=13584 width=28) (actual time=0.043..8.034 rows=19999 loops=1)
        Buffers: shared hit=334
        ->  Result  (cost=0.55..1146.19 rows=13584 width=28) (actual time=0.042..6.186 rows=19999 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=334
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1146.19 rows=13584 width=24) (actual time=0.033..3.271 rows=19999 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                    Heap Fetches: 0
                    Buffers: shared hit=334
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.239 ms
Execution Time: 9.511 ms
```

### vacuum A2 6%, 19,999 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2313.61..2313.62 rows=1 width=8) (actual time=17.772..17.784 rows=1 loops=1)
  Buffers: shared hit=13397
  ->  Limit  (cost=1316.55..2312.34 rows=102 width=4) (actual time=17.769..17.779 rows=0 loops=1)
        Buffers: shared hit=13397
        ->  Result  (cost=1316.55..2312.34 rows=102 width=4) (actual time=17.767..17.778 rows=0 loops=1)
              Buffers: shared hit=13397
              ->  Append  (cost=1316.55..2311.32 rows=102 width=0) (actual time=17.766..17.777 rows=0 loops=1)
                    Buffers: shared hit=13397
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1316.55..2025.28 rows=99 width=0) (actual time=17.675..17.679 rows=0 loops=1)
                          Buffers: shared hit=13383
                          ->  Limit  (cost=1316.55..2024.29 rows=99 width=4) (actual time=17.674..17.677 rows=0 loops=1)
                                Buffers: shared hit=13383
                                ->  Result  (cost=1316.55..2024.29 rows=99 width=4) (actual time=17.672..17.675 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13383
                                      ->  Hash Semi Join  (cost=1316.55..2024.29 rows=99 width=0) (actual time=17.662..17.665 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=13383
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.036..5.942 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=1146.19..1146.19 rows=13584 width=32) (actual time=7.819..7.820 rows=19999 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1506kB
                                                  Buffers: shared hit=334
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1146.19 rows=13584 width=32) (actual time=0.019..4.108 rows=19999 loops=1)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=334
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..136.67 rows=1 width=0) (actual time=0.041..0.042 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..136.66 rows=1 width=4) (actual time=0.040..0.041 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..136.66 rows=1 width=4) (actual time=0.038..0.039 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..136.66 rows=1 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.023..0.024 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..115.69 rows=1 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..115.68 rows=1 width=4) (actual time=0.018..0.019 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..115.68 rows=1 width=4) (actual time=0.018..0.018 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..115.68 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.014..0.014 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '30'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.028..0.031 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.027..0.029 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.027..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.024..0.027 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.024..0.026 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.023..0.024 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.020..0.020 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '30'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.758 ms
Execution Time: 17.965 ms
```

### vacuum before 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 1073741824) c
```

```
Aggregate  (cost=1241.81..1241.82 rows=1 width=8) (actual time=16.503..16.505 rows=1 loops=1)
  Buffers: shared hit=337
  ->  Limit  (cost=0.55..1081.36 rows=12836 width=28) (actual time=0.061..14.026 rows=20000 loops=1)
        Buffers: shared hit=337
        ->  Result  (cost=0.55..1081.36 rows=12836 width=28) (actual time=0.059..10.840 rows=20000 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=337
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1081.36 rows=12836 width=24) (actual time=0.048..5.744 rows=20000 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                    Heap Fetches: 0
                    Buffers: shared hit=337
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.260 ms
Execution Time: 16.557 ms
```

### vacuum before 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2239.19..2239.20 rows=1 width=8) (actual time=23.757..23.767 rows=1 loops=1)
  Buffers: shared hit=13400
  ->  Limit  (cost=1242.37..2237.98 rows=97 width=4) (actual time=23.754..23.763 rows=0 loops=1)
        Buffers: shared hit=13400
        ->  Result  (cost=1242.37..2237.98 rows=97 width=4) (actual time=23.753..23.762 rows=0 loops=1)
              Buffers: shared hit=13400
              ->  Append  (cost=1242.37..2237.01 rows=97 width=0) (actual time=23.752..23.761 rows=0 loops=1)
                    Buffers: shared hit=13400
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1242.37..1951.00 rows=94 width=0) (actual time=23.666..23.668 rows=0 loops=1)
                          Buffers: shared hit=13386
                          ->  Limit  (cost=1242.37..1950.06 rows=94 width=4) (actual time=23.665..23.667 rows=0 loops=1)
                                Buffers: shared hit=13386
                                ->  Result  (cost=1242.37..1950.06 rows=94 width=4) (actual time=23.664..23.666 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13386
                                      ->  Hash Semi Join  (cost=1242.37..1950.06 rows=94 width=0) (actual time=23.657..23.659 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=13386
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=32) (actual time=0.030..6.998 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=1081.36..1081.36 rows=12836 width=32) (actual time=12.217..12.218 rows=20000 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1506kB
                                                  Buffers: shared hit=337
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1081.36 rows=12836 width=32) (actual time=0.016..6.238 rows=20000 loops=1)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=337
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..136.67 rows=1 width=0) (actual time=0.039..0.040 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..136.66 rows=1 width=4) (actual time=0.038..0.039 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..136.66 rows=1 width=4) (actual time=0.035..0.036 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..136.66 rows=1 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=32) (actual time=0.022..0.022 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..115.69 rows=1 width=0) (actual time=0.019..0.021 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..115.68 rows=1 width=4) (actual time=0.018..0.019 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..115.68 rows=1 width=4) (actual time=0.016..0.017 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..115.68 rows=1 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=32) (actual time=0.013..0.013 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..4.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.16 rows=1 width=0) (actual time=0.024..0.027 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.15 rows=1 width=4) (actual time=0.023..0.026 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.15 rows=1 width=4) (actual time=0.023..0.025 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.15 rows=1 width=0) (actual time=0.021..0.023 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.021..0.023 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.021..0.021 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.74 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                  Filter: (value = '31'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.405 ms
Execution Time: 23.877 ms
```

### vacuum A2 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=1241.81..1241.82 rows=1 width=8) (actual time=15.042..15.045 rows=1 loops=1)
  Buffers: shared hit=337
  ->  Limit  (cost=0.55..1081.36 rows=12836 width=28) (actual time=0.051..12.484 rows=20000 loops=1)
        Buffers: shared hit=337
        ->  Result  (cost=0.55..1081.36 rows=12836 width=28) (actual time=0.050..9.695 rows=20000 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=337
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1081.36 rows=12836 width=24) (actual time=0.039..5.490 rows=20000 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '31'::double precision))
                    Heap Fetches: 0
                    Buffers: shared hit=337
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.271 ms
Execution Time: 15.096 ms
```

### vacuum A2 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 30001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 30001)) u LIMIT 30001) c
```

```
Aggregate  (cost=1069.24..1069.25 rows=1 width=8) (actual time=36.809..36.818 rows=1 loops=1)
  Buffers: shared hit=13063
  ->  Limit  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.058..33.443 rows=24055 loops=1)
        Buffers: shared hit=13063
        ->  Result  (cost=0.55..995.54 rows=5896 width=4) (actual time=0.057..29.762 rows=24055 loops=1)
              Buffers: shared hit=13063
              ->  Append  (cost=0.55..936.58 rows=5896 width=0) (actual time=0.056..24.511 rows=24055 loops=1)
                    Buffers: shared hit=13063
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..750.46 rows=5867 width=0) (actual time=0.055..20.986 rows=24055 loops=1)
                          Buffers: shared hit=13049
                          ->  Limit  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.054..17.103 rows=24055 loops=1)
                                Buffers: shared hit=13049
                                ->  Result  (cost=0.55..691.79 rows=5867 width=4) (actual time=0.054..13.923 rows=24055 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=13049
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..691.79 rows=5867 width=0) (actual time=0.044..9.388 rows=24055 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=13049
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..68.03 rows=15 width=0) (actual time=0.052..0.053 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..67.88 rows=15 width=4) (actual time=0.051..0.052 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..67.88 rows=15 width=4) (actual time=0.048..0.049 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..67.88 rows=15 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..56.19 rows=13 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..56.06 rows=13 width=4) (actual time=0.020..0.021 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..56.06 rows=13 width=4) (actual time=0.019..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.06 rows=13 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)))
                                            Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..32.42 rows=1 width=0) (actual time=0.034..0.038 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..32.41 rows=1 width=4) (actual time=0.034..0.037 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..32.41 rows=1 width=4) (actual time=0.033..0.036 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..32.41 rows=1 width=0) (actual time=0.031..0.033 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.030..0.031 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.027..0.029 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.062 ms
Execution Time: 36.942 ms
```

### vacuum A2 6%, 20,000 invisible side rows

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND (v.value + 0) = $9::double precision OFFSET 0) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17343.54..17343.55 rows=1 width=8) (actual time=199.961..199.973 rows=1 loops=1)
  Buffers: shared hit=109283
  ->  Limit  (cost=0.55..17331.03 rows=1001 width=4) (actual time=199.957..199.969 rows=0 loops=1)
        Buffers: shared hit=109283
        ->  Result  (cost=0.55..17590.73 rows=1016 width=4) (actual time=199.956..199.967 rows=0 loops=1)
              Buffers: shared hit=109283
              ->  Append  (cost=0.55..17580.57 rows=1016 width=0) (actual time=199.955..199.966 rows=0 loops=1)
                    Buffers: shared hit=109283
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..17165.41 rows=1001 width=0) (actual time=199.872..199.874 rows=0 loops=1)
                          Buffers: shared hit=109269
                          ->  Limit  (cost=0.55..17155.40 rows=1001 width=4) (actual time=199.871..199.872 rows=0 loops=1)
                                Buffers: shared hit=109269
                                ->  Result  (cost=0.55..50282.61 rows=2934 width=4) (actual time=199.870..199.871 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=109269
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..50282.61 rows=2934 width=0) (actual time=199.861..199.862 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                            Filter: (SubPlan 1)
                                            Rows Removed by Filter: 24055
                                            Heap Fetches: 0
                                            Buffers: shared hit=109269
                                            SubPlan 1
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (actual time=0.007..0.007 rows=0 loops=24055)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r.tenant_id)
                                                    Buffers: shared hit=96220
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..8.45 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=24055)
                                                          Index Cond: ((tenant_id = r.tenant_id) AND (record_id = r.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '31'::double precision)
                                                          Rows Removed by Filter: 1
                                                          Buffers: shared hit=96220
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..203.20 rows=8 width=0) (actual time=0.028..0.030 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..203.12 rows=8 width=4) (actual time=0.027..0.029 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..203.12 rows=8 width=4) (actual time=0.025..0.027 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..203.12 rows=8 width=0) (actual time=0.022..0.024 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND (SubPlan 2))
                                            Buffers: shared hit=4
                                            SubPlan 2
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_1.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_1.tenant_id) AND (record_id = r_1.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '31'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.55..166.00 rows=6 width=0) (actual time=0.019..0.021 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=0.55..165.94 rows=6 width=4) (actual time=0.018..0.020 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.55..165.94 rows=6 width=4) (actual time=0.018..0.019 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..165.94 rows=6 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                            Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                            Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (SubPlan 3))
                                            Buffers: shared hit=4
                                            SubPlan 3
                                              ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_2.tenant_id)
                                                    ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                          Index Cond: ((tenant_id = r_2.tenant_id) AND (record_id = r_2.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                          Filter: ((value + '0'::double precision) = '31'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.36..40.87 rows=1 width=0) (actual time=0.032..0.035 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.36..40.86 rows=1 width=4) (actual time=0.031..0.034 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.36..40.86 rows=1 width=4) (actual time=0.030..0.034 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop  (cost=24.36..40.86 rows=1 width=0) (actual time=0.028..0.031 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.027..0.029 rows=0 loops=1)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=6
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.023..0.024 rows=0 loops=1)
                                                        Recheck Cond: (((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                        ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid))
                                                                    Buffers: shared hit=2
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[])))
                                                                    Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..16.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,32858692-1b07-435c-b2cc-a99d222fdece}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ea1571fa-ab50-4b46-8455-0de0f90aeba6'::uuid)) AND (entity_definition_id = '01a1fc4d-13a1-7b42-b4b6-13b8acf1e13f'::uuid) AND (SubPlan 4))
                                                  SubPlan 4
                                                    ->  Result  (cost=0.42..8.45 rows=1 width=4) (never executed)
                                                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r_3.tenant_id)
                                                          ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..8.45 rows=1 width=0) (never executed)
                                                                Index Cond: ((tenant_id = r_3.tenant_id) AND (record_id = r_3.id) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid))
                                                                Filter: ((value + '0'::double precision) = '31'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.228 ms
Execution Time: 200.138 ms
```

### vacuum S-A2-skew pre-count

```sql
SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = $3::double precision ORDER BY v.value, v.record_id LIMIT 20000) c
```

```
Aggregate  (cost=1934.52..1934.53 rows=1 width=8) (actual time=0.161..0.163 rows=1 loops=1)
  Buffers: shared hit=5
  ->  Limit  (cost=0.55..1684.52 rows=20000 width=28) (actual time=0.097..0.144 rows=50 loops=1)
        Buffers: shared hit=5
        ->  Result  (cost=0.55..11207.28 rows=133099 width=28) (actual time=0.095..0.129 rows=50 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid)
              Buffers: shared hit=5
              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..11207.28 rows=133099 width=24) (actual time=0.083..0.100 rows=50 loops=1)
                    Index Cond: ((tenant_id = '43974321-4945-419c-8bc7-3abcb59b4a93'::uuid) AND (field_id = '778a1d04-5dc1-4539-9e38-093cd525f29c'::uuid) AND (value = '77'::double precision))
                    Heap Fetches: 0
                    Buffers: shared hit=5
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.363 ms
Execution Time: 0.227 ms
```
