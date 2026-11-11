## agent sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..676.18 rows=51 width=1277) (actual time=2.150..22.576 rows=51 loops=1)
  Buffers: shared hit=969
  ->  Result  (cost=1.00..1359947.88 rows=102724 width=1277) (actual time=2.145..22.496 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=969
        ->  Nested Loop Left Join  (cost=1.00..1359177.45 rows=102724 width=1250) (actual time=2.116..22.275 rows=51 loops=1)
              Buffers: shared hit=969
              ->  Nested Loop  (cost=0.70..1324706.80 rows=102724 width=1242) (actual time=1.993..20.152 rows=51 loops=1)
                    Buffers: shared hit=816
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..1323414.45 rows=102724 width=1235) (actual time=1.879..19.856 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 736
                          Buffers: shared hit=813
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.182..0.218 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.143..0.146 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.051..0.052 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.090..0.091 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.086..0.088 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.038..0.038 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.021..0.021 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 3.618 ms
Execution Time: 22.917 ms
```
count:
```
Aggregate  (cost=1323671.26..1323671.27 rows=1 width=8) (actual time=1181.438..1181.444 rows=1 loops=1)
  Buffers: shared hit=403755
  ->  Result  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.394..1175.879 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403755
        ->  Index Scan using records_list_created on records r  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.385..1169.399 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=403755
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.141..0.178 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.114..0.116 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.040..0.041 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.072..0.072 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=413
Planning Time: 2.679 ms
Execution Time: 1181.595 ms
```
## agent sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND (r.updated_at, r.id) < ('2026-09-30 23:25:18.681999+00'::timestamptz, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..680.53 rows=51 width=1277) (actual time=0.924..8.317 rows=51 loops=1)
  Buffers: shared hit=949
  ->  Result  (cost=1.00..1311425.70 rows=98424 width=1277) (actual time=0.922..8.302 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=949
        ->  Nested Loop Left Join  (cost=1.00..1310687.52 rows=98424 width=1250) (actual time=0.898..8.225 rows=51 loops=1)
              Buffers: shared hit=949
              ->  Nested Loop  (cost=0.70..1277582.13 rows=98424 width=1242) (actual time=0.846..7.890 rows=51 loops=1)
                    Buffers: shared hit=796
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..1276343.53 rows=98424 width=1235) (actual time=0.795..7.784 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:25:18.681999+00'::timestamp with time zone, '76100c27-3cdc-b7ab-2ac4-15099bf48fa5'::uuid)))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 719
                          Buffers: shared hit=793
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.157..0.192 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.124..0.125 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.043..0.043 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.080..0.081 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.039..0.040 rows=1 loops=1)
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
  Buffers: shared hit=581
Planning Time: 2.987 ms
Execution Time: 8.594 ms
```
count:
```
Aggregate  (cost=1323671.26..1323671.27 rows=1 width=8) (actual time=1129.189..1129.196 rows=1 loops=1)
  Buffers: shared hit=403755
  ->  Result  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.329..1122.989 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403755
        ->  Index Scan using records_list_created on records r  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.298..1117.389 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=403755
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.110..0.136 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.088..0.090 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.031..0.032 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.056..0.056 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=413
Planning Time: 1.712 ms
Execution Time: 1129.312 ms
```
## req1 sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..690.76 rows=51 width=1277) (actual time=1.345..37.698 rows=51 loops=1)
  Buffers: shared hit=4377
  ->  Result  (cost=1.00..1357407.67 rows=100364 width=1277) (actual time=1.342..37.666 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=4377
        ->  Nested Loop Left Join  (cost=1.00..1356654.94 rows=100364 width=1250) (actual time=1.315..37.528 rows=51 loops=1)
              Buffers: shared hit=4377
              ->  Nested Loop  (cost=0.70..1322933.59 rows=100364 width=1242) (actual time=1.256..36.714 rows=51 loops=1)
                    Buffers: shared hit=4224
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..1321670.75 rows=100364 width=1235) (actual time=1.205..36.538 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 4126
                          Buffers: shared hit=4221
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.046..0.049 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.044..0.047 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.043..0.043 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.038..0.040 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 3.022 ms
Execution Time: 38.013 ms
```
count:
```
Aggregate  (cost=1321921.66..1321921.67 rows=1 width=8) (actual time=1141.856..1141.863 rows=1 loops=1)
  Buffers: shared hit=403741
  ->  Result  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=1.008..1139.146 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403741
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=1.000..1136.899 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=403741
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.040..0.043 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.036..0.040 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.035..0.035 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 2.133 ms
Execution Time: 1141.998 ms
```
## req1 sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.updated_at, r.id) < ('2026-09-29 03:10:06.448328+00'::timestamptz, '0150dafd-b1c7-41f0-99ea-b49cdcff90ec'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..709.55 rows=51 width=1277) (actual time=5.828..43.369 rows=51 loops=1)
  Buffers: shared hit=4596
  ->  Result  (cost=1.00..1152036.94 rows=82921 width=1277) (actual time=5.825..43.333 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=4596
        ->  Nested Loop Left Join  (cost=1.00..1151415.03 rows=82921 width=1250) (actual time=5.804..43.156 rows=51 loops=1)
              Buffers: shared hit=4596
              ->  Nested Loop  (cost=0.70..1123231.80 rows=82921 width=1242) (actual time=5.740..41.966 rows=51 loops=1)
                    Buffers: shared hit=4443
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..1122186.99 rows=82921 width=1235) (actual time=5.657..41.723 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:10:06.448328+00'::timestamp with time zone, '0150dafd-b1c7-41f0-99ea-b49cdcff90ec'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 4343
                          Buffers: shared hit=4440
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.044..0.047 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.040..0.042 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.039..0.039 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.056..0.057 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.020..0.020 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.013..0.013 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 3.219 ms
Execution Time: 43.715 ms
```
count:
```
Aggregate  (cost=1321921.66..1321921.67 rows=1 width=8) (actual time=753.241..753.247 rows=1 loops=1)
  Buffers: shared hit=403741
  ->  Result  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=0.693..751.985 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403741
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=0.688..750.759 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=403741
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.036..0.040 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.033..0.036 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.032..0.032 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 1.780 ms
Execution Time: 753.377 ms
```
## req8 sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..694.41 rows=51 width=1277) (actual time=1.349..104.516 rows=51 loops=1)
  Buffers: shared hit=18791
  ->  Result  (cost=1.00..1357225.08 rows=99823 width=1277) (actual time=1.348..104.461 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=18791
        ->  Nested Loop Left Join  (cost=1.00..1356476.41 rows=99823 width=1250) (actual time=1.327..104.174 rows=51 loops=1)
              Buffers: shared hit=18791
              ->  Nested Loop  (cost=0.70..1322926.83 rows=99823 width=1242) (actual time=1.287..102.835 rows=51 loops=1)
                    Buffers: shared hit=18638
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..1321670.75 rows=99823 width=1235) (actual time=1.247..102.581 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 18405
                          Buffers: shared hit=18635
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.063..0.081 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.039..0.041 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.039..0.039 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.030..0.031 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.022..0.022 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.013..0.013 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 2.992 ms
Execution Time: 104.755 ms
```
count:
```
Aggregate  (cost=1321920.30..1321920.31 rows=1 width=8) (actual time=1298.123..1298.131 rows=1 loops=1)
  Buffers: shared hit=403743
  ->  Result  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=15.285..1296.205 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403743
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=15.273..1294.875 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=403743
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.088..0.113 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.058..0.061 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.056..0.056 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 2.530 ms
Execution Time: 1298.281 ms
```
## req8 sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.updated_at, r.id) < ('2026-02-15 00:27:00.844898+00'::timestamptz, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..832.30 rows=51 width=1277) (actual time=7.014..113.592 rows=51 loops=1)
  Buffers: shared hit=20739
  ->  Result  (cost=1.00..237722.26 rows=14584 width=1277) (actual time=7.012..113.532 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=20739
        ->  Nested Loop Left Join  (cost=1.00..237612.88 rows=14584 width=1250) (actual time=6.995..113.274 rows=51 loops=1)
              Buffers: shared hit=20739
              ->  Nested Loop  (cost=0.70..231126.60 rows=14584 width=1242) (actual time=6.911..111.803 rows=51 loops=1)
                    Buffers: shared hit=20586
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..230936.01 rows=14584 width=1235) (actual time=6.835..111.499 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-02-15 00:27:00.844898+00'::timestamp with time zone, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 20334
                          Buffers: shared hit=20583
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.064..0.084 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.041..0.043 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.040..0.041 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.056..0.057 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.43 rows=1 width=40) (actual time=0.025..0.025 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.43 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 2.410 ms
Execution Time: 113.901 ms
```
count:
```
Aggregate  (cost=1321920.30..1321920.31 rows=1 width=8) (actual time=1217.102..1217.108 rows=1 loops=1)
  Buffers: shared hit=403743
  ->  Result  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=17.341..1215.257 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403743
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=17.330..1213.918 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=403743
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.076..0.098 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.047..0.049 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.045..0.046 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 2.258 ms
Execution Time: 1217.309 ms
```
## admin sort=-updated_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..136.74 rows=51 width=1277) (actual time=0.311..2.434 rows=51 loops=1)
  Buffers: shared hit=210
  ->  Result  (cost=1.00..530396.30 rows=199281 width=1277) (actual time=0.308..2.418 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=210
        ->  Nested Loop Left Join  (cost=1.00..528901.69 rows=199281 width=1250) (actual time=0.279..2.334 rows=51 loops=1)
              Buffers: shared hit=210
              ->  Nested Loop  (cost=0.70..463774.34 rows=199281 width=1242) (actual time=0.215..1.703 rows=51 loops=1)
                    Buffers: shared hit=57
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..461275.03 rows=199281 width=1235) (actual time=0.152..1.582 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Buffers: shared hit=54
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.049..0.051 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.010..0.010 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=530
Planning Time: 3.878 ms
Execution Time: 2.642 ms
```
count:
```
Finalize Aggregate  (cost=11551.37..11551.38 rows=1 width=8) (actual time=142.270..149.252 rows=1 loops=1)
  Buffers: shared hit=211365
  ->  Gather  (cost=11551.16..11551.37 rows=2 width=8) (actual time=142.075..149.237 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=211365
        ->  Partial Aggregate  (cost=10551.16..10551.17 rows=1 width=8) (actual time=135.484..135.486 rows=1 loops=3)
              Buffers: shared hit=211365
              ->  Result  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.189..118.759 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=211365
                    ->  Parallel Index Only Scan using records_list_created on records r  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.167..93.856 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=211365
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=286
Planning Time: 1.300 ms
Execution Time: 149.374 ms
```
## admin sort=-updated_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND (r.updated_at, r.id) < ('2026-09-30 23:59:59.561867+00'::timestamptz, 'fb3ab87c-de21-eb9c-909b-5f44410a93b9'::uuid) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..136.87 rows=51 width=1277) (actual time=0.266..2.044 rows=51 loops=1)
  Buffers: shared hit=210
  ->  Result  (cost=1.00..530833.82 rows=199244 width=1277) (actual time=0.264..2.030 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=210
        ->  Nested Loop Left Join  (cost=1.00..529339.49 rows=199244 width=1250) (actual time=0.237..1.953 rows=51 loops=1)
              Buffers: shared hit=210
              ->  Nested Loop  (cost=0.70..464223.88 rows=199244 width=1242) (actual time=0.194..1.370 rows=51 loops=1)
                    Buffers: shared hit=57
                    ->  Index Scan using records_list_updated on records r  (cost=0.42..461725.04 rows=199244 width=1235) (actual time=0.147..1.269 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-30 23:59:59.561867+00'::timestamp with time zone, 'fb3ab87c-de21-eb9c-909b-5f44410a93b9'::uuid)))
                          Buffers: shared hit=54
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.039..0.040 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=519
Planning Time: 2.466 ms
Execution Time: 2.214 ms
```
count:
```
Finalize Aggregate  (cost=11551.37..11551.38 rows=1 width=8) (actual time=100.071..107.492 rows=1 loops=1)
  Buffers: shared hit=211352
  ->  Gather  (cost=11551.16..11551.37 rows=2 width=8) (actual time=99.790..107.473 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=211352
        ->  Partial Aggregate  (cost=10551.16..10551.17 rows=1 width=8) (actual time=95.245..95.248 rows=1 loops=3)
              Buffers: shared hit=211352
              ->  Result  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.159..83.848 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=211352
                    ->  Parallel Index Only Scan using records_list_created on records r  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.144..64.936 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=211352
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=286
Planning Time: 1.049 ms
Execution Time: 107.621 ms
```
## agent sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..676.18 rows=51 width=1277) (actual time=0.694..12.034 rows=51 loops=1)
  Buffers: shared hit=1083
  ->  Result  (cost=1.00..1359947.88 rows=102724 width=1277) (actual time=0.693..12.015 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=1083
        ->  Nested Loop Left Join  (cost=1.00..1359177.45 rows=102724 width=1250) (actual time=0.671..11.918 rows=51 loops=1)
              Buffers: shared hit=1083
              ->  Nested Loop  (cost=0.70..1324706.80 rows=102724 width=1242) (actual time=0.559..11.408 rows=51 loops=1)
                    Buffers: shared hit=930
                    ->  Index Scan using records_list_created on records r  (cost=0.42..1323414.45 rows=102724 width=1235) (actual time=0.511..11.275 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 849
                          Buffers: shared hit=927
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.212..0.244 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.178..0.180 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.071..0.071 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.106..0.107 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.038..0.039 rows=1 loops=1)
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
Planning Time: 3.798 ms
Execution Time: 12.687 ms
```
count:
```
Aggregate  (cost=1323671.26..1323671.27 rows=1 width=8) (actual time=789.193..789.198 rows=1 loops=1)
  Buffers: shared hit=403755
  ->  Result  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.311..786.455 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403755
        ->  Index Scan using records_list_created on records r  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.307..783.011 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=403755
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.116..0.140 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.093..0.095 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.033..0.034 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.059..0.059 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=413
Planning Time: 1.546 ms
Execution Time: 789.317 ms
```
## agent sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR (((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))))) AND (r.created_at, r.id) < ('2026-09-24 15:42:07.067406+00'::timestamptz, '345d89b0-b073-a6ef-d0d1-641e3d71e8cd'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..680.63 rows=51 width=1277) (actual time=1.192..11.954 rows=51 loops=1)
  Buffers: shared hit=1228
  ->  Result  (cost=1.00..1310362.80 rows=98331 width=1277) (actual time=1.190..11.937 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=1228
        ->  Nested Loop Left Join  (cost=1.00..1309625.31 rows=98331 width=1250) (actual time=1.169..11.854 rows=51 loops=1)
              Buffers: shared hit=1228
              ->  Nested Loop  (cost=0.70..1276549.44 rows=98331 width=1242) (actual time=1.047..11.379 rows=51 loops=1)
                    Buffers: shared hit=1075
                    ->  Index Scan using records_list_created on records r  (cost=0.42..1275312.01 rows=98331 width=1235) (actual time=0.998..11.264 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(created_at, id) < ROW('2026-09-24 15:42:07.067406+00'::timestamp with time zone, '345d89b0-b073-a6ef-d0d1-641e3d71e8cd'::uuid)))
                          Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 995
                          Buffers: shared hit=1072
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.162..0.195 rows=3 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=3
                                  Buffers: shared hit=13
                                  ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.128..0.130 rows=0 loops=1)
                                        Buffers: shared hit=10
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.047..0.048 rows=3 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.079..0.080 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                              Buffers: shared hit=8
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.037..0.039 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 3.239 ms
Execution Time: 12.283 ms
```
count:
```
Aggregate  (cost=1323671.26..1323671.27 rows=1 width=8) (actual time=1021.711..1021.717 rows=1 loops=1)
  Buffers: shared hit=403755
  ->  Result  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.525..1017.551 rows=24666 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403755
        ->  Index Scan using records_list_created on records r  (cost=0.42..1323414.45 rows=102724 width=0) (actual time=0.516..1012.425 rows=24666 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))) OR ((requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 375334
              Buffers: shared hit=403755
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.50..36.04 rows=1 width=32) (actual time=0.259..0.295 rows=3 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=3
                      Buffers: shared hit=13
                      ->  BitmapOr  (cost=21.50..21.50 rows=4 width=0) (actual time=0.213..0.215 rows=0 loops=1)
                            Buffers: shared hit=10
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.038..0.038 rows=3 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.19 rows=3 width=0) (actual time=0.174..0.175 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                  Buffers: shared hit=8
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=413
Planning Time: 2.457 ms
Execution Time: 1021.860 ms
```
## req1 sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..690.76 rows=51 width=1277) (actual time=0.717..25.655 rows=51 loops=1)
  Buffers: shared hit=3764
  ->  Result  (cost=1.00..1357407.67 rows=100364 width=1277) (actual time=0.715..25.632 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=3764
        ->  Nested Loop Left Join  (cost=1.00..1356654.94 rows=100364 width=1250) (actual time=0.696..25.538 rows=51 loops=1)
              Buffers: shared hit=3764
              ->  Nested Loop  (cost=0.70..1322933.59 rows=100364 width=1242) (actual time=0.631..24.904 rows=51 loops=1)
                    Buffers: shared hit=3611
                    ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=100364 width=1235) (actual time=0.592..24.786 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 3519
                          Buffers: shared hit=3608
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.034..0.046 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.033..0.043 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.032..0.035 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.002 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.029..0.031 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.011..0.011 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 2.706 ms
Execution Time: 25.902 ms
```
count:
```
Aggregate  (cost=1321921.66..1321921.67 rows=1 width=8) (actual time=762.507..762.513 rows=1 loops=1)
  Buffers: shared hit=403741
  ->  Result  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=0.598..761.163 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403741
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=0.592..759.837 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=403741
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.030..0.033 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.028..0.030 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.027..0.027 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 1.508 ms
Execution Time: 762.626 ms
```
## req1 sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.created_at, r.id) < ('2026-09-12 13:51:07.994144+00'::timestamptz, '1ad845b3-fc23-836e-dd8c-1da77c12053f'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..709.12 rows=51 width=1277) (actual time=1.664..41.871 rows=51 loops=1)
  Buffers: shared hit=4233
  ->  Result  (cost=1.00..1156497.50 rows=83292 width=1277) (actual time=1.662..41.825 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=4233
        ->  Nested Loop Left Join  (cost=1.00..1155872.81 rows=83292 width=1250) (actual time=1.635..41.615 rows=51 loops=1)
              Buffers: shared hit=4233
              ->  Nested Loop  (cost=0.70..1127571.79 rows=83292 width=1242) (actual time=1.512..40.235 rows=51 loops=1)
                    Buffers: shared hit=4080
                    ->  Index Scan using records_list_created on records r  (cost=0.42..1126522.34 rows=83292 width=1235) (actual time=1.439..39.987 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(created_at, id) < ROW('2026-09-12 13:51:07.994144+00'::timestamp with time zone, '1ad845b3-fc23-836e-dd8c-1da77c12053f'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 3984
                          Buffers: shared hit=4077
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.044..0.047 rows=0 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Buffers: shared hit=2
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.041..0.043 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.039..0.040 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.054..0.056 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.024..0.024 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.015..0.015 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 3.863 ms
Execution Time: 42.287 ms
```
count:
```
Aggregate  (cost=1321921.66..1321921.67 rows=1 width=8) (actual time=824.781..824.786 rows=1 loops=1)
  Buffers: shared hit=403741
  ->  Result  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=0.877..823.226 rows=5626 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403741
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=100364 width=0) (actual time=0.870..821.497 rows=5626 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 394374
              Buffers: shared hit=403741
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.042..0.045 rows=0 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Buffers: shared hit=2
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.038..0.040 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.037..0.037 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 1.973 ms
Execution Time: 824.920 ms
```
## req8 sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..694.41 rows=51 width=1277) (actual time=20.693..165.297 rows=51 loops=1)
  Buffers: shared hit=17556
  ->  Result  (cost=1.00..1357225.08 rows=99823 width=1277) (actual time=20.690..165.097 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=17556
        ->  Nested Loop Left Join  (cost=1.00..1356476.41 rows=99823 width=1250) (actual time=20.663..164.661 rows=51 loops=1)
              Buffers: shared hit=17556
              ->  Nested Loop  (cost=0.70..1322926.83 rows=99823 width=1242) (actual time=20.589..162.102 rows=51 loops=1)
                    Buffers: shared hit=17403
                    ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=99823 width=1235) (actual time=20.479..161.496 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 17181
                          Buffers: shared hit=17400
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.087..0.120 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.054..0.060 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.053..0.055 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.003..0.004 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.074..0.075 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.33 rows=1 width=40) (actual time=0.043..0.043 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.33 rows=1 width=40) (actual time=0.023..0.023 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=592
Planning Time: 5.117 ms
Execution Time: 165.718 ms
```
count:
```
Aggregate  (cost=1321920.30..1321920.31 rows=1 width=8) (actual time=1470.536..1470.544 rows=1 loops=1)
  Buffers: shared hit=403743
  ->  Result  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=29.214..1467.920 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403743
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=29.193..1466.393 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=403743
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.201..0.230 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.141..0.144 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.138..0.138 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.001..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 4.836 ms
Execution Time: 1470.883 ms
```
## req8 sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND (((r.requester_id IS NOT NULL AND r.requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) AND (r.created_at, r.id) < ('2026-01-31 15:23:22.324519+00'::timestamptz, '0014c6a0-72df-eac0-a48b-d96b066049c6'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..831.91 rows=51 width=1277) (actual time=0.971..140.203 rows=51 loops=1)
  Buffers: shared hit=20845
  ->  Result  (cost=1.00..239547.64 rows=14703 width=1277) (actual time=0.968..140.119 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=20845
        ->  Nested Loop Left Join  (cost=1.00..239437.36 rows=14703 width=1250) (actual time=0.941..139.751 rows=51 loops=1)
              Buffers: shared hit=20845
              ->  Nested Loop  (cost=0.70..232913.30 rows=14703 width=1242) (actual time=0.880..137.645 rows=51 loops=1)
                    Buffers: shared hit=20692
                    ->  Index Scan using records_list_created on records r  (cost=0.42..232721.22 rows=14703 width=1235) (actual time=0.818..137.230 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(created_at, id) < ROW('2026-01-31 15:23:22.324519+00'::timestamp with time zone, '0014c6a0-72df-eac0-a48b-d96b066049c6'::uuid)))
                          Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
                          Rows Removed by Filter: 20440
                          Buffers: shared hit=20689
                          SubPlan 2
                            ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.090..0.116 rows=2 loops=1)
                                  Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                                  Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                                  Heap Blocks: exact=2
                                  Buffers: shared hit=4
                                  ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.058..0.059 rows=0 loops=1)
                                        Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.056..0.057 rows=2 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                              Buffers: shared hit=2
                                        ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.000 rows=0 loops=1)
                                              Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.048..0.049 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.43 rows=1 width=40) (actual time=0.035..0.035 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.43 rows=1 width=40) (actual time=0.019..0.019 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=581
Planning Time: 3.396 ms
Execution Time: 140.678 ms
```
count:
```
Aggregate  (cost=1321920.30..1321920.31 rows=1 width=8) (actual time=877.030..877.036 rows=1 loops=1)
  Buffers: shared hit=403743
  ->  Result  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=21.523..876.143 rows=1111 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=403743
        ->  Index Scan using records_list_created on records r  (cost=0.42..1321670.75 rows=99823 width=0) (actual time=21.503..875.130 rows=1111 loops=1)
              Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
              Filter: (((requester_id IS NOT NULL) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((assignee_id IS NOT NULL) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR (hashed SubPlan 2))
              Rows Removed by Filter: 398889
              Buffers: shared hit=403743
              SubPlan 2
                ->  Bitmap Heap Scan on record_shares acl_share  (cost=8.61..12.64 rows=1 width=32) (actual time=0.111..0.139 rows=2 loops=1)
                      Recheck Cond: (((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[]))))
                      Filter: ((((principal_kind = 'user'::text) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{}'::uuid[])))) AND (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid))
                      Heap Blocks: exact=2
                      Buffers: shared hit=4
                      ->  BitmapOr  (cost=8.61..8.61 rows=1 width=0) (actual time=0.069..0.071 rows=0 loops=1)
                            Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.067..0.067 rows=2 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                  Buffers: shared hit=2
                            ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.30 rows=1 width=0) (actual time=0.000..0.001 rows=0 loops=1)
                                  Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (principal_id = ANY ('{}'::uuid[])))
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=408
Planning Time: 3.062 ms
Execution Time: 877.307 ms
```
## admin sort=-created_at page=1
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..136.74 rows=51 width=1277) (actual time=0.287..2.988 rows=51 loops=1)
  Buffers: shared hit=210
  ->  Result  (cost=1.00..530396.30 rows=199281 width=1277) (actual time=0.285..2.969 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=210
        ->  Nested Loop Left Join  (cost=1.00..528901.69 rows=199281 width=1250) (actual time=0.260..2.876 rows=51 loops=1)
              Buffers: shared hit=210
              ->  Nested Loop  (cost=0.70..463774.34 rows=199281 width=1242) (actual time=0.196..1.900 rows=51 loops=1)
                    Buffers: shared hit=57
                    ->  Index Scan using records_list_created on records r  (cost=0.42..461275.03 rows=199281 width=1235) (actual time=0.142..1.778 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Buffers: shared hit=54
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.045..0.047 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.017..0.017 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=530
Planning Time: 2.361 ms
Execution Time: 3.186 ms
```
count:
```
Finalize Aggregate  (cost=11551.37..11551.38 rows=1 width=8) (actual time=113.270..120.178 rows=1 loops=1)
  Buffers: shared hit=211359
  ->  Gather  (cost=11551.16..11551.37 rows=2 width=8) (actual time=113.086..120.166 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=211359
        ->  Partial Aggregate  (cost=10551.16..10551.17 rows=1 width=8) (actual time=108.677..108.679 rows=1 loops=3)
              Buffers: shared hit=211359
              ->  Result  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.097..95.387 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=211359
                    ->  Parallel Index Only Scan using records_list_created on records r  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.082..75.481 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=211359
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=286
Planning Time: 1.132 ms
Execution Time: 120.291 ms
```
## admin sort=-created_at page=20
```sql
SELECT ... FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840' AND r.entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2' AND ((((r.requester_id IS NOT NULL AND r.requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND (r.created_at, r.id) < ('2026-09-30 05:32:59.909441+00'::timestamptz, 'd2b91ace-2bd6-55a1-4b63-8e553d8345e4'::uuid) ORDER BY r.created_at DESC NULLS FIRST, r.id DESC LIMIT 51
```
```
Limit  (cost=1.00..137.05 rows=51 width=1277) (actual time=0.657..2.794 rows=51 loops=1)
  Buffers: shared hit=210
  ->  Result  (cost=1.00..529640.65 rows=198540 width=1277) (actual time=0.653..2.771 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
        Buffers: shared hit=210
        ->  Nested Loop Left Join  (cost=1.00..528151.60 rows=198540 width=1250) (actual time=0.608..2.659 rows=51 loops=1)
              Buffers: shared hit=210
              ->  Nested Loop  (cost=0.70..463259.52 rows=198540 width=1242) (actual time=0.281..1.650 rows=51 loops=1)
                    Buffers: shared hit=57
                    ->  Index Scan using records_list_created on records r  (cost=0.42..460769.47 rows=198540 width=1235) (actual time=0.204..1.496 rows=51 loops=1)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(created_at, id) < ROW('2026-09-30 05:32:59.909441+00'::timestamp with time zone, 'd2b91ace-2bd6-55a1-4b63-8e553d8345e4'::uuid)))
                          Buffers: shared hit=54
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.062..0.067 rows=1 loops=1)
                                Index Cond: (tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                Filter: (id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.019..0.019 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_live_key on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.016..0.016 rows=1 loops=51)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=519
Planning Time: 6.732 ms
Execution Time: 3.125 ms
```
count:
```
Finalize Aggregate  (cost=11551.37..11551.38 rows=1 width=8) (actual time=105.702..111.340 rows=1 loops=1)
  Buffers: shared hit=211337
  ->  Gather  (cost=11551.16..11551.37 rows=2 width=8) (actual time=105.540..111.328 rows=3 loops=1)
        Workers Planned: 2
        Workers Launched: 2
        Buffers: shared hit=211337
        ->  Partial Aggregate  (cost=10551.16..10551.17 rows=1 width=8) (actual time=101.001..101.003 rows=1 loops=3)
              Buffers: shared hit=211337
              ->  Result  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.091..88.435 rows=133333 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                    Buffers: shared hit=211337
                    ->  Parallel Index Only Scan using records_list_created on records r  (cost=0.42..10343.57 rows=83034 width=0) (actual time=0.079..69.350 rows=133333 loops=3)
                          Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=211337
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=286
Planning Time: 0.980 ms
Execution Time: 111.452 ms
```
