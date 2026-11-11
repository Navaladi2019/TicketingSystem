## agent-state-new page=1
count:
```
Aggregate  (cost=20626.64..20626.65 rows=1 width=8) (actual time=115.493..115.503 rows=1 loops=1)
  Buffers: shared hit=18568
  ->  Limit  (cost=457.70..20623.04 rows=288 width=4) (actual time=22.854..115.207 rows=1001 loops=1)
        Buffers: shared hit=18568
        ->  Result  (cost=457.70..20623.04 rows=288 width=4) (actual time=22.851..114.978 rows=1001 loops=1)
              Buffers: shared hit=18568
              ->  Append  (cost=457.70..20620.16 rows=288 width=0) (actual time=22.850..114.693 rows=1001 loops=1)
                    Buffers: shared hit=18568
                    ->  Subquery Scan on "*SELECT* 1"  (cost=457.70..19864.10 rows=278 width=0) (actual time=22.848..114.443 rows=1001 loops=1)
                          Buffers: shared hit=18568
                          ->  Limit  (cost=457.70..19861.32 rows=278 width=4) (actual time=22.847..114.169 rows=1001 loops=1)
                                Buffers: shared hit=18568
                                ->  Result  (cost=457.70..19861.32 rows=278 width=4) (actual time=22.846..113.965 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=18568
                                      ->  Bitmap Heap Scan on records r  (cost=457.70..19861.32 rows=278 width=0) (actual time=22.828..113.618 rows=1001 loops=1)
                                            Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Filter: (state = ANY ('{new}'::text[]))
                                            Rows Removed by Filter: 20147
                                            Heap Blocks: exact=18193
                                            Buffers: shared hit=18568
                                            ->  Bitmap Index Scan on records_list_group  (cost=0.00..457.62 rows=5829 width=0) (actual time=17.021..17.022 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=375
                    ->  Subquery Scan on "*SELECT* 2"  (cost=15.01..653.86 rows=8 width=0) (never executed)
                          ->  Limit  (cost=15.01..653.78 rows=8 width=4) (never executed)
                                ->  Result  (cost=15.01..653.78 rows=8 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Bitmap Heap Scan on records r_1  (cost=15.01..653.78 rows=8 width=0) (never executed)
                                            Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..56.27 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..56.26 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..56.26 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.26 rows=1 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.48 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=550
Planning Time: 12.811 ms
Execution Time: 116.164 ms
```
page:
```
Nested Loop Left Join  (cost=20570.02..20634.81 rows=1 width=1285) (actual time=142.924..143.303 rows=51 loops=1)
  Buffers: shared hit=21992
  ->  Nested Loop  (cost=20569.73..20626.48 rows=1 width=1250) (actual time=142.837..143.047 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=21839
        ->  Subquery Scan on x  (cost=20569.44..20618.17 rows=1 width=1243) (actual time=142.743..142.841 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=21686
              ->  Limit  (cost=20569.44..20617.02 rows=51 width=1243) (actual time=142.739..142.798 rows=51 loops=1)
                    Buffers: shared hit=21686
                    ->  Merge Append  (cost=20569.44..20626.35 rows=61 width=1243) (actual time=142.737..142.790 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=21686
                          ->  Limit  (cost=19870.59..19870.72 rows=51 width=1243) (actual time=139.588..139.613 rows=50 loops=1)
                                Buffers: shared hit=21119
                                ->  Sort  (cost=19870.59..19871.29 rows=278 width=1243) (actual time=139.587..139.606 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=21119
                                      ->  Result  (cost=457.70..19861.32 rows=278 width=1243) (actual time=20.086..137.780 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=21119
                                            ->  Bitmap Heap Scan on records r  (cost=457.70..19861.32 rows=278 width=1235) (actual time=20.077..137.212 rows=1168 loops=1)
                                                  Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Filter: (state = ANY ('{new}'::text[]))
                                                  Rows Removed by Filter: 22950
                                                  Heap Blocks: exact=20744
                                                  Buffers: shared hit=21119
                                                  ->  Bitmap Index Scan on records_list_group  (cost=0.00..457.62 rows=5829 width=0) (actual time=14.600..14.600 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=375
                          ->  Limit  (cost=653.90..653.92 rows=8 width=1243) (actual time=2.683..2.685 rows=2 loops=1)
                                Buffers: shared hit=522
                                ->  Sort  (cost=653.90..653.92 rows=8 width=1243) (actual time=2.680..2.682 rows=2 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 87kB
                                      Buffers: shared hit=522
                                      ->  Result  (cost=15.01..653.78 rows=8 width=1243) (actual time=0.768..2.376 rows=31 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=522
                                            ->  Bitmap Heap Scan on records r_1  (cost=15.01..653.78 rows=8 width=1235) (actual time=0.755..2.318 rows=31 loops=1)
                                                  Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 482
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=522
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (actual time=0.576..0.577 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=11
                          ->  Limit  (cost=0.42..56.26 rows=1 width=1243) (actual time=0.120..0.121 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..56.26 rows=1 width=1243) (actual time=0.119..0.119 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.26 rows=1 width=1235) (actual time=0.108..0.109 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.338..0.342 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.338..0.340 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1243) (actual time=0.293..0.296 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.288..0.291 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.194..0.197 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.138..0.165 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.111..0.112 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.046..0.046 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.064..0.064 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.029..0.029 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=668
Planning Time: 4.296 ms
Execution Time: 143.966 ms
```
## agent page=1
count:
```
Aggregate  (cost=757.34..757.35 rows=1 width=8) (actual time=1.895..1.903 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..744.83 rows=1001 width=4) (actual time=0.155..1.769 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..872.71 rows=1173 width=4) (actual time=0.153..1.631 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..860.98 rows=1173 width=0) (actual time=0.152..1.460 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..99.06 rows=1001 width=0) (actual time=0.151..1.313 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..89.05 rows=1001 width=4) (actual time=0.150..1.167 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..515.91 rows=5829 width=4) (actual time=0.149..1.049 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..515.91 rows=5829 width=0) (actual time=0.140..0.865 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=15.05..655.20 rows=158 width=0) (never executed)
                          ->  Limit  (cost=15.05..653.62 rows=158 width=4) (never executed)
                                ->  Result  (cost=15.05..653.62 rows=158 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Bitmap Heap Scan on records r_1  (cost=15.05..653.62 rows=158 width=0) (never executed)
                                            Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..56.38 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..56.25 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..56.25 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.25 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=496
Planning Time: 2.365 ms
Execution Time: 2.240 ms
```
page:
```
Nested Loop Left Join  (cost=46.50..1995.69 rows=1 width=1285) (actual time=1.377..8.668 rows=51 loops=1)
  Buffers: shared hit=1140
  ->  Nested Loop  (cost=46.21..1987.36 rows=1 width=1250) (actual time=1.306..8.369 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=987
        ->  Subquery Scan on x  (cost=45.92..1979.04 rows=1 width=1243) (actual time=1.272..8.175 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=834
              ->  Limit  (cost=45.92..1977.90 rows=51 width=1243) (actual time=1.269..8.107 rows=51 loops=1)
                    Buffers: shared hit=834
                    ->  Merge Append  (cost=45.92..4440.22 rows=116 width=1243) (actual time=1.267..8.096 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=834
                          ->  Limit  (cost=0.42..4123.87 rows=51 width=1243) (actual time=0.705..7.493 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..471285.36 rows=5829 width=1243) (actual time=0.704..7.482 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..471285.36 rows=5829 width=1235) (actual time=0.695..7.449 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.55..213.84 rows=51 width=1243) (actual time=0.093..0.108 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..661.32 rows=158 width=1243) (actual time=0.092..0.106 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..661.32 rows=158 width=1235) (actual time=0.086..0.098 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.42..56.25 rows=13 width=1243) (actual time=0.102..0.103 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..56.25 rows=13 width=1243) (actual time=0.101..0.102 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.25 rows=13 width=1235) (actual time=0.067..0.067 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1243) (actual time=0.365..0.370 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1243) (actual time=0.364..0.368 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1243) (actual time=0.230..0.345 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.223..0.336 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.139..0.143 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.111..0.131 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.090..0.091 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.039..0.039 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.050..0.050 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.062..0.062 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=617
Planning Time: 3.226 ms
Execution Time: 9.164 ms
```
## agent page=20
page:
```
Nested Loop Left Join  (cost=46.51..2044.06 rows=1 width=1285) (actual time=1.187..7.661 rows=51 loops=1)
  Buffers: shared hit=1125
  ->  Nested Loop  (cost=46.21..2035.73 rows=1 width=1250) (actual time=1.118..7.350 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=972
        ->  Subquery Scan on x  (cost=45.93..2027.42 rows=1 width=1243) (actual time=1.041..7.111 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=819
              ->  Limit  (cost=45.93..2026.27 rows=51 width=1243) (actual time=1.039..7.043 rows=51 loops=1)
                    Buffers: shared hit=819
                    ->  Merge Append  (cost=45.93..4511.40 rows=115 width=1243) (actual time=1.037..7.031 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=819
                          ->  Limit  (cost=0.42..4199.81 rows=51 width=1243) (actual time=0.516..6.466 rows=49 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..460450.37 rows=5592 width=1243) (actual time=0.516..6.457 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..460450.37 rows=5592 width=1235) (actual time=0.510..6.431 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 721
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..213.04 rows=51 width=1243) (actual time=0.099..0.108 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..633.85 rows=152 width=1243) (actual time=0.099..0.107 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..633.85 rows=152 width=1235) (actual time=0.095..0.103 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.42..52.30 rows=12 width=1243) (actual time=0.052..0.064 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..52.30 rows=12 width=1243) (actual time=0.052..0.064 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.30 rows=12 width=1235) (actual time=0.049..0.060 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.365..0.369 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.365..0.369 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1243) (actual time=0.238..0.347 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.235..0.343 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.152..0.156 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.122..0.147 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.096..0.098 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.039..0.039 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.056..0.057 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.060..0.061 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=606
Planning Time: 3.642 ms
Execution Time: 8.054 ms
```
## agent-group-desc page=1
count:
```
Aggregate  (cost=757.34..757.35 rows=1 width=8) (actual time=1.910..1.916 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..744.83 rows=1001 width=4) (actual time=0.186..1.789 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..872.71 rows=1173 width=4) (actual time=0.184..1.654 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..860.98 rows=1173 width=0) (actual time=0.184..1.477 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..99.06 rows=1001 width=0) (actual time=0.183..1.352 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..89.05 rows=1001 width=4) (actual time=0.182..1.210 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..515.91 rows=5829 width=4) (actual time=0.181..1.093 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..515.91 rows=5829 width=0) (actual time=0.169..0.917 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=15.05..655.20 rows=158 width=0) (never executed)
                          ->  Limit  (cost=15.05..653.62 rows=158 width=4) (never executed)
                                ->  Result  (cost=15.05..653.62 rows=158 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Bitmap Heap Scan on records r_1  (cost=15.05..653.62 rows=158 width=0) (never executed)
                                            Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..56.38 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..56.25 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..56.25 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.25 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=496
Planning Time: 6.887 ms
Execution Time: 2.925 ms
```
page:
```
Nested Loop Left Join  (cost=761.03..2603.22 rows=1 width=1293) (actual time=199.127..199.517 rows=51 loops=1)
  Buffers: shared hit=57634
  ->  Nested Loop  (cost=760.73..2594.89 rows=1 width=1258) (actual time=199.079..199.278 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=57481
        ->  Subquery Scan on x  (cost=760.45..2586.58 rows=1 width=1251) (actual time=199.040..199.129 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=57328
              ->  Limit  (cost=760.45..2585.43 rows=51 width=1251) (actual time=199.037..199.086 rows=51 loops=1)
                    Buffers: shared hit=57328
                    ->  Merge Append  (cost=760.45..4911.38 rows=116 width=1251) (actual time=199.034..199.078 rows=51 loops=1)
                          Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                          Buffers: shared hit=57328
                          ->  Limit  (cost=0.55..4149.58 rows=51 width=1251) (actual time=196.248..196.250 rows=1 loops=1)
                                Buffers: shared hit=56714
                                ->  Result  (cost=0.55..474210.32 rows=5829 width=1251) (actual time=196.246..196.247 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=56714
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..474210.32 rows=5829 width=1235) (actual time=196.235..196.236 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL))
                                            Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                            Rows Removed by Filter: 55887
                                            Buffers: shared hit=56714
                          ->  Limit  (cost=658.89..659.02 rows=51 width=1251) (actual time=2.172..2.187 rows=44 loops=1)
                                Buffers: shared hit=522
                                ->  Sort  (cost=658.89..659.28 rows=158 width=1251) (actual time=2.169..2.179 rows=44 loops=1)
                                      Sort Key: r_1.assignment_group_id DESC, r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: top-N heapsort  Memory: 189kB
                                      Buffers: shared hit=522
                                      ->  Result  (cost=15.05..653.62 rows=158 width=1251) (actual time=0.303..1.516 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=522
                                            ->  Bitmap Heap Scan on records r_1  (cost=15.05..653.62 rows=158 width=1235) (actual time=0.290..1.372 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=522
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (actual time=0.193..0.193 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=11
                          ->  Limit  (cost=56.49..56.52 rows=13 width=1251) (actual time=0.331..0.334 rows=8 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=56.49..56.52 rows=13 width=1251) (actual time=0.329..0.331 rows=8 loops=1)
                                      Sort Key: r_2.assignment_group_id DESC, r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 147kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..56.25 rows=13 width=1251) (actual time=0.041..0.188 rows=61 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.25 rows=13 width=1235) (actual time=0.032..0.163 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.275..0.278 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.275..0.277 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id DESC, r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.201..0.239 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.197..0.235 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.167..0.169 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.131..0.156 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.106..0.107 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.044..0.044 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.061..0.062 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.019..0.019 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=617
Planning Time: 7.394 ms
Execution Time: 201.273 ms
```
## agent-group-desc page=20
page:
```
Nested Loop Left Join  (cost=761.60..2748.37 rows=1 width=1293) (actual time=9.989..11.141 rows=51 loops=1)
  Buffers: shared hit=976
  ->  Nested Loop  (cost=761.30..2740.04 rows=1 width=1258) (actual time=9.877..10.848 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=823
        ->  Subquery Scan on x  (cost=761.02..2731.72 rows=1 width=1251) (actual time=9.759..10.590 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=670
              ->  Limit  (cost=761.02..2730.58 rows=51 width=1251) (actual time=9.754..10.532 rows=51 loops=1)
                    Buffers: shared hit=670
                    ->  Merge Append  (cost=761.02..5163.56 rows=114 width=1251) (actual time=9.752..10.521 rows=51 loops=1)
                          Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                          Buffers: shared hit=670
                          ->  Limit  (cost=0.55..4401.22 rows=51 width=1251) (actual time=0.163..0.900 rows=51 loops=1)
                                Buffers: shared hit=56
                                ->  Result  (cost=0.55..435235.26 rows=5044 width=1251) (actual time=0.162..0.890 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=56
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..435235.26 rows=5044 width=1235) (actual time=0.151..0.859 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                            Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                            Buffers: shared hit=56
                          ->  Limit  (cost=659.40..659.53 rows=51 width=1251) (actual time=7.622..7.627 rows=1 loops=1)
                                Buffers: shared hit=522
                                ->  Sort  (cost=659.40..659.75 rows=137 width=1251) (actual time=7.620..7.625 rows=1 loops=1)
                                      Sort Key: r_1.assignment_group_id DESC, r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: top-N heapsort  Memory: 181kB
                                      Buffers: shared hit=522
                                      ->  Result  (cost=15.04..654.83 rows=137 width=1251) (actual time=0.680..7.013 rows=412 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=522
                                            ->  Bitmap Heap Scan on records r_1  (cost=15.04..654.83 rows=137 width=1235) (actual time=0.676..6.871 rows=412 loops=1)
                                                  Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                  Rows Removed by Filter: 101
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=522
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (actual time=0.568..0.569 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=11
                          ->  Limit  (cost=56.53..56.56 rows=11 width=1251) (actual time=1.291..1.293 rows=1 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=56.53..56.56 rows=11 width=1251) (actual time=1.288..1.290 rows=1 loops=1)
                                      Sort Key: r_2.assignment_group_id DESC, r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 121kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..56.34 rows=11 width=1251) (actual time=0.095..0.907 rows=48 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.34 rows=11 width=1235) (actual time=0.081..0.872 rows=48 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                  Rows Removed by Filter: 16
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.49..44.50 rows=1 width=1251) (actual time=0.667..0.675 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.50 rows=1 width=1251) (actual time=0.663..0.671 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id DESC, r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.368..0.462 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.355..0.447 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.248..0.254 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.212..0.240 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.182..0.184 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.115..0.115 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.065..0.066 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.061..0.061 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=606
Planning Time: 6.761 ms
Execution Time: 12.262 ms
```
## agent-group-asc page=1
count:
```
Aggregate  (cost=757.34..757.35 rows=1 width=8) (actual time=1.434..1.439 rows=1 loops=1)
  Buffers: shared hit=546
  ->  Limit  (cost=0.55..744.83 rows=1001 width=4) (actual time=0.126..1.314 rows=1001 loops=1)
        Buffers: shared hit=546
        ->  Result  (cost=0.55..872.71 rows=1173 width=4) (actual time=0.125..1.212 rows=1001 loops=1)
              Buffers: shared hit=546
              ->  Append  (cost=0.55..860.98 rows=1173 width=0) (actual time=0.124..1.076 rows=1001 loops=1)
                    Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..99.06 rows=1001 width=0) (actual time=0.124..0.981 rows=1001 loops=1)
                          Buffers: shared hit=546
                          ->  Limit  (cost=0.55..89.05 rows=1001 width=4) (actual time=0.123..0.871 rows=1001 loops=1)
                                Buffers: shared hit=546
                                ->  Result  (cost=0.55..515.91 rows=5829 width=4) (actual time=0.123..0.782 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=546
                                      ->  Index Only Scan using records_list_group on records r  (cost=0.55..515.91 rows=5829 width=0) (actual time=0.115..0.648 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=15.05..655.20 rows=158 width=0) (never executed)
                          ->  Limit  (cost=15.05..653.62 rows=158 width=4) (never executed)
                                ->  Result  (cost=15.05..653.62 rows=158 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Bitmap Heap Scan on records r_1  (cost=15.05..653.62 rows=158 width=0) (never executed)
                                            Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..56.38 rows=13 width=0) (never executed)
                          ->  Limit  (cost=0.42..56.25 rows=13 width=4) (never executed)
                                ->  Result  (cost=0.42..56.25 rows=13 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.25 rows=13 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.42..44.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.42..44.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Nested Loop  (cost=36.42..44.47 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                  Group Key: acl_share.tenant_id, acl_share.record_id
                                                  ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                        Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=496
Planning Time: 2.115 ms
Execution Time: 1.651 ms
```
page:
```
Nested Loop Left Join  (cost=761.03..2603.22 rows=1 width=1293) (actual time=145.265..145.611 rows=51 loops=1)
  Buffers: shared hit=50200
  ->  Nested Loop  (cost=760.73..2594.89 rows=1 width=1258) (actual time=145.221..145.403 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=50047
        ->  Subquery Scan on x  (cost=760.45..2586.58 rows=1 width=1251) (actual time=145.187..145.278 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=49894
              ->  Limit  (cost=760.45..2585.43 rows=51 width=1251) (actual time=145.184..145.242 rows=51 loops=1)
                    Buffers: shared hit=49894
                    ->  Merge Append  (cost=760.45..4911.38 rows=116 width=1251) (actual time=145.182..145.235 rows=51 loops=1)
                          Sort Key: r.assignment_group_id, r.updated_at, r.id
                          Buffers: shared hit=49894
                          ->  Limit  (cost=0.55..4149.58 rows=51 width=1251) (actual time=141.044..141.045 rows=1 loops=1)
                                Buffers: shared hit=49280
                                ->  Result  (cost=0.55..474210.32 rows=5829 width=1251) (actual time=141.043..141.044 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=49280
                                      ->  Index Scan Backward using records_list_group on records r  (cost=0.55..474210.32 rows=5829 width=1235) (actual time=141.036..141.037 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL))
                                            Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                            Rows Removed by Filter: 47804
                                            Buffers: shared hit=49280
                          ->  Limit  (cost=658.89..659.02 rows=51 width=1251) (actual time=2.735..2.748 rows=44 loops=1)
                                Buffers: shared hit=522
                                ->  Sort  (cost=658.89..659.28 rows=158 width=1251) (actual time=2.731..2.740 rows=44 loops=1)
                                      Sort Key: r_1.assignment_group_id, r_1.updated_at, r_1.id
                                      Sort Method: top-N heapsort  Memory: 203kB
                                      Buffers: shared hit=522
                                      ->  Result  (cost=15.05..653.62 rows=158 width=1251) (actual time=0.254..1.592 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=522
                                            ->  Bitmap Heap Scan on records r_1  (cost=15.05..653.62 rows=158 width=1235) (actual time=0.242..1.456 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=522
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..15.00 rows=163 width=0) (actual time=0.161..0.161 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=11
                          ->  Limit  (cost=56.49..56.52 rows=13 width=1251) (actual time=1.081..1.083 rows=8 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=56.49..56.52 rows=13 width=1251) (actual time=1.078..1.080 rows=8 loops=1)
                                      Sort Key: r_2.assignment_group_id, r_2.updated_at, r_2.id
                                      Sort Method: quicksort  Memory: 147kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..56.25 rows=13 width=1251) (actual time=0.105..0.386 rows=61 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.25 rows=13 width=1235) (actual time=0.093..0.348 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.314..0.317 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.313..0.315 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id, r_3.updated_at, r_3.id
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.223..0.293 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.215..0.284 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.170..0.173 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.136..0.161 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.111..0.112 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.049..0.049 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.061..0.061 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.035..0.035 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=594
Planning Time: 4.059 ms
Execution Time: 146.055 ms
```
## admin-state-in page=1
count:
```
Aggregate  (cost=91.11..91.12 rows=1 width=8) (actual time=0.769..0.770 rows=1 loops=1)
  Buffers: shared hit=535
  ->  Limit  (cost=0.42..78.60 rows=1001 width=4) (actual time=0.107..0.712 rows=1001 loops=1)
        Buffers: shared hit=535
        ->  Result  (cost=0.42..1568.69 rows=20082 width=4) (actual time=0.106..0.640 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=535
              ->  Index Only Scan using records_list_state on records r  (cost=0.42..1568.69 rows=20082 width=0) (actual time=0.100..0.536 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (state = ANY ('{new,in_progress}'::text[])))
                    Heap Fetches: 0
                    Buffers: shared hit=535
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=348
Planning Time: 0.919 ms
Execution Time: 0.836 ms
```
page:
```
Limit  (cost=1.00..1218.52 rows=51 width=1277) (actual time=0.249..4.780 rows=51 loops=1)
  Buffers: shared hit=635
  ->  Result  (cost=1.00..479418.40 rows=20082 width=1277) (actual time=0.249..4.771 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=635
        ->  Nested Loop Left Join  (cost=1.00..479267.78 rows=20082 width=1250) (actual time=0.238..4.730 rows=51 loops=1)
              Buffers: shared hit=635
              ->  Nested Loop  (cost=0.70..471035.85 rows=20082 width=1242) (actual time=0.200..4.355 rows=51 loops=1)
                    Buffers: shared hit=482
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..470776.53 rows=20082 width=1235) (actual time=0.175..4.293 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (state = ANY ('{new,in_progress}'::text[]))
                          Rows Removed by Filter: 419
                          Buffers: shared hit=479
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.020..0.021 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.40 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.40 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=580
Planning Time: 1.436 ms
Execution Time: 4.885 ms
```
