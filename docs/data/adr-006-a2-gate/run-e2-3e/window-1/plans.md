# TICK-1048 / TICK-1083 / TICK-1098 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app)

### (3e) a50, pk = 3, arm A (shipped: CountBudget 170ms, workers off in the savepoint Exec)

Replayed on one connection, in `tenancy.InTenantTx` for the large tenant:

```sql
SAVEPOINT list_count; SET LOCAL statement_timeout = '170'; SET LOCAL max_parallel_workers_per_gather = 0;
-- harness addition, so that ANALYZE can finish (not a planner setting):
SET LOCAL statement_timeout = 0;
EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Gather: false; workers launched: false

```
Aggregate  (cost=28159.39..28159.40 rows=1 width=8) (actual time=65.452..65.464 rows=1 loops=1)
  Buffers: shared hit=9223, temp written=516
  I/O Timings: temp write=2.209
  ->  Limit  (cost=26770.14..28146.88 rows=1001 width=4) (actual time=62.904..65.385 rows=1001 loops=1)
        Buffers: shared hit=9223, temp written=516
        I/O Timings: temp write=2.209
        ->  Result  (cost=26770.14..28151.01 rows=1004 width=4) (actual time=62.903..65.301 rows=1001 loops=1)
              Buffers: shared hit=9223, temp written=516
              I/O Timings: temp write=2.209
              ->  Append  (cost=26770.14..28140.97 rows=1004 width=0) (actual time=62.902..65.197 rows=1001 loops=1)
                    Buffers: shared hit=9223, temp written=516
                    I/O Timings: temp write=2.209
                    ->  Subquery Scan on "*SELECT* 1"  (cost=26770.14..27816.33 rows=1001 width=0) (actual time=62.900..65.107 rows=1001 loops=1)
                          Buffers: shared hit=9223, temp written=516
                          I/O Timings: temp write=2.209
                          ->  Limit  (cost=26770.14..27806.32 rows=1001 width=4) (actual time=62.899..65.023 rows=1001 loops=1)
                                Buffers: shared hit=9223, temp written=516
                                I/O Timings: temp write=2.209
                                ->  Result  (cost=26770.14..32638.42 rows=5669 width=4) (actual time=62.898..64.955 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                      Buffers: shared hit=9223, temp written=516
                                      I/O Timings: temp write=2.209
                                      ->  Hash Semi Join  (cost=26770.14..32638.42 rows=5669 width=0) (actual time=62.890..64.855 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=9223, temp written=516
                                            I/O Timings: temp write=2.209
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5647.40 rows=49532 width=32) (actual time=0.034..0.802 rows=4044 loops=1)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=2077
                                            ->  Hash  (cost=25623.66..25623.66 rows=91673 width=32) (actual time=62.764..62.765 rows=155850 loops=1)
                                                  Buckets: 131072 (originally 131072)  Batches: 2 (originally 1)  Memory Usage: 7169kB
                                                  Buffers: shared hit=7146, temp written=494
                                                  I/O Timings: temp write=2.103
                                                  ->  Bitmap Heap Scan on rfv_num v  (cost=6945.38..25623.66 rows=91673 width=32) (actual time=9.082..32.339 rows=155850 loops=1)
                                                        Recheck Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision))
                                                        Heap Blocks: exact=4546
                                                        Buffers: shared hit=7146
                                                        ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..6922.47 rows=91673 width=0) (actual time=8.513..8.513 rows=155850 loops=1)
                                                              Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision))
                                                              Buffers: shared hit=2600
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..159.71 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..159.70 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..159.70 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..159.70 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.64 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..125.98 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..125.97 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..125.97 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..125.97 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.67 rows=9 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.69 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.92 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.91 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.91 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.91 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) OR ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.50 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (record_id = r_3.id) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', max_parallel_workers_per_gather = '0'
Planning Time: 1.689 ms
Execution Time: 66.663 ms
```

### (3e) a50, pk = 3, arm B (WithCountArm(140 ms, true): server settings)

Replayed on one connection, in `tenancy.InTenantTx` for the large tenant:

```sql
SAVEPOINT list_count; SET LOCAL statement_timeout = '140';
-- harness addition, so that ANALYZE can finish (not a planner setting):
SET LOCAL statement_timeout = 0;
EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

Gather: true; workers launched: true

```
Aggregate  (cost=27579.75..27579.76 rows=1 width=8) (actual time=43.508..49.696 rows=1 loops=1)
  Buffers: shared hit=9420
  ->  Limit  (cost=26165.86..27567.24 rows=1001 width=4) (actual time=41.817..49.627 rows=1001 loops=1)
        Buffers: shared hit=9420
        ->  Result  (cost=26165.86..27571.44 rows=1004 width=4) (actual time=41.815..49.552 rows=1001 loops=1)
              Buffers: shared hit=9420
              ->  Append  (cost=26165.86..27561.40 rows=1004 width=0) (actual time=41.814..49.454 rows=1001 loops=1)
                    Buffers: shared hit=9420
                    ->  Subquery Scan on "*SELECT* 1"  (cost=26165.86..27236.76 rows=1001 width=0) (actual time=41.812..49.373 rows=1001 loops=1)
                          Buffers: shared hit=9420
                          ->  Limit  (cost=26165.86..27226.75 rows=1001 width=4) (actual time=41.810..49.284 rows=1001 loops=1)
                                Buffers: shared hit=9420
                                ->  Gather  (cost=26165.86..32174.08 rows=5669 width=4) (actual time=41.809..49.217 rows=1001 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=9420
                                      ->  Result  (cost=25165.86..30607.18 rows=2362 width=4) (actual time=36.199..37.758 rows=709 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                            Buffers: shared hit=9420
                                            ->  Parallel Hash Semi Join  (cost=25165.86..30607.18 rows=2362 width=0) (actual time=36.187..37.680 rows=709 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=9420
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5358.46 rows=20638 width=32) (actual time=0.174..0.912 rows=1439 loops=3)
                                                        Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2216
                                                  ->  Parallel Hash  (cost=24687.83..24687.83 rows=38197 width=32) (actual time=35.337..35.338 rows=51950 loops=3)
                                                        Buckets: 262144 (originally 131072)  Batches: 1 (originally 1)  Memory Usage: 0kB
                                                        Buffers: shared hit=7146
                                                        ->  Parallel Bitmap Heap Scan on rfv_num v  (cost=6945.38..24687.83 rows=38197 width=32) (actual time=10.104..21.374 rows=51950 loops=3)
                                                              Recheck Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision))
                                                              Heap Blocks: exact=1711
                                                              Buffers: shared hit=7146
                                                              ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..6922.47 rows=91673 width=0) (actual time=13.124..13.124 rows=155850 loops=1)
                                                                    Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision))
                                                                    Buffers: shared hit=2600
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..159.71 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..159.70 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..159.70 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..159.70 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.64 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..125.98 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..125.97 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..125.97 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..125.97 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.67 rows=9 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.69 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.92 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.91 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.91 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.91 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) OR ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5935bbda-31f3-43ad-b529-8bb64ad4d2b3}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '581abb7d-0702-477a-abdf-2e7ffda60ad5'::uuid)) AND (entity_definition_id = '01a20457-1082-704c-96f1-3d900bb4a2b0'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.50 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '5b87c026-b5a9-40fc-bec8-f027847e8863'::uuid) AND (record_id = r_3.id) AND (field_id = 'ffd62370-e297-40b2-92fb-0b1e3f8090d7'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.033 ms
Execution Time: 49.906 ms
```
