## agent sort=-state page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..676.38 rows=51 width=1252) (actual time=0.467..10.920 rows=51 loops=1)
  Buffers: shared hit=983
  ->  Result  (cost=1.00..1360354.26 rows=102724 width=1252) (actual time=0.465..10.902 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=983
        ->  Nested Loop Left Join  (cost=1.00..1360097.45 rows=102724 width=1250) (actual time=0.450..10.854 rows=51 loops=1)
              Buffers: shared hit=983
              ->  Nested Loop  (cost=0.70..1325626.80 rows=102724 width=1242) (actual time=0.403..10.328 rows=51 loops=1)
                    Buffers: shared hit=830
                    ->  Index Scan using records_list_state on records r  (cost=0.42..1324334.45 rows=102724 width=1235) (actual time=0.361..10.196 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 749
                          Buffers: shared hit=827
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.127..0.153 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.102..0.103 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.033..0.033 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.067..0.067 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.033..0.034 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=588
Planning Time: 3.667 ms
Execution Time: 11.111 ms
```
## agent sort=-state page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND (r.state, r.updated_at, r.id) < ('on_hold'::text, '2026-04-28 02:19:57.679848+00'::timestamptz, '0a83c977-e471-84b5-771f-d66ac05f6052'::uuid) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..681.70 rows=51 width=1252) (actual time=0.671..9.295 rows=51 loops=1)
  Buffers: shared hit=939
  ->  Result  (cost=1.00..1300500.43 rows=97437 width=1252) (actual time=0.670..9.280 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=939
        ->  Nested Loop Left Join  (cost=1.00..1300256.84 rows=97437 width=1250) (actual time=0.662..9.242 rows=51 loops=1)
              Buffers: shared hit=939
              ->  Nested Loop  (cost=0.70..1267464.82 rows=97437 width=1242) (actual time=0.623..8.867 rows=51 loops=1)
                    Buffers: shared hit=786
                    ->  Index Scan using records_list_state on records r  (cost=0.42..1266238.56 rows=97437 width=1235) (actual time=0.585..8.769 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(state, updated_at, id) < ROW('on_hold'::text, '2026-04-28 02:19:57.679848+00'::timestamp with time zone, '0a83c977-e471-84b5-771f-d66ac05f6052'::uuid)))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 708
                          Buffers: shared hit=783
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.111..0.136 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.090..0.092 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.031..0.031 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.058..0.059 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.030..0.031 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=577
Planning Time: 2.272 ms
Execution Time: 9.485 ms
```
## req1 sort=-state page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..690.98 rows=51 width=1252) (actual time=0.968..32.669 rows=51 loops=1)
  Buffers: shared hit=3805
  ->  Result  (cost=1.00..1357825.85 rows=100364 width=1252) (actual time=0.964..32.639 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=3805
        ->  Nested Loop Left Join  (cost=1.00..1357574.94 rows=100364 width=1250) (actual time=0.942..32.571 rows=51 loops=1)
              Buffers: shared hit=3805
              ->  Nested Loop  (cost=0.70..1323853.59 rows=100364 width=1242) (actual time=0.612..31.457 rows=51 loops=1)
                    Buffers: shared hit=3652
                    ->  Index Scan using records_list_state on records r  (cost=0.42..1322590.75 rows=100364 width=1235) (actual time=0.389..31.109 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 3556
                          Buffers: shared hit=3649
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.049..0.052 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.046..0.048 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.044..0.044 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.005..0.005 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.207..0.212 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.020..0.020 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.015..0.015 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=588
Planning Time: 4.536 ms
Execution Time: 34.207 ms
```
## req1 sort=-state page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.state, r.updated_at, r.id) < ('closed'::text, '2026-09-30 23:50:42.243604+00'::timestamptz, '637095e4-261f-0e89-7aab-6a41d300eb92'::uuid) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..29.34 rows=1 width=1252) (actual time=3.231..49.365 rows=51 loops=1)
  Buffers: shared hit=10880
  ->  Result  (cost=1.00..29.34 rows=1 width=1252) (actual time=3.228..49.324 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=10880
        ->  Nested Loop Left Join  (cost=1.00..29.34 rows=1 width=1250) (actual time=3.111..49.147 rows=51 loops=1)
              Buffers: shared hit=10880
              ->  Nested Loop  (cost=0.70..21.01 rows=1 width=1242) (actual time=3.072..48.394 rows=51 loops=1)
                    Buffers: shared hit=10727
                    ->  Index Scan using records_list_state on records r  (cost=0.42..12.70 rows=1 width=1235) (actual time=3.010..48.005 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(state, updated_at, id) < ROW('closed'::text, '2026-09-30 23:50:42.243604+00'::timestamp with time zone, '637095e4-261f-0e89-7aab-6a41d300eb92'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (SubPlan 1))
                          Rows Removed by Filter: 3494
                          Buffers: shared hit=10574
                          SubPlan 1
                            ->  Result  (cost=0.29..4.31 rows=1 width=0) (actual time=0.004..0.004 rows=0 loops=3494)
                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r.tenant_id)
                                  Buffers: shared hit=6989
                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.31 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=3494)
                                        Index Cond: ((tenant_id = r.tenant_id) AND (record_id = r.id))
                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                        Rows Removed by Filter: 0
                                        Heap Fetches: 0
                                        Buffers: shared hit=6989
                    ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.005..0.006 rows=1 loops=51)
                          Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                          Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                          Buffers: shared hit=153
              ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.013..0.013 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=577
Planning Time: 2.800 ms
Execution Time: 49.630 ms
```
## req8 sort=-state page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..694.62 rows=51 width=1252) (actual time=5.058..137.634 rows=51 loops=1)
  Buffers: shared hit=16040
  ->  Result  (cost=1.00..1357645.97 rows=99823 width=1252) (actual time=5.050..137.564 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=16040
        ->  Nested Loop Left Join  (cost=1.00..1357396.41 rows=99823 width=1250) (actual time=5.004..137.409 rows=51 loops=1)
              Buffers: shared hit=16040
              ->  Nested Loop  (cost=0.70..1323846.83 rows=99823 width=1242) (actual time=4.929..135.486 rows=51 loops=1)
                    Buffers: shared hit=15887
                    ->  Index Scan using records_list_state on records r  (cost=0.42..1322590.75 rows=99823 width=1235) (actual time=4.842..135.100 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 15662
                          Buffers: shared hit=15884
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.176..0.206 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.132..0.134 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.129..0.130 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.061..0.063 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.032..0.032 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.019..0.019 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=588
Planning Time: 5.744 ms
Execution Time: 138.173 ms
```
## req8 sort=-state page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.state, r.updated_at, r.id) < ('closed'::text, '2026-03-03 03:54:02.361285+00'::timestamptz, '919e8099-b9b4-f7cc-9c73-08e067759ae9'::uuid) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..29.34 rows=1 width=1252) (actual time=6.995..127.467 rows=51 loops=1)
  Buffers: shared hit=55283
  ->  Result  (cost=1.00..29.34 rows=1 width=1252) (actual time=6.993..127.426 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=55283
        ->  Nested Loop Left Join  (cost=1.00..29.34 rows=1 width=1250) (actual time=6.986..127.356 rows=51 loops=1)
              Buffers: shared hit=55283
              ->  Nested Loop  (cost=0.70..21.01 rows=1 width=1242) (actual time=6.940..126.761 rows=51 loops=1)
                    Buffers: shared hit=55130
                    ->  Index Scan using records_list_state on records r  (cost=0.42..12.70 rows=1 width=1235) (actual time=6.898..126.391 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(state, updated_at, id) < ROW('closed'::text, '2026-03-03 03:54:02.361285+00'::timestamp with time zone, '919e8099-b9b4-f7cc-9c73-08e067759ae9'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (SubPlan 1))
                          Rows Removed by Filter: 18244
                          Buffers: shared hit=54977
                          SubPlan 1
                            ->  Result  (cost=0.29..4.31 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=18244)
                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = r.tenant_id)
                                  Buffers: shared hit=36489
                                  ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..4.31 rows=1 width=0) (actual time=0.002..0.002 rows=0 loops=18244)
                                        Index Cond: ((tenant_id = r.tenant_id) AND (record_id = r.id))
                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[]))))
                                        Rows Removed by Filter: 0
                                        Heap Fetches: 0
                                        Buffers: shared hit=36489
                    ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.005..0.005 rows=1 loops=51)
                          Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                          Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                          Buffers: shared hit=153
              ->  Result  (cost=0.29..8.31 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..8.31 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=577
Planning Time: 1.923 ms
Execution Time: 127.689 ms
```
## admin sort=-state page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..136.72 rows=51 width=1252) (actual time=0.152..1.130 rows=51 loops=1)
  Buffers: shared hit=210
  ->  Result  (cost=1.00..530319.89 rows=199281 width=1252) (actual time=0.152..1.123 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=210
        ->  Nested Loop Left Join  (cost=1.00..529821.69 rows=199281 width=1250) (actual time=0.139..1.098 rows=51 loops=1)
              Buffers: shared hit=210
              ->  Nested Loop  (cost=0.70..464694.34 rows=199281 width=1242) (actual time=0.104..0.814 rows=51 loops=1)
                    Buffers: shared hit=57
                    ->  Index Scan using records_list_state on records r  (cost=0.42..462195.03 rows=199281 width=1235) (actual time=0.074..0.758 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Buffers: shared hit=54
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.025..0.026 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=526
Planning Time: 1.778 ms
Execution Time: 1.239 ms
```
## admin sort=-state page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND (r.state, r.updated_at, r.id) < ('on_hold'::text, '2026-09-30 22:51:53.391587+00'::timestamptz, '7ba39b95-a0b8-d442-cf23-4a5fdf7a2b3c'::uuid) ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..139.46 rows=51 width=1252) (actual time=0.168..1.555 rows=51 loops=1)
  Buffers: shared hit=210
  ->  Result  (cost=1.00..513187.92 rows=189024 width=1252) (actual time=0.167..1.544 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=210
        ->  Nested Loop Left Join  (cost=1.00..512715.36 rows=189024 width=1250) (actual time=0.156..1.512 rows=51 loops=1)
              Buffers: shared hit=210
              ->  Nested Loop  (cost=0.70..450844.57 rows=189024 width=1242) (actual time=0.120..1.080 rows=51 loops=1)
                    Buffers: shared hit=57
                    ->  Index Scan using records_list_state on records r  (cost=0.42..448473.48 rows=189024 width=1235) (actual time=0.085..1.007 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(state, updated_at, id) < ROW('on_hold'::text, '2026-09-30 22:51:53.391587+00'::timestamp with time zone, '7ba39b95-a0b8-d442-cf23-4a5fdf7a2b3c'::uuid)))
                          Buffers: shared hit=54
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.029..0.029 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=515
Planning Time: 1.798 ms
Execution Time: 1.697 ms
```
## agent sort=-assignee_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..677.13 rows=51 width=1277) (actual time=0.523..13.395 rows=51 loops=1)
  Buffers: shared hit=1105
  ->  Result  (cost=1.12..1361620.01 rows=102724 width=1277) (actual time=0.521..13.351 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=1105
        ->  Nested Loop Left Join  (cost=1.12..1360849.58 rows=102724 width=1250) (actual time=0.510..13.302 rows=51 loops=1)
              Buffers: shared hit=1105
              ->  Nested Loop  (cost=0.82..1326378.92 rows=102724 width=1242) (actual time=0.461..12.632 rows=51 loops=1)
                    Buffers: shared hit=952
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..1325086.58 rows=102724 width=1235) (actual time=0.422..12.501 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 869
                          Buffers: shared hit=949
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.128..0.156 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.102..0.104 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.035..0.036 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.065..0.066 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.032..0.033 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 2.822 ms
Execution Time: 13.612 ms
```
## agent sort=-assignee_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND r.assignee_id IS NULL AND (r.updated_at, r.id) < ('2026-09-03 03:46:35.170987+00'::timestamptz, 'bc473acd-2770-8e37-a327-ace4219d7b62'::uuid) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..843.96 rows=51 width=1277) (actual time=0.718..9.231 rows=51 loops=1)
  Buffers: shared hit=974
  ->  Result  (cost=1.12..95985.32 rows=5808 width=1277) (actual time=0.716..9.217 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=974
        ->  Nested Loop Left Join  (cost=1.12..95941.76 rows=5808 width=1250) (actual time=0.707..9.175 rows=51 loops=1)
              Buffers: shared hit=974
              ->  Nested Loop  (cost=0.82..92241.71 rows=5808 width=1242) (actual time=0.643..8.802 rows=51 loops=1)
                    Buffers: shared hit=821
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..92160.82 rows=5808 width=1235) (actual time=0.589..8.688 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NULL) AND (ROW(updated_at, id) < ROW('2026-09-03 03:46:35.170987+00'::timestamp with time zone, 'bc473acd-2770-8e37-a327-ace4219d7b62'::uuid)))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 742
                          Buffers: shared hit=818
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.116..0.139 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.092..0.094 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.028..0.028 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.063..0.063 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.042..0.043 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.63 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.63 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 3.317 ms
Execution Time: 9.411 ms
```
## req1 sort=-assignee_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..691.74 rows=51 width=1277) (actual time=3.632..24.446 rows=51 loops=1)
  Buffers: shared hit=3495
  ->  Result  (cost=1.12..1359079.80 rows=100364 width=1277) (actual time=3.630..24.425 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=3495
        ->  Nested Loop Left Join  (cost=1.12..1358327.07 rows=100364 width=1250) (actual time=3.616..24.373 rows=51 loops=1)
              Buffers: shared hit=3495
              ->  Nested Loop  (cost=0.82..1324605.72 rows=100364 width=1242) (actual time=3.568..23.700 rows=51 loops=1)
                    Buffers: shared hit=3342
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..1323342.87 rows=100364 width=1235) (actual time=3.504..23.558 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 3248
                          Buffers: shared hit=3339
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.033..0.035 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.032..0.034 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.031..0.031 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.043..0.044 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 2.559 ms
Execution Time: 24.688 ms
```
## req1 sort=-assignee_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.assignee_id, r.updated_at, r.id) < ('ed771678-7ebb-578a-22b7-486e73f2f3e8'::uuid, '2026-09-04 21:44:35.107969+00'::timestamptz, '7c3adb41-4f44-a2fd-8afa-644528543379'::uuid) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..709.57 rows=51 width=1277) (actual time=1.428..35.648 rows=51 loops=1)
  Buffers: shared hit=3541
  ->  Result  (cost=1.12..1163516.58 rows=83760 width=1277) (actual time=1.425..35.603 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=3541
        ->  Nested Loop Left Join  (cost=1.12..1162888.38 rows=83760 width=1250) (actual time=1.404..35.472 rows=51 loops=1)
              Buffers: shared hit=3541
              ->  Nested Loop  (cost=0.82..1134438.76 rows=83760 width=1242) (actual time=1.357..34.221 rows=51 loops=1)
                    Buffers: shared hit=3388
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..1133383.47 rows=83760 width=1235) (actual time=1.280..33.972 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(assignee_id, updated_at, id) < ROW('ed771678-7ebb-578a-22b7-486e73f2f3e8'::uuid, '2026-09-04 21:44:35.107969+00'::timestamp with time zone, '7c3adb41-4f44-a2fd-8afa-644528543379'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 3290
                          Buffers: shared hit=3385
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.061..0.064 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.059..0.061 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.057..0.058 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.059..0.061 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.021..0.021 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 5.385 ms
Execution Time: 36.168 ms
```
## req8 sort=-assignee_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..695.39 rows=51 width=1277) (actual time=4.378..106.044 rows=51 loops=1)
  Buffers: shared hit=18369
  ->  Result  (cost=1.12..1358897.21 rows=99823 width=1277) (actual time=4.376..105.945 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=18369
        ->  Nested Loop Left Join  (cost=1.12..1358148.53 rows=99823 width=1250) (actual time=4.359..105.833 rows=51 loops=1)
              Buffers: shared hit=18369
              ->  Nested Loop  (cost=0.82..1324598.95 rows=99823 width=1242) (actual time=4.294..104.603 rows=51 loops=1)
                    Buffers: shared hit=18216
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..1323342.87 rows=99823 width=1235) (actual time=4.219..104.309 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 17967
                          Buffers: shared hit=18213
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.195..0.234 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.045..0.047 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.043..0.044 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.051..0.053 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.021..0.021 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 4.355 ms
Execution Time: 106.319 ms
```
## req8 sort=-assignee_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.assignee_id, r.updated_at, r.id) < ('304bd41c-1872-66c8-e59d-37d31e44966d'::uuid, '2026-02-15 00:27:00.844898+00'::timestamptz, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..827.42 rows=51 width=1277) (actual time=1.025..107.586 rows=51 loops=1)
  Buffers: shared hit=26341
  ->  Result  (cost=1.12..270264.18 rows=16681 width=1277) (actual time=1.023..107.537 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=26341
        ->  Nested Loop Left Join  (cost=1.12..270139.07 rows=16681 width=1250) (actual time=1.009..107.374 rows=51 loops=1)
              Buffers: shared hit=26341
              ->  Nested Loop  (cost=0.82..262986.98 rows=16681 width=1242) (actual time=0.942..105.691 rows=51 loops=1)
                    Buffers: shared hit=26188
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..262770.17 rows=16681 width=1235) (actual time=0.901..105.374 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(assignee_id, updated_at, id) < ROW('304bd41c-1872-66c8-e59d-37d31e44966d'::uuid, '2026-02-15 00:27:00.844898+00'::timestamp with time zone, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 25829
                          Buffers: shared hit=26185
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.048..0.064 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.029..0.032 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.028..0.029 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.031..0.032 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.42 rows=1 width=40) (actual time=0.029..0.029 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.42 rows=1 width=40) (actual time=0.018..0.018 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 2.220 ms
Execution Time: 107.813 ms
```
## admin sort=-assignee_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..137.29 rows=51 width=1277) (actual time=0.207..1.254 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.12..532068.42 rows=199281 width=1277) (actual time=0.206..1.246 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.12..530573.81 rows=199281 width=1250) (actual time=0.195..1.218 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.82..465446.46 rows=199281 width=1242) (actual time=0.155..0.884 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..462947.15 rows=199281 width=1235) (actual time=0.119..0.817 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.030..0.031 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=530
Planning Time: 1.693 ms
Execution Time: 1.346 ms
```
## admin sort=-assignee_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND r.assignee_id IS NULL AND (r.updated_at, r.id) < ('2026-09-30 23:51:41.929388+00'::timestamptz, '0a7a86fe-cab1-3204-c526-5394aede79a8'::uuid) ORDER BY r.assignee_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..214.61 rows=51 width=1277) (actual time=0.324..1.488 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.12..79658.18 rows=19029 width=1277) (actual time=0.323..1.479 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.12..79515.46 rows=19029 width=1250) (actual time=0.308..1.446 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.82..71617.86 rows=19029 width=1242) (actual time=0.253..1.075 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_assignee on records r  (cost=0.55..71371.70 rows=19029 width=1235) (actual time=0.200..0.989 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NULL) AND (ROW(updated_at, id) < ROW('2026-09-30 23:51:41.929388+00'::timestamp with time zone, '0a7a86fe-cab1-3204-c526-5394aede79a8'::uuid)))
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.044..0.045 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.41 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.41 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=519
Planning Time: 1.753 ms
Execution Time: 1.651 ms
```
## agent sort=-requester_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..677.18 rows=51 width=1277) (actual time=0.894..10.776 rows=51 loops=1)
  Buffers: shared hit=890
  ->  Result  (cost=1.12..1361708.01 rows=102724 width=1277) (actual time=0.892..10.755 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=890
        ->  Nested Loop Left Join  (cost=1.12..1360937.58 rows=102724 width=1250) (actual time=0.872..10.670 rows=51 loops=1)
              Buffers: shared hit=890
              ->  Nested Loop  (cost=0.82..1326466.92 rows=102724 width=1242) (actual time=0.797..10.117 rows=51 loops=1)
                    Buffers: shared hit=737
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..1325174.58 rows=102724 width=1235) (actual time=0.730..9.960 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 655
                          Buffers: shared hit=734
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.165..0.200 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.132..0.133 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.048..0.048 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.083..0.083 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.052..0.053 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 3.600 ms
Execution Time: 11.018 ms
```
## agent sort=-requester_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND (r.requester_id, r.updated_at, r.id) < ('f8ee9ff7-37f9-86e7-241c-f98ec7ef224c'::uuid, '2026-09-30 18:18:44.976143+00'::timestamptz, '6d55cafc-2aef-4545-df1b-dadf533dc8d8'::uuid) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..680.39 rows=51 width=1277) (actual time=0.587..9.476 rows=51 loops=1)
  Buffers: shared hit=877
  ->  Result  (cost=1.12..1326577.44 rows=99600 width=1277) (actual time=0.585..9.458 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=877
        ->  Nested Loop Left Join  (cost=1.12..1325830.44 rows=99600 width=1250) (actual time=0.567..9.370 rows=51 loops=1)
              Buffers: shared hit=877
              ->  Nested Loop  (cost=0.82..1292351.66 rows=99600 width=1242) (actual time=0.501..8.931 rows=51 loops=1)
                    Buffers: shared hit=724
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..1291098.37 rows=99600 width=1235) (actual time=0.423..8.783 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(requester_id, updated_at, id) < ROW('f8ee9ff7-37f9-86e7-241c-f98ec7ef224c'::uuid, '2026-09-30 18:18:44.976143+00'::timestamp with time zone, '6d55cafc-2aef-4545-df1b-dadf533dc8d8'::uuid)))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 645
                          Buffers: shared hit=721
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.148..0.182 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.124..0.127 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.050..0.050 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.073..0.073 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.067..0.068 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 4.296 ms
Execution Time: 9.824 ms
```
## req1 sort=-requester_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..691.78 rows=51 width=1277) (actual time=24.242..43.619 rows=51 loops=1)
  Buffers: shared hit=6017
  ->  Result  (cost=1.12..1359167.80 rows=100364 width=1277) (actual time=24.240..43.606 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=6017
        ->  Nested Loop Left Join  (cost=1.12..1358415.07 rows=100364 width=1250) (actual time=24.227..43.546 rows=51 loops=1)
              Buffers: shared hit=6017
              ->  Nested Loop  (cost=0.82..1324693.72 rows=100364 width=1242) (actual time=24.171..43.121 rows=51 loops=1)
                    Buffers: shared hit=5864
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..1323430.87 rows=100364 width=1235) (actual time=24.100..43.006 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 5737
                          Buffers: shared hit=5861
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.032..0.035 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.031..0.033 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.030..0.030 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.047..0.048 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 2.274 ms
Execution Time: 43.990 ms
```
## req1 sort=-requester_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.requester_id, r.updated_at, r.id) < ('fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid, '2026-09-29 01:24:54.740311+00'::timestamptz, '711ba42e-837e-d1a7-c37b-702f9afe97a5'::uuid) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..694.14 rows=51 width=1277) (actual time=0.387..1.507 rows=51 loops=1)
  Buffers: shared hit=212
  ->  Result  (cost=1.12..1334958.12 rows=98241 width=1277) (actual time=0.385..1.498 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=212
        ->  Nested Loop Left Join  (cost=1.12..1334221.31 rows=98241 width=1250) (actual time=0.373..1.448 rows=51 loops=1)
              Buffers: shared hit=212
              ->  Nested Loop  (cost=0.82..1301174.01 rows=98241 width=1242) (actual time=0.315..1.055 rows=51 loops=1)
                    Buffers: shared hit=59
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..1299937.71 rows=98241 width=1235) (actual time=0.188..0.898 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(requester_id, updated_at, id) < ROW('fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid, '2026-09-29 01:24:54.740311+00'::timestamp with time zone, '711ba42e-837e-d1a7-c37b-702f9afe97a5'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Buffers: shared hit=56
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (never executed)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (never executed)
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.117..0.118 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 2.192 ms
Execution Time: 1.678 ms
```
## req8 sort=-requester_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..695.43 rows=51 width=1277) (actual time=0.887..159.877 rows=51 loops=1)
  Buffers: shared hit=28799
  ->  Result  (cost=1.12..1358985.21 rows=99823 width=1277) (actual time=0.885..159.844 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=28799
        ->  Nested Loop Left Join  (cost=1.12..1358236.53 rows=99823 width=1250) (actual time=0.869..159.735 rows=51 loops=1)
              Buffers: shared hit=28799
              ->  Nested Loop  (cost=0.82..1324686.95 rows=99823 width=1242) (actual time=0.809..158.861 rows=51 loops=1)
                    Buffers: shared hit=28646
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..1323430.87 rows=99823 width=1235) (actual time=0.752..158.675 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 28258
                          Buffers: shared hit=28643
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.059..0.077 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.040..0.043 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.039..0.040 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.043..0.044 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.015..0.015 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 2.949 ms
Execution Time: 160.115 ms
```
## req8 sort=-requester_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.requester_id, r.updated_at, r.id) < ('f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid, '2025-12-21 13:47:19.026143+00'::timestamptz, 'ab2f1f80-16af-53e0-3991-43e911b7bb41'::uuid) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..702.07 rows=51 width=1277) (actual time=0.604..2.393 rows=51 loops=1)
  Buffers: shared hit=212
  ->  Result  (cost=1.12..1286002.58 rows=93568 width=1277) (actual time=0.601..2.376 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=212
        ->  Nested Loop Left Join  (cost=1.12..1285300.82 rows=93568 width=1250) (actual time=0.580..2.291 rows=51 loops=1)
              Buffers: shared hit=212
              ->  Nested Loop  (cost=0.82..1253737.16 rows=93568 width=1242) (actual time=0.450..1.682 rows=51 loops=1)
                    Buffers: shared hit=59
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..1252559.26 rows=93568 width=1235) (actual time=0.275..1.453 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(requester_id, updated_at, id) < ROW('f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid, '2025-12-21 13:47:19.026143+00'::timestamp with time zone, 'ab2f1f80-16af-53e0-3991-43e911b7bb41'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Buffers: shared hit=56
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (never executed)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (never executed)
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.003..0.004 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.133..0.135 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 4.109 ms
Execution Time: 3.693 ms
```
## admin sort=-requester_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..137.31 rows=51 width=1277) (actual time=0.206..1.274 rows=51 loops=1)
  Buffers: shared hit=212
  ->  Result  (cost=1.12..532156.42 rows=199281 width=1277) (actual time=0.205..1.265 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=212
        ->  Nested Loop Left Join  (cost=1.12..530661.81 rows=199281 width=1250) (actual time=0.192..1.211 rows=51 loops=1)
              Buffers: shared hit=212
              ->  Nested Loop  (cost=0.82..465534.46 rows=199281 width=1242) (actual time=0.149..0.860 rows=51 loops=1)
                    Buffers: shared hit=59
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..463035.15 rows=199281 width=1235) (actual time=0.111..0.790 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Buffers: shared hit=56
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.031..0.032 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=530
Planning Time: 1.848 ms
Execution Time: 1.435 ms
```
## admin sort=-requester_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND (r.requester_id, r.updated_at, r.id) < ('ff963e93-89fd-bc6b-9bbb-89bf5e9db48e'::uuid, '2026-02-14 07:21:54.71314+00'::timestamptz, '053c26fd-7d03-f872-246d-661d005c7a3a'::uuid) ORDER BY r.requester_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..137.67 rows=51 width=1277) (actual time=0.324..1.986 rows=51 loops=1)
  Buffers: shared hit=212
  ->  Result  (cost=1.12..531057.31 rows=198343 width=1277) (actual time=0.322..1.972 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=212
        ->  Nested Loop Left Join  (cost=1.12..529569.74 rows=198343 width=1250) (actual time=0.300..1.895 rows=51 loops=1)
              Buffers: shared hit=212
              ->  Nested Loop  (cost=0.82..464740.20 rows=198343 width=1242) (actual time=0.228..1.314 rows=51 loops=1)
                    Buffers: shared hit=59
                    ->  Index Scan using records_list_requester on records r  (cost=0.55..462252.62 rows=198343 width=1235) (actual time=0.162..1.199 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(requester_id, updated_at, id) < ROW('ff963e93-89fd-bc6b-9bbb-89bf5e9db48e'::uuid, '2026-02-14 07:21:54.71314+00'::timestamp with time zone, '053c26fd-7d03-f872-246d-661d005c7a3a'::uuid)))
                          Buffers: shared hit=56
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.052..0.054 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=519
Planning Time: 2.401 ms
Execution Time: 2.180 ms
```
## agent sort=-assignment_group_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..677.18 rows=51 width=1277) (actual time=7.238..175.772 rows=51 loops=1)
  Buffers: shared hit=34351
  ->  Result  (cost=1.12..1361708.01 rows=102724 width=1277) (actual time=7.236..175.707 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=34351
        ->  Nested Loop Left Join  (cost=1.12..1360937.58 rows=102724 width=1250) (actual time=7.205..175.508 rows=51 loops=1)
              Buffers: shared hit=34351
              ->  Nested Loop  (cost=0.82..1326466.92 rows=102724 width=1242) (actual time=7.121..173.993 rows=51 loops=1)
                    Buffers: shared hit=34198
                    ->  Index Scan using records_list_group on records r  (cost=0.55..1325174.58 rows=102724 width=1235) (actual time=7.043..173.629 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 33735
                          Buffers: shared hit=34195
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.227..0.273 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.160..0.162 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.061..0.062 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.096..0.097 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.053..0.054 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.025..0.025 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=610
Planning Time: 4.714 ms
Execution Time: 176.251 ms
```
## agent sort=-assignment_group_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND (r.assignment_group_id, r.updated_at, r.id) < ('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamptz, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..690.95 rows=51 width=1277) (actual time=0.285..1.695 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.12..1205475.38 rows=89122 width=1277) (actual time=0.283..1.681 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.12..1204806.97 rows=89122 width=1250) (actual time=0.265..1.604 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.82..1174654.92 rows=89122 width=1242) (actual time=0.208..1.320 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_list_group on records r  (cost=0.55..1173532.60 rows=89122 width=1235) (actual time=0.162..1.222 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Buffers: shared hit=55
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (never executed)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (never executed)
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (never executed)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (never executed)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.037..0.038 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=599
Planning Time: 3.673 ms
Execution Time: 2.024 ms
```
## req1 sort=-assignment_group_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..691.78 rows=51 width=1277) (actual time=0.820..27.711 rows=51 loops=1)
  Buffers: shared hit=3298
  ->  Result  (cost=1.12..1359167.80 rows=100364 width=1277) (actual time=0.818..27.686 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=3298
        ->  Nested Loop Left Join  (cost=1.12..1358415.07 rows=100364 width=1250) (actual time=0.804..27.592 rows=51 loops=1)
              Buffers: shared hit=3298
              ->  Nested Loop  (cost=0.82..1324693.72 rows=100364 width=1242) (actual time=0.757..27.090 rows=51 loops=1)
                    Buffers: shared hit=3145
                    ->  Index Scan using records_list_group on records r  (cost=0.55..1323430.87 rows=100364 width=1235) (actual time=0.708..26.926 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 3049
                          Buffers: shared hit=3142
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.032..0.034 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.030..0.030 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.035..0.036 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=610
Planning Time: 3.035 ms
Execution Time: 27.981 ms
```
## req1 sort=-assignment_group_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.assignment_group_id, r.updated_at, r.id) < ('e3c52f7e-6344-75f6-938a-7d2ef574a7d1'::uuid, '2026-06-25 05:48:51.575508+00'::timestamptz, '56e4ae8f-3d68-bd60-e400-dfa962068d41'::uuid) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..707.14 rows=51 width=1277) (actual time=0.304..25.993 rows=51 loops=1)
  Buffers: shared hit=2709
  ->  Result  (cost=1.12..1189845.68 rows=85950 width=1277) (actual time=0.303..25.967 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=2709
        ->  Nested Loop Left Join  (cost=1.12..1189201.05 rows=85950 width=1250) (actual time=0.292..25.875 rows=51 loops=1)
              Buffers: shared hit=2709
              ->  Nested Loop  (cost=0.82..1160056.12 rows=85950 width=1242) (actual time=0.241..25.240 rows=51 loops=1)
                    Buffers: shared hit=2556
                    ->  Index Scan using records_list_group on records r  (cost=0.55..1158973.45 rows=85950 width=1235) (actual time=0.173..25.050 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(assignment_group_id, updated_at, id) < ROW('e3c52f7e-6344-75f6-938a-7d2ef574a7d1'::uuid, '2026-06-25 05:48:51.575508+00'::timestamp with time zone, '56e4ae8f-3d68-bd60-e400-dfa962068d41'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 2467
                          Buffers: shared hit=2553
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.030..0.033 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.028..0.030 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.028..0.028 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.061..0.062 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=599
Planning Time: 3.042 ms
Execution Time: 26.209 ms
```
## req8 sort=-assignment_group_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..695.43 rows=51 width=1277) (actual time=4.190..86.972 rows=51 loops=1)
  Buffers: shared hit=17534
  ->  Result  (cost=1.12..1358985.21 rows=99823 width=1277) (actual time=4.188..86.931 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=17534
        ->  Nested Loop Left Join  (cost=1.12..1358236.53 rows=99823 width=1250) (actual time=4.177..86.807 rows=51 loops=1)
              Buffers: shared hit=17534
              ->  Nested Loop  (cost=0.82..1324686.95 rows=99823 width=1242) (actual time=4.128..85.961 rows=51 loops=1)
                    Buffers: shared hit=17381
                    ->  Index Scan using records_list_group on records r  (cost=0.55..1323430.87 rows=99823 width=1235) (actual time=4.077..85.750 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 17121
                          Buffers: shared hit=17378
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.056..0.073 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.034..0.036 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.033..0.034 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.035..0.036 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=610
Planning Time: 2.582 ms
Execution Time: 87.163 ms
```
## req8 sort=-assignment_group_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.assignment_group_id, r.updated_at, r.id) < ('2f067c2d-b39a-c447-6bd8-6f26d53e8105'::uuid, '2026-07-15 00:04:53.763927+00'::timestamptz, '2e74102c-3b9f-efda-68ab-1b412e23ca0f'::uuid) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..831.56 rows=51 width=1277) (actual time=3.258..132.161 rows=51 loops=1)
  Buffers: shared hit=18100
  ->  Result  (cost=1.12..247358.43 rows=15191 width=1277) (actual time=3.255..132.086 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=18100
        ->  Nested Loop Left Join  (cost=1.12..247244.50 rows=15191 width=1250) (actual time=3.233..131.867 rows=51 loops=1)
              Buffers: shared hit=18100
              ->  Nested Loop  (cost=0.82..240565.49 rows=15191 width=1242) (actual time=3.173..129.978 rows=51 loops=1)
                    Buffers: shared hit=17947
                    ->  Index Scan using records_list_group on records r  (cost=0.55..240367.31 rows=15191 width=1235) (actual time=3.093..129.573 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(assignment_group_id, updated_at, id) < ROW('2f067c2d-b39a-c447-6bd8-6f26d53e8105'::uuid, '2026-07-15 00:04:53.763927+00'::timestamp with time zone, '2e74102c-3b9f-efda-68ab-1b412e23ca0f'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 17682
                          Buffers: shared hit=17944
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.099..0.124 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.064..0.066 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.062..0.062 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.059..0.060 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.43 rows=1 width=40) (actual time=0.032..0.032 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.43 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=599
Planning Time: 4.734 ms
Execution Time: 132.601 ms
```
## admin sort=-assignment_group_id page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..137.31 rows=51 width=1277) (actual time=0.221..1.135 rows=51 loops=1)
  Buffers: shared hit=212
  ->  Result  (cost=1.12..532156.42 rows=199281 width=1277) (actual time=0.220..1.125 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=212
        ->  Nested Loop Left Join  (cost=1.12..530661.81 rows=199281 width=1250) (actual time=0.207..1.071 rows=51 loops=1)
              Buffers: shared hit=212
              ->  Nested Loop  (cost=0.82..465534.46 rows=199281 width=1242) (actual time=0.171..0.903 rows=51 loops=1)
                    Buffers: shared hit=59
                    ->  Index Scan using records_list_group on records r  (cost=0.55..463035.15 rows=199281 width=1235) (actual time=0.133..0.832 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Buffers: shared hit=56
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.029..0.030 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=548
Planning Time: 1.789 ms
Execution Time: 1.238 ms
```
## admin sort=-assignment_group_id page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND (r.assignment_group_id, r.updated_at, r.id) < ('fcb88798-7208-926a-2cd2-13ba3946b83b'::uuid, '2026-09-30 08:45:08.261119+00'::timestamptz, 'de81b909-7581-fb44-b045-f41d96a280bd'::uuid) ORDER BY r.assignment_group_id DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT 51
```
```
Limit  (cost=1.12..138.27 rows=51 width=1277) (actual time=0.336..1.568 rows=51 loops=1)
  Buffers: shared hit=212
  ->  Result  (cost=1.12..527021.53 rows=195984 width=1277) (actual time=0.334..1.556 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=212
        ->  Nested Loop Left Join  (cost=1.12..525551.65 rows=195984 width=1250) (actual time=0.315..1.491 rows=51 loops=1)
              Buffers: shared hit=212
              ->  Nested Loop  (cost=0.82..461471.06 rows=195984 width=1242) (actual time=0.264..1.240 rows=51 loops=1)
                    Buffers: shared hit=59
                    ->  Index Scan using records_list_group on records r  (cost=0.55..459012.97 rows=195984 width=1235) (actual time=0.214..1.146 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(assignment_group_id, updated_at, id) < ROW('fcb88798-7208-926a-2cd2-13ba3946b83b'::uuid, '2026-09-30 08:45:08.261119+00'::timestamp with time zone, 'de81b909-7581-fb44-b045-f41d96a280bd'::uuid)))
                          Buffers: shared hit=56
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.037..0.038 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=537
Planning Time: 2.512 ms
Execution Time: 1.732 ms
```
