## agent-state-new page=1
count:
```
Aggregate  (cost=561.75..561.76 rows=1 width=8) (actual time=1.060..1.064 rows=1 loops=1)
  Buffers: shared hit=534
  ->  Limit  (cost=0.55..558.00 rows=300 width=4) (actual time=0.148..0.989 rows=1001 loops=1)
        Buffers: shared hit=534
        ->  Result  (cost=0.55..558.00 rows=300 width=4) (actual time=0.147..0.910 rows=1001 loops=1)
              Buffers: shared hit=534
              ->  Append  (cost=0.55..555.00 rows=300 width=0) (actual time=0.147..0.806 rows=1001 loops=1)
                    Buffers: shared hit=534
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..29.17 rows=293 width=0) (actual time=0.146..0.732 rows=1001 loops=1)
                          Buffers: shared hit=534
                          ->  Limit  (cost=0.55..26.24 rows=293 width=4) (actual time=0.145..0.648 rows=1001 loops=1)
                                Buffers: shared hit=534
                                ->  Result  (cost=0.55..26.24 rows=293 width=4) (actual time=0.145..0.579 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=534
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..26.24 rows=293 width=0) (actual time=0.137..0.474 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=534
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..427.47 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.55..427.42 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.55..427.42 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.42 rows=5 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..52.38 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.42..52.37 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.42..52.37 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.37 rows=1 width=0) (never executed)
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
  Buffers: shared hit=554
Planning Time: 3.158 ms
Execution Time: 1.224 ms
```
page:
```
Nested Loop Left Join  (cost=1216.27..1655.39 rows=1 width=1285) (actual time=10.414..10.898 rows=51 loops=1)
  Buffers: shared hit=1584
  ->  Nested Loop  (cost=1215.97..1647.05 rows=1 width=1250) (actual time=10.360..10.732 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1431
        ->  Subquery Scan on x  (cost=1215.69..1638.74 rows=1 width=1243) (actual time=10.330..10.630 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1278
              ->  Limit  (cost=1215.69..1637.59 rows=51 width=1243) (actual time=10.326..10.600 rows=51 loops=1)
                    Buffers: shared hit=1278
                    ->  Merge Append  (cost=1215.69..1695.50 rows=58 width=1243) (actual time=10.325..10.595 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1278
                          ->  Limit  (cost=1170.19..1170.32 rows=51 width=1243) (actual time=9.597..9.615 rows=50 loops=1)
                                Buffers: shared hit=1186
                                ->  Sort  (cost=1170.19..1170.92 rows=293 width=1243) (actual time=9.595..9.608 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=1186
                                      ->  Result  (cost=23.40..1160.42 rows=293 width=1243) (actual time=0.594..8.827 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=1186
                                            ->  Bitmap Heap Scan on records r  (cost=23.40..1160.42 rows=293 width=1235) (actual time=0.588..8.590 rows=1168 loops=1)
                                                  Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                  Heap Blocks: exact=1156
                                                  Buffers: shared hit=1186
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..23.31 rows=293 width=0) (actual time=0.442..0.442 rows=1168 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                        Buffers: shared hit=30
                          ->  Limit  (cost=0.55..427.42 rows=5 width=1243) (actual time=0.216..0.446 rows=2 loops=1)
                                Buffers: shared hit=47
                                ->  Result  (cost=0.55..427.42 rows=5 width=1243) (actual time=0.212..0.441 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=47
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.42 rows=5 width=1235) (actual time=0.194..0.422 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=47
                          ->  Limit  (cost=0.42..52.37 rows=1 width=1243) (actual time=0.246..0.247 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..52.37 rows=1 width=1243) (actual time=0.245..0.245 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.37 rows=1 width=1235) (actual time=0.240..0.241 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.259..0.262 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.258..0.261 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1243) (actual time=0.240..0.244 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.233..0.236 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.115..0.118 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.086..0.107 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.064..0.065 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.029..0.029 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.038..0.038 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=672
Planning Time: 3.859 ms
Execution Time: 11.336 ms
```
## agent page=1
count:
```
Aggregate  (cost=577.73..577.74 rows=1 width=8) (actual time=1.589..1.594 rows=1 loops=1)
  Buffers: shared hit=521
  ->  Limit  (cost=0.55..565.22 rows=1001 width=4) (actual time=0.141..1.471 rows=1001 loops=1)
        Buffers: shared hit=521
        ->  Result  (cost=0.55..629.52 rows=1115 width=4) (actual time=0.140..1.341 rows=1001 loops=1)
              Buffers: shared hit=521
              ->  Append  (cost=0.55..618.37 rows=1115 width=0) (actual time=0.140..1.166 rows=1001 loops=1)
                    Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..87.55 rows=1001 width=0) (actual time=0.139..1.043 rows=1001 loops=1)
                          Buffers: shared hit=521
                          ->  Limit  (cost=0.55..77.54 rows=1001 width=4) (actual time=0.138..0.878 rows=1001 loops=1)
                                Buffers: shared hit=521
                                ->  Result  (cost=0.55..453.40 rows=5888 width=4) (actual time=0.138..0.764 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=521
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..453.40 rows=5888 width=0) (actual time=0.129..0.594 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..428.30 rows=101 width=0) (never executed)
                          ->  Limit  (cost=0.55..427.29 rows=101 width=4) (never executed)
                                ->  Result  (cost=0.55..427.29 rows=101 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..52.47 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..52.35 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..52.35 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=0) (never executed)
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
  Buffers: shared hit=500
Planning Time: 2.382 ms
Execution Time: 1.849 ms
```
page:
```
Nested Loop Left Join  (cost=46.50..1957.22 rows=1 width=1285) (actual time=1.293..8.021 rows=51 loops=1)
  Buffers: shared hit=1140
  ->  Nested Loop  (cost=46.21..1948.88 rows=1 width=1250) (actual time=1.230..7.752 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=987
        ->  Subquery Scan on x  (cost=45.92..1940.57 rows=1 width=1243) (actual time=1.198..7.576 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=834
              ->  Limit  (cost=45.92..1939.42 rows=51 width=1243) (actual time=1.194..7.513 rows=51 loops=1)
                    Buffers: shared hit=834
                    ->  Merge Append  (cost=45.92..4315.58 rows=115 width=1243) (actual time=1.193..7.502 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=834
                          ->  Limit  (cost=0.42..4000.94 rows=51 width=1243) (actual time=0.710..6.981 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..461863.95 rows=5888 width=1243) (actual time=0.709..6.971 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..461863.95 rows=5888 width=1235) (actual time=0.703..6.944 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.55..216.03 rows=51 width=1243) (actual time=0.084..0.097 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..427.29 rows=101 width=1243) (actual time=0.083..0.096 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=1235) (actual time=0.078..0.088 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.42..52.35 rows=12 width=1243) (actual time=0.047..0.048 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..52.35 rows=12 width=1243) (actual time=0.046..0.047 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=1235) (actual time=0.044..0.044 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1243) (actual time=0.348..0.354 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1243) (actual time=0.348..0.353 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1243) (actual time=0.220..0.332 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.218..0.328 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.121..0.125 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.096..0.113 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.075..0.076 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.034..0.034 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.040..0.040 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.065..0.065 rows=1 loops=3)
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
  Buffers: shared hit=621
Planning Time: 4.067 ms
Execution Time: 8.449 ms
```
## agent page=20
page:
```
Nested Loop Left Join  (cost=46.51..2004.58 rows=1 width=1285) (actual time=1.023..7.448 rows=51 loops=1)
  Buffers: shared hit=1125
  ->  Nested Loop  (cost=46.21..1996.24 rows=1 width=1250) (actual time=0.966..7.209 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=972
        ->  Subquery Scan on x  (cost=45.93..1987.93 rows=1 width=1243) (actual time=0.937..7.062 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=819
              ->  Limit  (cost=45.93..1986.78 rows=51 width=1243) (actual time=0.934..7.012 rows=51 loops=1)
                    Buffers: shared hit=819
                    ->  Merge Append  (cost=45.93..4384.31 rows=114 width=1243) (actual time=0.933..7.002 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=819
                          ->  Limit  (cost=0.42..4071.13 rows=51 width=1243) (actual time=0.502..6.521 rows=49 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..451370.24 rows=5655 width=1243) (actual time=0.502..6.513 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..451370.24 rows=5655 width=1235) (actual time=0.496..6.491 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 721
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..214.56 rows=51 width=1243) (actual time=0.086..0.098 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..407.60 rows=97 width=1243) (actual time=0.085..0.096 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..407.60 rows=97 width=1235) (actual time=0.081..0.091 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.42..52.37 rows=11 width=1243) (actual time=0.043..0.056 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..52.37 rows=11 width=1243) (actual time=0.043..0.055 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.37 rows=11 width=1235) (actual time=0.041..0.051 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.298..0.307 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.298..0.306 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1243) (actual time=0.178..0.288 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.176..0.284 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.113..0.118 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.089..0.110 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.068..0.070 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.028..0.029 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.039..0.039 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.053..0.053 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=610
Planning Time: 2.583 ms
Execution Time: 7.865 ms
```
## agent-group-desc page=1
count:
```
Aggregate  (cost=577.73..577.74 rows=1 width=8) (actual time=1.092..1.095 rows=1 loops=1)
  Buffers: shared hit=521
  ->  Limit  (cost=0.55..565.22 rows=1001 width=4) (actual time=0.127..1.012 rows=1001 loops=1)
        Buffers: shared hit=521
        ->  Result  (cost=0.55..629.52 rows=1115 width=4) (actual time=0.126..0.926 rows=1001 loops=1)
              Buffers: shared hit=521
              ->  Append  (cost=0.55..618.37 rows=1115 width=0) (actual time=0.126..0.809 rows=1001 loops=1)
                    Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..87.55 rows=1001 width=0) (actual time=0.125..0.727 rows=1001 loops=1)
                          Buffers: shared hit=521
                          ->  Limit  (cost=0.55..77.54 rows=1001 width=4) (actual time=0.124..0.608 rows=1001 loops=1)
                                Buffers: shared hit=521
                                ->  Result  (cost=0.55..453.40 rows=5888 width=4) (actual time=0.124..0.531 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=521
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..453.40 rows=5888 width=0) (actual time=0.116..0.416 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..428.30 rows=101 width=0) (never executed)
                          ->  Limit  (cost=0.55..427.29 rows=101 width=4) (never executed)
                                ->  Result  (cost=0.55..427.29 rows=101 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..52.47 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..52.35 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..52.35 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=0) (never executed)
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
  Buffers: shared hit=500
Planning Time: 2.009 ms
Execution Time: 1.291 ms
```
page:
```
Nested Loop Left Join  (cost=625.20..2421.29 rows=1 width=1293) (actual time=398.592..398.891 rows=51 loops=1)
  Buffers: shared hit=187055
  ->  Nested Loop  (cost=624.91..2412.95 rows=1 width=1258) (actual time=398.549..398.700 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=186902
        ->  Subquery Scan on x  (cost=624.62..2404.64 rows=1 width=1251) (actual time=398.516..398.583 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=186749
              ->  Limit  (cost=624.62..2403.49 rows=51 width=1251) (actual time=398.512..398.550 rows=51 loops=1)
                    Buffers: shared hit=186749
                    ->  Merge Append  (cost=624.62..4635.80 rows=115 width=1251) (actual time=398.511..398.544 rows=51 loops=1)
                          Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                          Buffers: shared hit=186749
                          ->  Limit  (cost=96.89..4106.17 rows=51 width=1251) (actual time=396.275..396.277 rows=1 loops=1)
                                Buffers: shared hit=186133
                                ->  Incremental Sort  (cost=96.89..462973.15 rows=5888 width=1251) (actual time=396.274..396.276 rows=1 loops=1)
                                      Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                                      Presorted Key: r.assignment_group_id
                                      Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 155kB  Peak Memory: 155kB
                                      Pre-sorted Groups: 1  Sort Method: top-N heapsort  Average Memory: 127kB  Peak Memory: 127kB
                                      Buffers: shared hit=186133
                                      ->  Result  (cost=0.55..462746.13 rows=5888 width=1251) (actual time=174.606..390.676 rows=8139 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=186133
                                            ->  Index Scan using records_list_group_state on records r  (cost=0.55..462746.13 rows=5888 width=1235) (actual time=174.599..388.807 rows=8139 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL))
                                                  Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                                  Rows Removed by Filter: 175656
                                                  Buffers: shared hit=186133
                          ->  Limit  (cost=430.65..430.78 rows=51 width=1251) (actual time=1.717..1.727 rows=44 loops=1)
                                Buffers: shared hit=524
                                ->  Sort  (cost=430.65..430.90 rows=101 width=1251) (actual time=1.714..1.721 rows=44 loops=1)
                                      Sort Key: r_1.assignment_group_id DESC, r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: top-N heapsort  Memory: 195kB
                                      Buffers: shared hit=524
                                      ->  Result  (cost=0.55..427.29 rows=101 width=1251) (actual time=0.151..1.217 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=524
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=1235) (actual time=0.139..1.102 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=524
                          ->  Limit  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.306..0.308 rows=8 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.304..0.306 rows=8 loops=1)
                                      Sort Key: r_2.assignment_group_id DESC, r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 147kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..52.35 rows=12 width=1251) (actual time=0.038..0.167 rows=61 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=1235) (actual time=0.027..0.143 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.205..0.207 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.204..0.206 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id DESC, r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.153..0.191 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.149..0.186 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.125..0.127 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.103..0.116 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.062..0.062 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.028..0.028 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.032..0.032 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.018..0.018 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=621
Planning Time: 3.072 ms
Execution Time: 399.230 ms
```
## agent-group-desc page=20
page:
```
Nested Loop Left Join  (cost=628.55..2562.79 rows=1 width=1293) (actual time=352.273..352.548 rows=51 loops=1)
  Buffers: shared hit=129592
  ->  Nested Loop  (cost=628.25..2554.46 rows=1 width=1258) (actual time=352.212..352.371 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=129439
        ->  Subquery Scan on x  (cost=627.97..2546.14 rows=1 width=1251) (actual time=352.151..352.225 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=129286
              ->  Limit  (cost=627.97..2545.00 rows=51 width=1251) (actual time=352.148..352.192 rows=51 loops=1)
                    Buffers: shared hit=129286
                    ->  Merge Append  (cost=627.97..4875.50 rows=113 width=1251) (actual time=352.147..352.186 rows=51 loops=1)
                          Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                          Buffers: shared hit=129286
                          ->  Limit  (cost=99.92..4345.59 rows=51 width=1251) (actual time=350.249..350.264 rows=51 loops=1)
                                Buffers: shared hit=128670
                                ->  Incremental Sort  (cost=99.92..424750.50 rows=5101 width=1251) (actual time=350.249..350.259 rows=51 loops=1)
                                      Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                                      Presorted Key: r.assignment_group_id
                                      Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 155kB  Peak Memory: 155kB
                                      Pre-sorted Groups: 1  Sort Method: top-N heapsort  Average Memory: 127kB  Peak Memory: 127kB
                                      Buffers: shared hit=128670
                                      ->  Result  (cost=0.55..424549.93 rows=5101 width=1251) (actual time=0.164..343.778 rows=7273 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=128670
                                            ->  Index Scan using records_list_group_state on records r  (cost=0.55..424549.93 rows=5101 width=1235) (actual time=0.160..341.636 rows=7273 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                  Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                                  Rows Removed by Filter: 119769
                                                  Buffers: shared hit=128670
                          ->  Limit  (cost=430.92..431.04 rows=51 width=1251) (actual time=1.458..1.459 rows=1 loops=1)
                                Buffers: shared hit=524
                                ->  Sort  (cost=430.92..431.14 rows=88 width=1251) (actual time=1.456..1.457 rows=1 loops=1)
                                      Sort Key: r_1.assignment_group_id DESC, r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=524
                                      ->  Result  (cost=0.55..428.07 rows=88 width=1251) (actual time=0.054..0.996 rows=412 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=524
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..428.07 rows=88 width=1235) (actual time=0.043..0.897 rows=412 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                  Rows Removed by Filter: 101
                                                  Buffers: shared hit=524
                          ->  Limit  (cost=52.61..52.63 rows=10 width=1251) (actual time=0.276..0.277 rows=1 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=52.61..52.63 rows=10 width=1251) (actual time=0.274..0.275 rows=1 loops=1)
                                      Sort Key: r_2.assignment_group_id DESC, r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 121kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..52.44 rows=10 width=1251) (actual time=0.030..0.177 rows=48 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.44 rows=10 width=1235) (actual time=0.023..0.159 rows=48 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                  Rows Removed by Filter: 16
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.49..44.50 rows=1 width=1251) (actual time=0.155..0.158 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.50 rows=1 width=1251) (actual time=0.155..0.157 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id DESC, r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.111..0.144 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.107..0.138 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.088..0.090 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.070..0.081 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.053..0.054 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.024..0.024 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.028..0.028 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=610
Planning Time: 3.753 ms
Execution Time: 352.945 ms
```
## agent-group-asc page=1
count:
```
Aggregate  (cost=577.73..577.74 rows=1 width=8) (actual time=1.218..1.223 rows=1 loops=1)
  Buffers: shared hit=521
  ->  Limit  (cost=0.55..565.22 rows=1001 width=4) (actual time=0.146..1.132 rows=1001 loops=1)
        Buffers: shared hit=521
        ->  Result  (cost=0.55..629.52 rows=1115 width=4) (actual time=0.145..1.039 rows=1001 loops=1)
              Buffers: shared hit=521
              ->  Append  (cost=0.55..618.37 rows=1115 width=0) (actual time=0.145..0.915 rows=1001 loops=1)
                    Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..87.55 rows=1001 width=0) (actual time=0.144..0.826 rows=1001 loops=1)
                          Buffers: shared hit=521
                          ->  Limit  (cost=0.55..77.54 rows=1001 width=4) (actual time=0.143..0.723 rows=1001 loops=1)
                                Buffers: shared hit=521
                                ->  Result  (cost=0.55..453.40 rows=5888 width=4) (actual time=0.143..0.642 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=521
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..453.40 rows=5888 width=0) (actual time=0.131..0.516 rows=1001 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.55..428.30 rows=101 width=0) (never executed)
                          ->  Limit  (cost=0.55..427.29 rows=101 width=4) (never executed)
                                ->  Result  (cost=0.55..427.29 rows=101 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=0) (never executed)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.42..52.47 rows=12 width=0) (never executed)
                          ->  Limit  (cost=0.42..52.35 rows=12 width=4) (never executed)
                                ->  Result  (cost=0.42..52.35 rows=12 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=0) (never executed)
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
  Buffers: shared hit=500
Planning Time: 2.316 ms
Execution Time: 1.525 ms
```
page:
```
Nested Loop Left Join  (cost=625.20..2421.29 rows=1 width=1293) (actual time=409.823..410.138 rows=51 loops=1)
  Buffers: shared hit=214441
  ->  Nested Loop  (cost=624.91..2412.95 rows=1 width=1258) (actual time=409.777..409.929 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=214288
        ->  Subquery Scan on x  (cost=624.62..2404.64 rows=1 width=1251) (actual time=409.745..409.815 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=214135
              ->  Limit  (cost=624.62..2403.49 rows=51 width=1251) (actual time=409.739..409.780 rows=51 loops=1)
                    Buffers: shared hit=214135
                    ->  Merge Append  (cost=624.62..4635.80 rows=115 width=1251) (actual time=409.737..409.773 rows=51 loops=1)
                          Sort Key: r.assignment_group_id, r.updated_at, r.id
                          Buffers: shared hit=214135
                          ->  Limit  (cost=96.89..4106.17 rows=51 width=1251) (actual time=407.747..407.749 rows=1 loops=1)
                                Buffers: shared hit=213519
                                ->  Incremental Sort  (cost=96.89..462973.15 rows=5888 width=1251) (actual time=407.745..407.747 rows=1 loops=1)
                                      Sort Key: r.assignment_group_id, r.updated_at, r.id
                                      Presorted Key: r.assignment_group_id
                                      Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 155kB  Peak Memory: 155kB
                                      Pre-sorted Groups: 1  Sort Method: top-N heapsort  Average Memory: 127kB  Peak Memory: 127kB
                                      Buffers: shared hit=213519
                                      ->  Result  (cost=0.55..462746.13 rows=5888 width=1251) (actual time=145.243..403.094 rows=7996 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=213519
                                            ->  Index Scan Backward using records_list_group_state on records r  (cost=0.55..462746.13 rows=5888 width=1235) (actual time=145.222..401.550 rows=7996 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL))
                                                  Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                                  Rows Removed by Filter: 200226
                                                  Buffers: shared hit=213519
                          ->  Limit  (cost=430.65..430.78 rows=51 width=1251) (actual time=1.507..1.519 rows=44 loops=1)
                                Buffers: shared hit=524
                                ->  Sort  (cost=430.65..430.90 rows=101 width=1251) (actual time=1.504..1.513 rows=44 loops=1)
                                      Sort Key: r_1.assignment_group_id, r_1.updated_at, r_1.id
                                      Sort Method: top-N heapsort  Memory: 191kB
                                      Buffers: shared hit=524
                                      ->  Result  (cost=0.55..427.29 rows=101 width=1251) (actual time=0.056..1.038 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=524
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=1235) (actual time=0.044..0.931 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=524
                          ->  Limit  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.302..0.306 rows=8 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.300..0.303 rows=8 loops=1)
                                      Sort Key: r_2.assignment_group_id, r_2.updated_at, r_2.id
                                      Sort Method: quicksort  Memory: 147kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..52.35 rows=12 width=1251) (actual time=0.026..0.175 rows=61 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=1235) (actual time=0.019..0.156 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.172..0.174 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.171..0.173 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id, r_3.updated_at, r_3.id
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.123..0.157 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.118..0.151 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.096..0.099 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.077..0.089 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.059..0.060 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.028..0.028 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.030..0.030 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.015..0.015 rows=1 loops=3)
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
  Buffers: shared hit=598
Planning Time: 3.253 ms
Execution Time: 410.825 ms
```
## admin-state-in page=1
count:
```
Aggregate  (cost=91.23..91.24 rows=1 width=8) (actual time=1.119..1.121 rows=1 loops=1)
  Buffers: shared hit=535
  ->  Limit  (cost=0.42..78.71 rows=1001 width=4) (actual time=0.216..1.055 rows=1001 loops=1)
        Buffers: shared hit=535
        ->  Result  (cost=0.42..1534.04 rows=19609 width=4) (actual time=0.210..0.972 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=535
              ->  Index Only Scan using records_list_state on records r  (cost=0.42..1534.04 rows=19609 width=0) (actual time=0.193..0.847 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (state = ANY ('{new,in_progress}'::text[])))
                    Heap Fetches: 0
                    Buffers: shared hit=535
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=352
Planning Time: 2.903 ms
Execution Time: 1.291 ms
```
page:
```
Limit  (cost=1.00..1223.02 rows=51 width=1277) (actual time=0.465..8.285 rows=51 loops=1)
  Buffers: shared hit=635
  ->  Result  (cost=1.00..469856.12 rows=19609 width=1277) (actual time=0.462..8.267 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=635
        ->  Nested Loop Left Join  (cost=1.00..469709.05 rows=19609 width=1250) (actual time=0.442..8.173 rows=51 loops=1)
              Buffers: shared hit=635
              ->  Nested Loop  (cost=0.70..461627.30 rows=19609 width=1242) (actual time=0.369..7.387 rows=51 loops=1)
                    Buffers: shared hit=482
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..461373.90 rows=19609 width=1235) (actual time=0.266..7.208 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (state = ANY ('{new,in_progress}'::text[]))
                          Rows Removed by Filter: 419
                          Buffers: shared hit=479
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.040..0.041 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.40 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.40 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=584
Planning Time: 3.021 ms
Execution Time: 8.462 ms
```
