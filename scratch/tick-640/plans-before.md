## agent sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..1147.68 rows=51 width=1277) (actual time=1.761..10.338 rows=51 loops=1)
  Buffers: shared hit=968
  ->  Result  (cost=1.00..2309635.90 rows=102724 width=1277) (actual time=1.759..10.317 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=968
        ->  Nested Loop Left Join  (cost=1.00..2308865.47 rows=102724 width=1250) (actual time=1.737..10.238 rows=51 loops=1)
              Buffers: shared hit=968
              ->  Nested Loop  (cost=0.70..2274394.82 rows=102724 width=1242) (actual time=1.567..9.672 rows=51 loops=1)
                    Buffers: shared hit=815
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..2273102.47 rows=102724 width=1235) (actual time=1.399..9.438 rows=51 loops=1)
                          Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                          Filter: ((entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2)))
                          Rows Removed by Filter: 736
                          Buffers: shared hit=812
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.501..0.620 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.448..0.460 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.189..0.191 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.257..0.259 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.003..0.004 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.155..0.157 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=519
Planning Time: 4.102 ms
Execution Time: 10.673 ms
```
count:
```
Aggregate  (cost=1326887.55..1326887.56 rows=1 width=8) (actual time=994.064..994.072 rows=1 loops=1)
  Buffers: shared hit=69994
  ->  Result  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=653.917..990.705 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69994
        ->  Index Scan using records_number_key on records r  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=29.884..362.353 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=69994
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.175..0.206 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.132..0.135 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.059..0.059 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.072..0.072 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=340
Planning Time: 1.880 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.235 ms, Inlining 187.136 ms, Optimization 267.960 ms, Emission 198.074 ms, Total 655.405 ms
Execution Time: 1022.215 ms
```
## agent sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND (r.updated_at, r.id) < ('2026-09-30 23:25:18.681999+00'::timestamptz, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..1159.59 rows=51 width=1277) (actual time=1.095..10.310 rows=51 loops=1)
  Buffers: shared hit=947
  ->  Result  (cost=1.00..2235980.16 rows=98425 width=1277) (actual time=1.093..10.293 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=947
        ->  Nested Loop Left Join  (cost=1.00..2235241.97 rows=98425 width=1250) (actual time=1.076..10.212 rows=51 loops=1)
              Buffers: shared hit=947
              ->  Nested Loop  (cost=0.70..2202136.26 rows=98425 width=1242) (actual time=1.017..9.618 rows=51 loops=1)
                    Buffers: shared hit=794
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..2200897.65 rows=98425 width=1235) (actual time=0.958..9.485 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                          Filter: ((entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2)))
                          Rows Removed by Filter: 719
                          Buffers: shared hit=791
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.137..0.167 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.110..0.112 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.039..0.040 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.070..0.070 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.042..0.043 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=508
Planning Time: 2.605 ms
Execution Time: 10.574 ms
```
count:
```
Aggregate  (cost=1326887.55..1326887.56 rows=1 width=8) (actual time=855.614..855.620 rows=1 loops=1)
  Buffers: shared hit=69994
  ->  Result  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=536.033..852.311 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69994
        ->  Index Scan using records_number_key on records r  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=32.005..344.352 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=69994
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.205..0.236 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.156..0.158 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.074..0.074 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.080..0.080 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=340
Planning Time: 1.819 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.438 ms, Inlining 169.731 ms, Optimization 212.356 ms, Emission 153.141 ms, Total 537.667 ms
Execution Time: 887.540 ms
```
## req1 sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..1172.45 rows=51 width=1277) (actual time=0.558..38.476 rows=51 loops=1)
  Buffers: shared hit=4368
  ->  Result  (cost=1.00..2305346.02 rows=100365 width=1277) (actual time=0.557..38.441 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=4368
        ->  Nested Loop Left Join  (cost=1.00..2304593.28 rows=100365 width=1250) (actual time=0.542..38.269 rows=51 loops=1)
              Buffers: shared hit=4368
              ->  Nested Loop  (cost=0.70..2270871.61 rows=100365 width=1242) (actual time=0.497..36.225 rows=51 loops=1)
                    Buffers: shared hit=4215
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..2269608.76 rows=100365 width=1235) (actual time=0.458..36.038 rows=51 loops=1)
                          Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                          Filter: ((entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2)))
                          Rows Removed by Filter: 4126
                          Buffers: shared hit=4212
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.126..0.129 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.125..0.127 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.123..0.124 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.030..0.031 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.038..0.038 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.032..0.032 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=519
Planning Time: 2.483 ms
Execution Time: 38.686 ms
```
count:
```
Aggregate  (cost=1325137.94..1325137.95 rows=1 width=8) (actual time=691.766..691.772 rows=1 loops=1)
  Buffers: shared hit=69980
  ->  Result  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=417.417..690.511 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69980
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=23.187..294.811 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=69980
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.098..0.101 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.090..0.093 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.087..0.088 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 2.048 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.636 ms, Inlining 133.548 ms, Optimization 172.036 ms, Emission 111.234 ms, Total 419.454 ms
Execution Time: 719.325 ms
```
## req1 sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.updated_at, r.id) < ('2026-09-29 03:10:06.448328+00'::timestamptz, '0150dafd-b1c7-41f0-99ea-b49cdcff90ec'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..1229.29 rows=51 width=1277) (actual time=6.517..52.177 rows=51 loops=1)
  Buffers: shared hit=4586
  ->  Result  (cost=1.00..1997115.34 rows=82922 width=1277) (actual time=6.513..52.129 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=4586
        ->  Nested Loop Left Join  (cost=1.00..1996493.43 rows=82922 width=1250) (actual time=6.481..51.859 rows=51 loops=1)
              Buffers: shared hit=4586
              ->  Nested Loop  (cost=0.70..1968309.88 rows=82922 width=1242) (actual time=6.314..49.634 rows=51 loops=1)
                    Buffers: shared hit=4433
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..1967265.06 rows=82922 width=1235) (actual time=6.226..49.260 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:10:06.448328+00'::timestamp with time zone, '0150dafd-b1c7-41f0-99ea-b49cdcff90ec'::uuid)))
                          Filter: ((entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2)))
                          Rows Removed by Filter: 4343
                          Buffers: shared hit=4430
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.056..0.058 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.050..0.052 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.049..0.049 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.064..0.066 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.039..0.040 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.030..0.030 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=508
Planning Time: 7.996 ms
Execution Time: 52.805 ms
```
count:
```
Aggregate  (cost=1325137.94..1325137.95 rows=1 width=8) (actual time=1543.822..1543.831 rows=1 loops=1)
  Buffers: shared hit=69980
  ->  Result  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=845.698..1539.265 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69980
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=69.041..756.749 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=69980
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.106..0.109 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.092..0.094 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.089..0.089 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 9.131 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.868 ms, Inlining 306.252 ms, Optimization 320.054 ms, Emission 218.515 ms, Total 847.689 ms
Execution Time: 1614.610 ms
```
## req8 sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..1178.71 rows=51 width=1277) (actual time=2.864..238.233 rows=51 loops=1)
  Buffers: shared hit=18751
  ->  Result  (cost=1.00..2305163.09 rows=99823 width=1277) (actual time=2.860..238.095 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=18751
        ->  Nested Loop Left Join  (cost=1.00..2304414.42 rows=99823 width=1250) (actual time=2.826..237.340 rows=51 loops=1)
              Buffers: shared hit=18751
              ->  Nested Loop  (cost=0.70..2270864.84 rows=99823 width=1242) (actual time=2.739..232.486 rows=51 loops=1)
                    Buffers: shared hit=18598
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..2269608.76 rows=99823 width=1235) (actual time=2.537..231.655 rows=51 loops=1)
                          Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                          Filter: ((entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2)))
                          Rows Removed by Filter: 18405
                          Buffers: shared hit=18595
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.321..0.510 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.215..0.218 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.213..0.213 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.006..0.006 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.171..0.173 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.084..0.084 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.050..0.050 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=519
Planning Time: 6.214 ms
Execution Time: 238.751 ms
```
count:
```
Aggregate  (cost=1325136.58..1325136.59 rows=1 width=8) (actual time=871.664..871.670 rows=1 loops=1)
  Buffers: shared hit=69982
  ->  Result  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=511.781..870.867 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69982
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=25.302..383.606 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=69982
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.125..0.145 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.083..0.085 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.081..0.082 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 2.522 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 4.937 ms, Inlining 183.007 ms, Optimization 202.162 ms, Emission 125.304 ms, Total 515.410 ms
Execution Time: 918.039 ms
```
## req8 sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.updated_at, r.id) < ('2026-02-15 00:27:00.844898+00'::timestamptz, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..1575.81 rows=51 width=1277) (actual time=8.786..124.223 rows=51 loops=1)
  Buffers: shared hit=20693
  ->  Result  (cost=1.00..450335.81 rows=14584 width=1277) (actual time=8.784..124.159 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=20693
        ->  Nested Loop Left Join  (cost=1.00..450226.43 rows=14584 width=1250) (actual time=8.765..123.859 rows=51 loops=1)
              Buffers: shared hit=20693
              ->  Nested Loop  (cost=0.70..443740.15 rows=14584 width=1242) (actual time=8.693..121.832 rows=51 loops=1)
                    Buffers: shared hit=20540
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..443549.56 rows=14584 width=1235) (actual time=8.583..121.474 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (ROW(updated_at, id) < ROW('2026-02-15 00:27:00.844898+00'::timestamp with time zone, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid)))
                          Filter: ((entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2)))
                          Rows Removed by Filter: 20334
                          Buffers: shared hit=20537
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.062..0.079 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.040..0.042 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.039..0.040 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.072..0.074 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.43 rows=1 width=40) (actual time=0.035..0.035 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.43 rows=1 width=40) (actual time=0.019..0.019 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=508
Planning Time: 2.235 ms
Execution Time: 124.481 ms
```
count:
```
Aggregate  (cost=1325136.58..1325136.59 rows=1 width=8) (actual time=753.849..753.857 rows=1 loops=1)
  Buffers: shared hit=69982
  ->  Result  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=440.791..753.018 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69982
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=28.432..340.049 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=69982
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.133..0.154 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.087..0.090 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.085..0.085 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 5.508 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 1.738 ms, Inlining 118.519 ms, Optimization 174.030 ms, Emission 147.146 ms, Total 441.434 ms
Execution Time: 781.274 ms
```
## admin sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..158.34 rows=51 width=1277) (actual time=0.199..1.329 rows=51 loops=1)
  Buffers: shared hit=211
  ->  Result  (cost=1.00..614830.33 rows=199282 width=1277) (actual time=0.197..1.320 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=211
        ->  Nested Loop Left Join  (cost=1.00..613335.71 rows=199282 width=1250) (actual time=0.177..1.271 rows=51 loops=1)
              Buffers: shared hit=211
              ->  Nested Loop  (cost=0.70..548208.04 rows=199282 width=1242) (actual time=0.128..0.936 rows=51 loops=1)
                    Buffers: shared hit=58
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..545708.72 rows=199282 width=1235) (actual time=0.089..0.865 rows=51 loops=1)
                          Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                          Filter: (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                          Buffers: shared hit=55
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.032..0.033 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=454
Planning Time: 1.985 ms
Execution Time: 1.478 ms
```
count:
```
Finalize Aggregate  (cost=14759.39..14759.40 rows=1 width=8) (actual time=180.597..194.288 rows=1 loops=1)
  Buffers: shared hit=3321
  ->  Gather  (cost=14759.17..14759.38 rows=2 width=8) (actual time=180.348..194.269 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=3321
        ->  Partial Aggregate  (cost=13759.17..13759.18 rows=1 width=8) (actual time=173.588..173.591 rows=1 loops=3)
              Buffers: shared hit=3321
              ->  Result  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.250..154.060 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=3321
                    ->  Parallel Index Only Scan using records_number_key on records r  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.211..124.375 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=3321
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=210
Planning Time: 1.515 ms
Execution Time: 194.471 ms
```
## admin sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND (r.updated_at, r.id) < ('2026-09-30 23:59:59.561867+00'::timestamptz, 'fb3ab87c-de21-eb9c-909b-5f44410a93b9'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..158.62 rows=51 width=1277) (actual time=0.258..2.368 rows=51 loops=1)
  Buffers: shared hit=210
  ->  Result  (cost=1.00..615814.22 rows=199245 width=1277) (actual time=0.256..2.353 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=210
        ->  Nested Loop Left Join  (cost=1.00..614319.88 rows=199245 width=1250) (actual time=0.233..2.283 rows=51 loops=1)
              Buffers: shared hit=210
              ->  Nested Loop  (cost=0.70..549203.96 rows=199245 width=1242) (actual time=0.168..1.607 rows=51 loops=1)
                    Buffers: shared hit=57
                    ->  Index Scan using records_tenant_updated on records r  (cost=0.42..546705.10 rows=199245 width=1235) (actual time=0.123..1.509 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:59:59.561867+00'::timestamp with time zone, 'fb3ab87c-de21-eb9c-909b-5f44410a93b9'::uuid)))
                          Filter: (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                          Buffers: shared hit=54
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.036..0.037 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=443
Planning Time: 2.473 ms
Execution Time: 2.575 ms
```
count:
```
Finalize Aggregate  (cost=14759.39..14759.40 rows=1 width=8) (actual time=109.812..120.246 rows=1 loops=1)
  Buffers: shared hit=3321
  ->  Gather  (cost=14759.17..14759.38 rows=2 width=8) (actual time=109.577..120.232 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=3321
        ->  Partial Aggregate  (cost=13759.17..13759.18 rows=1 width=8) (actual time=105.790..105.792 rows=1 loops=3)
              Buffers: shared hit=3321
              ->  Result  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.084..94.143 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=3321
                    ->  Parallel Index Only Scan using records_number_key on records r  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.072..76.273 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=3321
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=210
Planning Time: 1.031 ms
Execution Time: 120.327 ms
```
## agent sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1332280.25..1332280.37 rows=51 width=1277) (actual time=1389.612..1389.648 rows=51 loops=1)
  Buffers: shared hit=70059
  ->  Sort  (cost=1332280.25..1332537.06 rows=102724 width=1277) (actual time=499.457..499.486 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 201kB
        Buffers: shared hit=70059
        ->  Result  (cost=144.46..1328853.15 rows=102724 width=1277) (actual time=34.190..472.236 rows=24666 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=70053
              ->  Hash Left Join  (cost=144.46..1328082.72 rows=102724 width=1250) (actual time=34.151..456.189 rows=24666 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=70053
                    ->  Nested Loop  (cost=0.42..1327668.98 rows=102724 width=1242) (actual time=33.180..444.585 rows=24666 loops=1)
                          Buffers: shared hit=69999
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.038..0.234 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Index Scan using records_number_key on records r  (cost=0.42..1326630.74 rows=102724 width=1235) (actual time=33.130..438.028 rows=24666 loops=1)
                                Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                                Rows Removed by Filter: 375334
                                Buffers: shared hit=69994
                                SubPlan 2
                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.225..0.269 rows=3 loops=1)
                                        Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                        Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                        Heap Blocks: exact=3
                                        Buffers: shared hit=13
                                        ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.145..0.147 rows=0 loops=1)
                                              Buffers: shared hit=10
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.064..0.064 rows=3 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                    Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.079..0.079 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                    Buffers: shared hit=8
                    ->  Hash  (cost=143.38..143.38 rows=52 width=40) (actual time=0.892..0.897 rows=52 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.70..143.38 rows=52 width=40) (actual time=0.215..0.854 rows=52 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.70..143.38 rows=52 width=40) (actual time=0.188..0.811 rows=52 loops=1)
                                      Recheck Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=52 width=0) (actual time=0.043..0.044 rows=52 loops=1)
                                            Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=519
Planning Time: 3.242 ms
JIT:
  Functions: 41
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 4.618 ms, Inlining 178.359 ms, Optimization 424.434 ms, Emission 319.636 ms, Total 927.046 ms
Execution Time: 1424.199 ms
```
count:
```
Aggregate  (cost=1326887.55..1326887.56 rows=1 width=8) (actual time=727.939..727.944 rows=1 loops=1)
  Buffers: shared hit=69994
  ->  Result  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=423.067..725.145 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69994
        ->  Index Scan using records_number_key on records r  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=22.833..321.099 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=69994
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.161..0.186 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.122..0.124 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.057..0.057 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.064..0.064 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=340
Planning Time: 1.772 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 4.208 ms, Inlining 134.773 ms, Optimization 174.497 ms, Emission 113.194 ms, Total 426.672 ms
Execution Time: 757.968 ms
```
## agent sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND (r.created_at, r.id) < ('2026-09-24 15:42:07.067406+00'::timestamptz, '345d89b0-b073-a6ef-d0d1-641e3d71e8cd'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1333041.74..1333041.87 rows=51 width=1277) (actual time=1023.997..1024.032 rows=51 loops=1)
  Buffers: shared hit=70056
  ->  Sort  (cost=1333041.74..1333287.57 rows=98332 width=1277) (actual time=371.517..371.546 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 199kB
        Buffers: shared hit=70056
        ->  Result  (cost=144.46..1329761.18 rows=98332 width=1277) (actual time=25.272..351.779 rows=23715 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=70050
              ->  Hash Left Join  (cost=144.46..1329023.69 rows=98332 width=1250) (actual time=25.233..340.860 rows=23715 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=70050
                    ->  Nested Loop  (cost=0.42..1328621.47 rows=98332 width=1242) (actual time=24.616..333.202 rows=23715 loops=1)
                          Buffers: shared hit=69996
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.033..0.075 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Index Scan using records_number_key on records r  (cost=0.42..1327627.15 rows=98332 width=1235) (actual time=24.574..328.997 rows=23715 loops=1)
                                Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Filter: ((ROW(created_at, id) < ROW('2026-09-24 15:42:07.067406+00'::timestamp with time zone, '345d89b0-b073-a6ef-d0d1-641e3d71e8cd'::uuid)) AND (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2)))
                                Rows Removed by Filter: 376285
                                Buffers: shared hit=69991
                                SubPlan 2
                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.176..0.205 rows=3 loops=1)
                                        Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                        Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                        Heap Blocks: exact=3
                                        Buffers: shared hit=13
                                        ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.133..0.135 rows=0 loops=1)
                                              Buffers: shared hit=10
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.063..0.063 rows=3 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                    Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.069..0.069 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                    Buffers: shared hit=8
                    ->  Hash  (cost=143.38..143.38 rows=52 width=40) (actual time=0.564..0.567 rows=52 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.70..143.38 rows=52 width=40) (actual time=0.075..0.538 rows=52 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.70..143.38 rows=52 width=40) (actual time=0.062..0.513 rows=52 loops=1)
                                      Recheck Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=52 width=0) (actual time=0.039..0.039 rows=52 loops=1)
                                            Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=508
Planning Time: 3.288 ms
JIT:
  Functions: 41
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 5.050 ms, Inlining 143.333 ms, Optimization 321.609 ms, Emission 211.491 ms, Total 681.483 ms
Execution Time: 1057.674 ms
```
count:
```
Aggregate  (cost=1326887.55..1326887.56 rows=1 width=8) (actual time=731.176..731.181 rows=1 loops=1)
  Buffers: shared hit=69994
  ->  Result  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=417.560..728.083 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69994
        ->  Index Scan using records_number_key on records r  (cost=0.42..1326630.74 rows=102724 width=0) (actual time=28.287..334.983 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=69994
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.155..0.183 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.116..0.118 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.045..0.046 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.069..0.070 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=330
Planning Time: 1.764 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.174 ms, Inlining 124.790 ms, Optimization 169.988 ms, Emission 121.857 ms, Total 418.809 ms
Execution Time: 759.981 ms
```
## req1 sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1330410.35..1330410.48 rows=51 width=1277) (actual time=857.695..857.730 rows=51 loops=1)
  Buffers: shared hit=70045
  ->  Sort  (cost=1330410.35..1330661.26 rows=100365 width=1277) (actual time=323.184..323.212 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 189kB
        Buffers: shared hit=70045
        ->  Result  (cost=144.46..1327061.96 rows=100365 width=1277) (actual time=24.507..315.722 rows=5626 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=70039
              ->  Hash Left Join  (cost=144.46..1326309.22 rows=100365 width=1250) (actual time=24.474..310.619 rows=5626 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=70039
                    ->  Nested Loop  (cost=0.42..1325901.67 rows=100365 width=1242) (actual time=23.994..306.116 rows=5626 loops=1)
                          Buffers: shared hit=69985
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.034..0.074 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=100365 width=1235) (actual time=23.952..304.075 rows=5626 loops=1)
                                Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                                Rows Removed by Filter: 394374
                                Buffers: shared hit=69980
                                SubPlan 2
                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.106..0.109 rows=0 loops=1)
                                        Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                        Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                        Buffers: shared hit=2
                                        ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.103..0.105 rows=0 loops=1)
                                              Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.102..0.102 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                    Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Hash  (cost=143.38..143.38 rows=52 width=40) (actual time=0.453..0.455 rows=52 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.70..143.38 rows=52 width=40) (actual time=0.068..0.436 rows=52 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.70..143.38 rows=52 width=40) (actual time=0.060..0.420 rows=52 loops=1)
                                      Recheck Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=52 width=0) (actual time=0.029..0.030 rows=52 loops=1)
                                            Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=519
Planning Time: 2.353 ms
JIT:
  Functions: 41
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 3.274 ms, Inlining 107.128 ms, Optimization 260.433 ms, Emission 190.400 ms, Total 561.235 ms
Execution Time: 886.560 ms
```
count:
```
Aggregate  (cost=1325137.94..1325137.95 rows=1 width=8) (actual time=816.713..816.720 rows=1 loops=1)
  Buffers: shared hit=69980
  ->  Result  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=555.274..815.509 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69980
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=26.206..285.097 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=69980
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.084..0.088 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.077..0.080 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.075..0.075 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 5.948 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 3.475 ms, Inlining 203.899 ms, Optimization 220.660 ms, Emission 130.155 ms, Total 558.190 ms
Execution Time: 859.527 ms
```
## req1 sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.created_at, r.id) < ('2026-09-12 13:51:07.994144+00'::timestamptz, '1ad845b3-fc23-836e-dd8c-1da77c12053f'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1330493.62..1330493.75 rows=51 width=1277) (actual time=1391.650..1391.689 rows=51 loops=1)
  Buffers: shared hit=70045
  ->  Sort  (cost=1330493.62..1330701.85 rows=83293 width=1277) (actual time=439.518..439.549 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 197kB
        Buffers: shared hit=70045
        ->  Result  (cost=144.46..1327714.79 rows=83293 width=1277) (actual time=33.933..429.946 rows=4675 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=70039
              ->  Hash Left Join  (cost=144.46..1327090.09 rows=83293 width=1250) (actual time=33.843..423.152 rows=4675 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=70039
                    ->  Nested Loop  (cost=0.42..1326727.36 rows=83293 width=1242) (actual time=33.232..417.300 rows=4675 loops=1)
                          Buffers: shared hit=69985
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.273..0.320 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Index Scan using records_number_key on records r  (cost=0.42..1325883.43 rows=83293 width=1235) (actual time=32.946..414.684 rows=4675 loops=1)
                                Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Filter: ((ROW(created_at, id) < ROW('2026-09-12 13:51:07.994144+00'::timestamp with time zone, '1ad845b3-fc23-836e-dd8c-1da77c12053f'::uuid)) AND (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2)))
                                Rows Removed by Filter: 395325
                                Buffers: shared hit=69980
                                SubPlan 2
                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.071..0.074 rows=0 loops=1)
                                        Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                        Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                        Buffers: shared hit=2
                                        ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.067..0.068 rows=0 loops=1)
                                              Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.064..0.065 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                    Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Hash  (cost=143.38..143.38 rows=52 width=40) (actual time=0.571..0.574 rows=52 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.70..143.38 rows=52 width=40) (actual time=0.074..0.551 rows=52 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.70..143.38 rows=52 width=40) (actual time=0.064..0.529 rows=52 loops=1)
                                      Recheck Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=52 width=0) (actual time=0.037..0.038 rows=52 loops=1)
                                            Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=508
Planning Time: 4.115 ms
JIT:
  Functions: 41
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 4.488 ms, Inlining 237.209 ms, Optimization 421.827 ms, Emission 325.288 ms, Total 988.813 ms
Execution Time: 1428.687 ms
```
count:
```
Aggregate  (cost=1325137.94..1325137.95 rows=1 width=8) (actual time=696.047..696.105 rows=1 loops=1)
  Buffers: shared hit=69980
  ->  Result  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=455.150..695.000 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69980
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=100365 width=0) (actual time=26.153..264.679 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=69980
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.138..0.194 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.116..0.171 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.113..0.114 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 2.091 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.995 ms, Inlining 130.203 ms, Optimization 177.910 ms, Emission 146.276 ms, Total 457.384 ms
Execution Time: 729.007 ms
```
## req8 sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1330381.36..1330381.49 rows=51 width=1277) (actual time=844.871..844.942 rows=51 loops=1)
  Buffers: shared hit=70047
  ->  Sort  (cost=1330381.36..1330630.92 rows=99823 width=1277) (actual time=328.151..328.215 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 203kB
        Buffers: shared hit=70047
        ->  Result  (cost=144.46..1327051.05 rows=99823 width=1277) (actual time=23.871..325.541 rows=1111 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=70041
              ->  Hash Left Join  (cost=144.46..1326302.38 rows=99823 width=1250) (actual time=23.840..323.702 rows=1111 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=70041
                    ->  Nested Loop  (cost=0.42..1325896.25 rows=99823 width=1242) (actual time=23.454..321.936 rows=1111 loops=1)
                          Buffers: shared hit=69987
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.033..0.081 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=99823 width=1235) (actual time=23.413..321.154 rows=1111 loops=1)
                                Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                                Rows Removed by Filter: 398889
                                Buffers: shared hit=69982
                                SubPlan 2
                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.102..0.122 rows=2 loops=1)
                                        Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                        Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                        Heap Blocks: exact=2
                                        Buffers: shared hit=4
                                        ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.068..0.071 rows=0 loops=1)
                                              Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.066..0.066 rows=2 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                                    Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Hash  (cost=143.38..143.38 rows=52 width=40) (actual time=0.361..0.364 rows=52 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.70..143.38 rows=52 width=40) (actual time=0.063..0.348 rows=52 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.70..143.38 rows=52 width=40) (actual time=0.053..0.330 rows=52 loops=1)
                                      Recheck Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=52 width=0) (actual time=0.026..0.026 rows=52 loops=1)
                                            Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=519
Planning Time: 3.492 ms
JIT:
  Functions: 41
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 3.285 ms, Inlining 111.813 ms, Optimization 250.389 ms, Emission 176.926 ms, Total 542.413 ms
Execution Time: 872.878 ms
```
count:
```
Aggregate  (cost=1325136.58..1325136.59 rows=1 width=8) (actual time=814.032..814.038 rows=1 loops=1)
  Buffers: shared hit=69982
  ->  Result  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=486.067..813.378 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69982
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=29.405..356.077 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=69982
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.108..0.127 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.069..0.071 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.068..0.068 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 2.390 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.564 ms, Inlining 130.119 ms, Optimization 231.445 ms, Emission 123.393 ms, Total 487.520 ms
Execution Time: 846.715 ms
```
## req8 sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.created_at, r.id) < ('2026-01-31 15:23:22.324519+00'::timestamptz, '0014c6a0-72df-eac0-a48b-d96b066049c6'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1333166.32..1333166.45 rows=51 width=1277) (actual time=779.855..779.884 rows=51 loops=1)
  Buffers: shared hit=70473
  ->  Sort  (cost=1333166.32..1333203.08 rows=14703 width=1277) (actual time=256.908..256.931 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 209kB
        Buffers: shared hit=70473
        ->  Result  (cost=0.72..1332675.80 rows=14703 width=1277) (actual time=26.303..256.064 rows=160 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=70467
              ->  Nested Loop Left Join  (cost=0.72..1332565.53 rows=14703 width=1250) (actual time=26.268..255.546 rows=160 loops=1)
                    Buffers: shared hit=70467
                    ->  Nested Loop  (cost=0.42..1326041.46 rows=14703 width=1242) (actual time=26.207..252.049 rows=160 loops=1)
                          Buffers: shared hit=69987
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.037..0.078 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Index Scan using records_number_key on records r  (cost=0.42..1325883.43 rows=14703 width=1235) (actual time=26.161..251.734 rows=160 loops=1)
                                Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Filter: ((ROW(created_at, id) < ROW('2026-01-31 15:23:22.324519+00'::timestamp with time zone, '0014c6a0-72df-eac0-a48b-d96b066049c6'::uuid)) AND (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2)))
                                Rows Removed by Filter: 399840
                                Buffers: shared hit=69982
                                SubPlan 2
                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.099..0.118 rows=2 loops=1)
                                        Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                        Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                        Heap Blocks: exact=2
                                        Buffers: shared hit=4
                                        ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.061..0.063 rows=0 loops=1)
                                              Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.059..0.059 rows=2 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                                    Buffers: shared hit=2
                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                                    Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Result  (cost=0.29..0.43 rows=1 width=40) (actual time=0.018..0.018 rows=1 loops=160)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                          Buffers: shared hit=480
                          ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.43 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=160)
                                Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=480
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=508
Planning Time: 2.610 ms
JIT:
  Functions: 37
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.534 ms, Inlining 120.820 ms, Optimization 239.684 ms, Emission 187.526 ms, Total 550.564 ms
Execution Time: 804.464 ms
```
count:
```
Aggregate  (cost=1325136.58..1325136.59 rows=1 width=8) (actual time=942.337..942.344 rows=1 loops=1)
  Buffers: shared hit=69982
  ->  Result  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=600.334..941.668 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=69982
        ->  Index Scan using records_number_key on records r  (cost=0.42..1324887.02 rows=99823 width=0) (actual time=55.028..395.753 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=69982
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.153..0.179 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.093..0.096 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.091..0.091 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=335
Planning Time: 2.109 ms
JIT:
  Functions: 26
  Options: Inlining true, Optimization true, Expressions true, Deforming true
  Timing: Generation 2.563 ms, Inlining 203.495 ms, Optimization 249.634 ms, Emission 145.528 ms, Total 601.219 ms
Execution Time: 975.698 ms
```
## admin sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=156148.23..156148.36 rows=51 width=1277) (actual time=1183.882..1183.918 rows=51 loops=1)
  Buffers: shared hit=133399
  ->  Sort  (cost=156148.23..156646.44 rows=199282 width=1277) (actual time=1164.026..1164.055 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 207kB
        Buffers: shared hit=133399
        ->  Result  (cost=144.04..149499.76 rows=199282 width=1277) (actual time=156.128..894.612 rows=400000 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=133393
              ->  Hash Left Join  (cost=144.04..148005.14 rows=199282 width=1250) (actual time=156.093..723.922 rows=400000 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=133393
                    ->  Nested Loop  (cost=0.00..147337.88 rows=199282 width=1242) (actual time=155.705..605.159 rows=400000 loops=1)
                          Buffers: shared hit=133339
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.029..0.082 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Seq Scan on records r  (cost=0.00..145334.06 rows=199282 width=1235) (actual time=155.665..531.852 rows=400000 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 400000
                                Buffers: shared hit=133334
                    ->  Hash  (cost=143.38..143.38 rows=52 width=40) (actual time=0.359..0.364 rows=52 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.70..143.38 rows=52 width=40) (actual time=0.082..0.345 rows=52 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.70..143.38 rows=52 width=40) (actual time=0.064..0.319 rows=52 loops=1)
                                      Recheck Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=52 width=0) (actual time=0.032..0.032 rows=52 loops=1)
                                            Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=454
Planning Time: 2.289 ms
JIT:
  Functions: 22
  Options: Inlining false, Optimization false, Expressions true, Deforming true
  Timing: Generation 1.511 ms, Inlining 0.000 ms, Optimization 0.940 ms, Emission 18.972 ms, Total 21.422 ms
Execution Time: 1212.468 ms
```
count:
```
Finalize Aggregate  (cost=14759.39..14759.40 rows=1 width=8) (actual time=123.091..134.488 rows=1 loops=1)
  Buffers: shared hit=3321
  ->  Gather  (cost=14759.17..14759.38 rows=2 width=8) (actual time=122.906..134.475 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=3321
        ->  Partial Aggregate  (cost=13759.17..13759.18 rows=1 width=8) (actual time=116.239..116.241 rows=1 loops=3)
              Buffers: shared hit=3321
              ->  Result  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.172..102.034 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=3321
                    ->  Parallel Index Only Scan using records_number_key on records r  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.137..80.176 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=3321
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=210
Planning Time: 1.459 ms
Execution Time: 134.607 ms
```
## admin sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND (r.created_at, r.id) < ('2026-09-30 05:32:59.909441+00'::timestamptz, 'd2b91ace-2bd6-55a1-4b63-8e553d8345e4'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=160108.62..160108.74 rows=51 width=1277) (actual time=1654.696..1654.780 rows=51 loops=1)
  Buffers: shared hit=133399
  ->  Sort  (cost=160108.62..160604.97 rows=198541 width=1277) (actual time=1633.673..1633.749 rows=51 loops=1)
        Sort Key: r.created_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 207kB
        Buffers: shared hit=133399
        ->  Result  (cost=144.04..153484.87 rows=198541 width=1277) (actual time=198.438..1227.779 rows=399049 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=133393
              ->  Hash Left Join  (cost=144.04..151995.81 rows=198541 width=1250) (actual time=198.389..976.976 rows=399049 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=133393
                    ->  Nested Loop  (cost=0.00..151330.49 rows=198541 width=1242) (actual time=197.367..805.742 rows=399049 loops=1)
                          Buffers: shared hit=133339
                          ->  Seq Scan on entity_definitions d  (cost=0.00..11.00 rows=1 width=39) (actual time=0.039..0.098 rows=1 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                Rows Removed by Filter: 399
                                Buffers: shared hit=5
                          ->  Seq Scan on records r  (cost=0.00..149334.08 rows=198541 width=1235) (actual time=197.315..694.933 rows=399049 loops=1)
                                Filter: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(created_at, id) < ROW('2026-09-30 05:32:59.909441+00'::timestamp with time zone, 'd2b91ace-2bd6-55a1-4b63-8e553d8345e4'::uuid)))
                                Rows Removed by Filter: 400951
                                Buffers: shared hit=133334
                    ->  Hash  (cost=143.38..143.38 rows=52 width=40) (actual time=0.954..0.967 rows=52 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.70..143.38 rows=52 width=40) (actual time=0.158..0.922 rows=52 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.70..143.38 rows=52 width=40) (actual time=0.130..0.872 rows=52 loops=1)
                                      Recheck Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=52 width=0) (actual time=0.050..0.052 rows=52 loops=1)
                                            Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=443
Planning Time: 2.057 ms
JIT:
  Functions: 22
  Options: Inlining false, Optimization false, Expressions true, Deforming true
  Timing: Generation 1.827 ms, Inlining 0.000 ms, Optimization 1.161 ms, Emission 19.959 ms, Total 22.948 ms
Execution Time: 1680.084 ms
```
count:
```
Finalize Aggregate  (cost=14759.39..14759.40 rows=1 width=8) (actual time=174.181..190.140 rows=1 loops=1)
  Buffers: shared hit=3321
  ->  Gather  (cost=14759.17..14759.38 rows=2 width=8) (actual time=173.852..190.075 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=3321
        ->  Partial Aggregate  (cost=13759.17..13759.18 rows=1 width=8) (actual time=167.531..167.535 rows=1 loops=3)
              Buffers: shared hit=3321
              ->  Result  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.460..147.686 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=3321
                    ->  Parallel Index Only Scan using records_number_key on records r  (cost=0.42..13551.59 rows=83034 width=0) (actual time=0.401..118.237 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=3321
Settings: effective_cache_size = '6GB'
Planning:
  Buffers: shared hit=210
Planning Time: 1.133 ms
Execution Time: 190.244 ms
```
