## agent-state-new page=1
count:
```
Aggregate  (cost=561.75..561.76 rows=1 width=8) (actual time=2.288..2.295 rows=1 loops=1)
  Buffers: shared hit=534
  ->  Limit  (cost=0.55..558.00 rows=300 width=4) (actual time=0.228..2.152 rows=1001 loops=1)
        Buffers: shared hit=534
        ->  Result  (cost=0.55..558.00 rows=300 width=4) (actual time=0.226..1.983 rows=1001 loops=1)
              Buffers: shared hit=534
              ->  Append  (cost=0.55..555.00 rows=300 width=0) (actual time=0.225..1.782 rows=1001 loops=1)
                    Buffers: shared hit=534
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..29.17 rows=293 width=0) (actual time=0.224..1.644 rows=1001 loops=1)
                          Buffers: shared hit=534
                          ->  Limit  (cost=0.55..26.24 rows=293 width=4) (actual time=0.223..1.482 rows=1001 loops=1)
                                Buffers: shared hit=534
                                ->  Result  (cost=0.55..26.24 rows=293 width=4) (actual time=0.222..1.341 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=534
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..26.24 rows=293 width=0) (actual time=0.208..1.147 rows=1001 loops=1)
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
  Buffers: shared hit=578
Planning Time: 3.994 ms
Execution Time: 2.740 ms
```
page:
```
Nested Loop Left Join  (cost=1216.27..1655.39 rows=1 width=1285) (actual time=12.766..13.367 rows=51 loops=1)
  Buffers: shared hit=1584
  ->  Nested Loop  (cost=1215.97..1647.05 rows=1 width=1250) (actual time=12.712..13.161 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1431
        ->  Subquery Scan on x  (cost=1215.69..1638.74 rows=1 width=1243) (actual time=12.661..13.003 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1278
              ->  Limit  (cost=1215.69..1637.59 rows=51 width=1243) (actual time=12.657..12.963 rows=51 loops=1)
                    Buffers: shared hit=1278
                    ->  Merge Append  (cost=1215.69..1695.50 rows=58 width=1243) (actual time=12.655..12.957 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1278
                          ->  Limit  (cost=1170.19..1170.32 rows=51 width=1243) (actual time=12.069..12.087 rows=50 loops=1)
                                Buffers: shared hit=1186
                                ->  Sort  (cost=1170.19..1170.92 rows=293 width=1243) (actual time=12.068..12.081 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=1186
                                      ->  Result  (cost=23.40..1160.42 rows=293 width=1243) (actual time=0.895..10.987 rows=1168 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=1186
                                            ->  Bitmap Heap Scan on records r  (cost=23.40..1160.42 rows=293 width=1235) (actual time=0.889..10.658 rows=1168 loops=1)
                                                  Recheck Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                  Heap Blocks: exact=1156
                                                  Buffers: shared hit=1186
                                                  ->  Bitmap Index Scan on records_list_group_state  (cost=0.00..23.31 rows=293 width=0) (actual time=0.701..0.701 rows=1168 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])) AND (state = ANY ('{new}'::text[])))
                                                        Buffers: shared hit=30
                          ->  Limit  (cost=0.55..427.42 rows=5 width=1243) (actual time=0.139..0.399 rows=2 loops=1)
                                Buffers: shared hit=47
                                ->  Result  (cost=0.55..427.42 rows=5 width=1243) (actual time=0.137..0.397 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=47
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.42 rows=5 width=1235) (actual time=0.123..0.380 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 40
                                            Buffers: shared hit=47
                          ->  Limit  (cost=0.42..52.37 rows=1 width=1243) (actual time=0.184..0.185 rows=1 loops=1)
                                Buffers: shared hit=20
                                ->  Result  (cost=0.42..52.37 rows=1 width=1243) (actual time=0.184..0.184 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=20
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.37 rows=1 width=1235) (actual time=0.181..0.181 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Rows Removed by Filter: 16
                                            Buffers: shared hit=20
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.256..0.260 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.256..0.259 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1243) (actual time=0.217..0.220 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.213..0.215 rows=1 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.114..0.117 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.085..0.106 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.064..0.064 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.025..0.026 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.037..0.037 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.031..0.031 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: ((state = ANY ('{new}'::text[])) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=696
Planning Time: 3.659 ms
Execution Time: 13.760 ms
```
## agent page=1
count:
```
Aggregate  (cost=577.73..577.74 rows=1 width=8) (actual time=1.240..1.247 rows=1 loops=1)
  Buffers: shared hit=521
  ->  Limit  (cost=0.55..565.22 rows=1001 width=4) (actual time=0.269..1.165 rows=1001 loops=1)
        Buffers: shared hit=521
        ->  Result  (cost=0.55..629.52 rows=1115 width=4) (actual time=0.266..1.079 rows=1001 loops=1)
              Buffers: shared hit=521
              ->  Append  (cost=0.55..618.37 rows=1115 width=0) (actual time=0.265..0.965 rows=1001 loops=1)
                    Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..87.55 rows=1001 width=0) (actual time=0.264..0.883 rows=1001 loops=1)
                          Buffers: shared hit=521
                          ->  Limit  (cost=0.55..77.54 rows=1001 width=4) (actual time=0.263..0.785 rows=1001 loops=1)
                                Buffers: shared hit=521
                                ->  Result  (cost=0.55..453.40 rows=5888 width=4) (actual time=0.262..0.711 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=521
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..453.40 rows=5888 width=0) (actual time=0.250..0.597 rows=1001 loops=1)
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
  Buffers: shared hit=524
Planning Time: 2.706 ms
Execution Time: 1.504 ms
```
page:
```
Nested Loop Left Join  (cost=46.50..1957.22 rows=1 width=1285) (actual time=1.137..7.257 rows=51 loops=1)
  Buffers: shared hit=1140
  ->  Nested Loop  (cost=46.21..1948.88 rows=1 width=1250) (actual time=1.088..7.045 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=987
        ->  Subquery Scan on x  (cost=45.92..1940.57 rows=1 width=1243) (actual time=1.058..6.883 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=834
              ->  Limit  (cost=45.92..1939.42 rows=51 width=1243) (actual time=1.056..6.839 rows=51 loops=1)
                    Buffers: shared hit=834
                    ->  Merge Append  (cost=45.92..4315.58 rows=115 width=1243) (actual time=1.055..6.831 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=834
                          ->  Limit  (cost=0.42..4000.94 rows=51 width=1243) (actual time=0.647..6.388 rows=50 loops=1)
                                Buffers: shared hit=799
                                ->  Result  (cost=0.42..461863.95 rows=5888 width=1243) (actual time=0.646..6.380 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=799
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..461863.95 rows=5888 width=1235) (actual time=0.641..6.359 rows=50 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 737
                                            Buffers: shared hit=799
                          ->  Limit  (cost=0.55..216.03 rows=51 width=1243) (actual time=0.074..0.089 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..427.29 rows=101 width=1243) (actual time=0.073..0.088 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=1235) (actual time=0.071..0.084 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.42..52.35 rows=12 width=1243) (actual time=0.045..0.046 rows=1 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=0.42..52.35 rows=12 width=1243) (actual time=0.044..0.045 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=4
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=1235) (actual time=0.043..0.043 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=4
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1243) (actual time=0.287..0.291 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1243) (actual time=0.287..0.291 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1243) (actual time=0.172..0.272 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.170..0.269 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.102..0.105 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.080..0.096 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.063..0.064 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.028..0.028 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.035..0.035 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.053..0.053 rows=1 loops=3)
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
  Buffers: shared hit=645
Planning Time: 4.701 ms
Execution Time: 7.616 ms
```
## agent page=20
page:
```
Nested Loop Left Join  (cost=46.51..2004.58 rows=1 width=1285) (actual time=1.358..7.608 rows=51 loops=1)
  Buffers: shared hit=1125
  ->  Nested Loop  (cost=46.21..1996.24 rows=1 width=1250) (actual time=1.309..7.362 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=972
        ->  Subquery Scan on x  (cost=45.93..1987.93 rows=1 width=1243) (actual time=1.276..7.200 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=819
              ->  Limit  (cost=45.93..1986.78 rows=51 width=1243) (actual time=1.269..7.138 rows=51 loops=1)
                    Buffers: shared hit=819
                    ->  Merge Append  (cost=45.93..4384.31 rows=114 width=1243) (actual time=1.266..7.126 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=819
                          ->  Limit  (cost=0.42..4071.13 rows=51 width=1243) (actual time=0.664..6.481 rows=49 loops=1)
                                Buffers: shared hit=783
                                ->  Result  (cost=0.42..451370.24 rows=5655 width=1243) (actual time=0.663..6.472 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=783
                                      ->  Index Scan using records_list_updated on records r  (cost=0.42..451370.24 rows=5655 width=1235) (actual time=0.648..6.438 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Rows Removed by Filter: 721
                                            Buffers: shared hit=783
                          ->  Limit  (cost=0.55..214.56 rows=51 width=1243) (actual time=0.092..0.100 rows=2 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=0.55..407.60 rows=97 width=1243) (actual time=0.091..0.099 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=6
                                      ->  Index Scan using records_list_requester on records r_1  (cost=0.55..407.60 rows=97 width=1235) (actual time=0.087..0.094 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            Buffers: shared hit=6
                          ->  Limit  (cost=0.42..52.37 rows=11 width=1243) (actual time=0.046..0.058 rows=2 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.42..52.37 rows=11 width=1243) (actual time=0.045..0.057 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.37 rows=11 width=1235) (actual time=0.043..0.054 rows=2 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                                            Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            Buffers: shared hit=5
                          ->  Limit  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.462..0.466 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.49 rows=1 width=1243) (actual time=0.461..0.465 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1243) (actual time=0.340..0.447 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.337..0.443 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.258..0.262 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.225..0.251 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.153..0.154 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.048..0.048 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.104..0.104 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.058..0.058 rows=1 loops=3)
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
  Buffers: shared hit=634
Planning Time: 4.067 ms
Execution Time: 8.127 ms
```
## agent-group-desc page=1
count:
```
Aggregate  (cost=577.73..577.74 rows=1 width=8) (actual time=1.599..1.612 rows=1 loops=1)
  Buffers: shared hit=521
  ->  Limit  (cost=0.55..565.22 rows=1001 width=4) (actual time=0.225..1.504 rows=1001 loops=1)
        Buffers: shared hit=521
        ->  Result  (cost=0.55..629.52 rows=1115 width=4) (actual time=0.224..1.397 rows=1001 loops=1)
              Buffers: shared hit=521
              ->  Append  (cost=0.55..618.37 rows=1115 width=0) (actual time=0.223..1.210 rows=1001 loops=1)
                    Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..87.55 rows=1001 width=0) (actual time=0.222..1.104 rows=1001 loops=1)
                          Buffers: shared hit=521
                          ->  Limit  (cost=0.55..77.54 rows=1001 width=4) (actual time=0.220..0.983 rows=1001 loops=1)
                                Buffers: shared hit=521
                                ->  Result  (cost=0.55..453.40 rows=5888 width=4) (actual time=0.220..0.892 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=521
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..453.40 rows=5888 width=0) (actual time=0.206..0.745 rows=1001 loops=1)
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
  Buffers: shared hit=524
Planning Time: 2.668 ms
Execution Time: 1.901 ms
```
page:
```
Nested Loop Left Join  (cost=528.87..2331.89 rows=1 width=1293) (actual time=189.109..189.420 rows=51 loops=1)
  Buffers: shared hit=57636
  ->  Nested Loop  (cost=528.57..2323.55 rows=1 width=1258) (actual time=189.063..189.219 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=57483
        ->  Subquery Scan on x  (cost=528.29..2315.24 rows=1 width=1251) (actual time=189.019..189.089 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=57330
              ->  Limit  (cost=528.29..2314.09 rows=51 width=1251) (actual time=189.014..189.054 rows=51 loops=1)
                    Buffers: shared hit=57330
                    ->  Merge Append  (cost=528.29..4555.10 rows=115 width=1251) (actual time=189.013..189.048 rows=51 loops=1)
                          Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                          Buffers: shared hit=57330
                          ->  Limit  (cost=0.55..4025.48 rows=51 width=1251) (actual time=186.558..186.559 rows=1 loops=1)
                                Buffers: shared hit=56714
                                ->  Result  (cost=0.55..464682.13 rows=5888 width=1251) (actual time=186.556..186.557 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=56714
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..464682.13 rows=5888 width=1235) (actual time=186.547..186.547 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL))
                                            Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                            Rows Removed by Filter: 55887
                                            Buffers: shared hit=56714
                          ->  Limit  (cost=430.65..430.78 rows=51 width=1251) (actual time=1.830..1.841 rows=44 loops=1)
                                Buffers: shared hit=524
                                ->  Sort  (cost=430.65..430.90 rows=101 width=1251) (actual time=1.828..1.835 rows=44 loops=1)
                                      Sort Key: r_1.assignment_group_id DESC, r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: top-N heapsort  Memory: 195kB
                                      Buffers: shared hit=524
                                      ->  Result  (cost=0.55..427.29 rows=101 width=1251) (actual time=0.068..1.326 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=524
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=1235) (actual time=0.056..1.209 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=524
                          ->  Limit  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.348..0.351 rows=8 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.347..0.349 rows=8 loops=1)
                                      Sort Key: r_2.assignment_group_id DESC, r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 147kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..52.35 rows=12 width=1251) (actual time=0.036..0.223 rows=61 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=1235) (actual time=0.025..0.198 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.269..0.271 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.268..0.270 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id DESC, r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.219..0.256 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.215..0.251 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.187..0.189 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.155..0.178 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.130..0.130 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.046..0.046 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.082..0.082 rows=0 loops=1)
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
  Buffers: shared hit=645
Planning Time: 5.035 ms
Execution Time: 190.002 ms
```
## agent-group-desc page=20
page:
```
Nested Loop Left Join  (cost=529.18..2470.53 rows=1 width=1293) (actual time=8.115..8.920 rows=51 loops=1)
  Buffers: shared hit=978
  ->  Nested Loop  (cost=528.89..2462.20 rows=1 width=1258) (actual time=8.073..8.706 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=825
        ->  Subquery Scan on x  (cost=528.60..2453.88 rows=1 width=1251) (actual time=8.044..8.562 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=672
              ->  Limit  (cost=528.60..2452.74 rows=51 width=1251) (actual time=8.038..8.514 rows=51 loops=1)
                    Buffers: shared hit=672
                    ->  Merge Append  (cost=528.60..4791.88 rows=113 width=1251) (actual time=8.037..8.504 rows=51 loops=1)
                          Sort Key: r.assignment_group_id DESC, r.updated_at DESC, r.id DESC
                          Buffers: shared hit=672
                          ->  Limit  (cost=0.55..4261.97 rows=51 width=1251) (actual time=0.232..0.679 rows=51 loops=1)
                                Buffers: shared hit=56
                                ->  Result  (cost=0.55..426225.93 rows=5101 width=1251) (actual time=0.231..0.671 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=56
                                      ->  Index Scan using records_list_group on records r  (cost=0.55..426225.93 rows=5101 width=1235) (actual time=0.224..0.647 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                            Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                            Buffers: shared hit=56
                          ->  Limit  (cost=430.92..431.04 rows=51 width=1251) (actual time=6.825..6.826 rows=1 loops=1)
                                Buffers: shared hit=524
                                ->  Sort  (cost=430.92..431.14 rows=88 width=1251) (actual time=6.823..6.824 rows=1 loops=1)
                                      Sort Key: r_1.assignment_group_id DESC, r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=524
                                      ->  Result  (cost=0.55..428.07 rows=88 width=1251) (actual time=0.372..6.283 rows=412 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=524
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..428.07 rows=88 width=1235) (actual time=0.368..6.139 rows=412 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                  Rows Removed by Filter: 101
                                                  Buffers: shared hit=524
                          ->  Limit  (cost=52.61..52.63 rows=10 width=1251) (actual time=0.690..0.691 rows=1 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=52.61..52.63 rows=10 width=1251) (actual time=0.688..0.689 rows=1 loops=1)
                                      Sort Key: r_2.assignment_group_id DESC, r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 121kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..52.44 rows=10 width=1251) (actual time=0.047..0.576 rows=48 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.44 rows=10 width=1235) (actual time=0.038..0.552 rows=48 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                  Rows Removed by Filter: 16
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.49..44.50 rows=1 width=1251) (actual time=0.283..0.287 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.49..44.50 rows=1 width=1251) (actual time=0.282..0.285 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id DESC, r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.48 rows=1 width=1251) (actual time=0.205..0.271 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.48 rows=1 width=1235) (actual time=0.201..0.264 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.154..0.157 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.123..0.145 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.100..0.101 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.053..0.053 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.046..0.046 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.034..0.034 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) AND (ROW(assignment_group_id, updated_at, id) < ROW('e43d6642-9bbe-05be-b6a6-8d3a87bf4141'::uuid, '2026-09-30 11:50:01.62409+00'::timestamp with time zone, '0ec6dc88-cedd-0aec-ddcc-c4bd2d12942a'::uuid)))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=634
Planning Time: 3.673 ms
Execution Time: 9.331 ms
```
## agent-group-asc page=1
count:
```
Aggregate  (cost=577.73..577.74 rows=1 width=8) (actual time=1.351..1.357 rows=1 loops=1)
  Buffers: shared hit=521
  ->  Limit  (cost=0.55..565.22 rows=1001 width=4) (actual time=0.157..1.256 rows=1001 loops=1)
        Buffers: shared hit=521
        ->  Result  (cost=0.55..629.52 rows=1115 width=4) (actual time=0.156..1.146 rows=1001 loops=1)
              Buffers: shared hit=521
              ->  Append  (cost=0.55..618.37 rows=1115 width=0) (actual time=0.155..1.004 rows=1001 loops=1)
                    Buffers: shared hit=521
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.55..87.55 rows=1001 width=0) (actual time=0.155..0.904 rows=1001 loops=1)
                          Buffers: shared hit=521
                          ->  Limit  (cost=0.55..77.54 rows=1001 width=4) (actual time=0.154..0.791 rows=1001 loops=1)
                                Buffers: shared hit=521
                                ->  Result  (cost=0.55..453.40 rows=5888 width=4) (actual time=0.153..0.699 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=521
                                      ->  Index Only Scan using records_list_group_state on records r  (cost=0.55..453.40 rows=5888 width=0) (actual time=0.144..0.562 rows=1001 loops=1)
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
  Buffers: shared hit=524
Planning Time: 2.802 ms
Execution Time: 1.604 ms
```
page:
```
Nested Loop Left Join  (cost=528.87..2331.89 rows=1 width=1293) (actual time=151.549..151.997 rows=51 loops=1)
  Buffers: shared hit=50202
  ->  Nested Loop  (cost=528.57..2323.55 rows=1 width=1258) (actual time=151.496..151.704 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=50049
        ->  Subquery Scan on x  (cost=528.29..2315.24 rows=1 width=1251) (actual time=151.456..151.543 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=49896
              ->  Limit  (cost=528.29..2314.09 rows=51 width=1251) (actual time=151.450..151.497 rows=51 loops=1)
                    Buffers: shared hit=49896
                    ->  Merge Append  (cost=528.29..4555.10 rows=115 width=1251) (actual time=151.449..151.491 rows=51 loops=1)
                          Sort Key: r.assignment_group_id, r.updated_at, r.id
                          Buffers: shared hit=49896
                          ->  Limit  (cost=0.55..4025.48 rows=51 width=1251) (actual time=148.510..148.511 rows=1 loops=1)
                                Buffers: shared hit=49280
                                ->  Result  (cost=0.55..464682.13 rows=5888 width=1251) (actual time=148.509..148.510 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=49280
                                      ->  Index Scan Backward using records_list_group on records r  (cost=0.55..464682.13 rows=5888 width=1235) (actual time=148.504..148.504 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignment_group_id IS NOT NULL))
                                            Filter: (assignment_group_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))
                                            Rows Removed by Filter: 47804
                                            Buffers: shared hit=49280
                          ->  Limit  (cost=430.65..430.78 rows=51 width=1251) (actual time=1.937..1.948 rows=44 loops=1)
                                Buffers: shared hit=524
                                ->  Sort  (cost=430.65..430.90 rows=101 width=1251) (actual time=1.934..1.942 rows=44 loops=1)
                                      Sort Key: r_1.assignment_group_id, r_1.updated_at, r_1.id
                                      Sort Method: top-N heapsort  Memory: 191kB
                                      Buffers: shared hit=524
                                      ->  Result  (cost=0.55..427.29 rows=101 width=1251) (actual time=0.061..1.324 rows=484 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=524
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..427.29 rows=101 width=1235) (actual time=0.049..1.198 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=524
                          ->  Limit  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.506..0.509 rows=8 loops=1)
                                Buffers: shared hit=67
                                ->  Sort  (cost=52.57..52.60 rows=12 width=1251) (actual time=0.504..0.506 rows=8 loops=1)
                                      Sort Key: r_2.assignment_group_id, r_2.updated_at, r_2.id
                                      Sort Method: quicksort  Memory: 147kB
                                      Buffers: shared hit=67
                                      ->  Result  (cost=0.42..52.35 rows=12 width=1251) (actual time=0.040..0.234 rows=61 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=67
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..52.35 rows=12 width=1235) (actual time=0.031..0.186 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                          ->  Limit  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.485..0.488 rows=1 loops=1)
                                Buffers: shared hit=25
                                ->  Sort  (cost=44.48..44.49 rows=1 width=1251) (actual time=0.483..0.486 rows=1 loops=1)
                                      Sort Key: r_3.assignment_group_id, r_3.updated_at, r_3.id
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=25
                                      ->  Result  (cost=36.42..44.47 rows=1 width=1251) (actual time=0.423..0.464 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=25
                                            ->  Nested Loop  (cost=36.42..44.47 rows=1 width=1235) (actual time=0.417..0.456 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.382..0.385 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.343..0.368 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.253..0.254 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.098..0.098 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.153..0.153 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.021..0.021 rows=1 loops=3)
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
  Buffers: shared hit=622
Planning Time: 3.463 ms
Execution Time: 152.315 ms
```
## admin-state-in page=1
count:
```
Aggregate  (cost=91.23..91.24 rows=1 width=8) (actual time=0.931..0.932 rows=1 loops=1)
  Buffers: shared hit=535
  ->  Limit  (cost=0.42..78.71 rows=1001 width=4) (actual time=0.097..0.861 rows=1001 loops=1)
        Buffers: shared hit=535
        ->  Result  (cost=0.42..1534.04 rows=19609 width=4) (actual time=0.096..0.774 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
              Buffers: shared hit=535
              ->  Index Only Scan using records_list_state on records r  (cost=0.42..1534.04 rows=19609 width=0) (actual time=0.091..0.648 rows=1001 loops=1)
                    Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (state = ANY ('{new,in_progress}'::text[])))
                    Heap Fetches: 0
                    Buffers: shared hit=535
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=376
Planning Time: 1.093 ms
Execution Time: 0.991 ms
```
page:
```
Limit  (cost=1.00..1223.02 rows=51 width=1277) (actual time=0.766..5.903 rows=51 loops=1)
  Buffers: shared hit=635
  ->  Result  (cost=1.00..469856.12 rows=19609 width=1277) (actual time=0.763..5.891 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=635
        ->  Nested Loop Left Join  (cost=1.00..469709.05 rows=19609 width=1250) (actual time=0.736..5.823 rows=51 loops=1)
              Buffers: shared hit=635
              ->  Nested Loop  (cost=0.70..461627.30 rows=19609 width=1242) (actual time=0.628..5.343 rows=51 loops=1)
                    Buffers: shared hit=482
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..461373.90 rows=19609 width=1235) (actual time=0.575..5.247 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (state = ANY ('{new,in_progress}'::text[]))
                          Rows Removed by Filter: 419
                          Buffers: shared hit=479
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.036..0.037 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.40 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.40 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off', plan_cache_mode = 'force_custom_plan'
Planning:
  Buffers: shared hit=608
Planning Time: 3.754 ms
Execution Time: 6.322 ms
```
