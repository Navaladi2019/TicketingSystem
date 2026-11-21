# TICK-1022: the count statement's plans, pk=3 (48%) as a narrowing agent

EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app in the agent's tenant transaction, exactly as the handler sent it. Full captures were 0.5-0.6 MB a run and are not kept; rerun the harness to regenerate them.

## states: none pk=3 (48%), count statement

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=68.23..68.24 rows=1 width=8) (actual time=184.262..184.269 rows=1 loops=1)
  Buffers: shared hit=76983, temp written=1518
  I/O Timings: temp write=6.157
  ->  Limit  (cost=8.89..68.18 rows=4 width=4) (actual time=121.329..184.145 rows=1001 loops=1)
        Buffers: shared hit=76983, temp written=1518
        I/O Timings: temp write=6.157
        ->  Result  (cost=8.89..68.18 rows=4 width=4) (actual time=121.327..184.040 rows=1001 loops=1)
              Buffers: shared hit=76983, temp written=1518
              I/O Timings: temp write=6.157
              ->  Append  (cost=8.89..68.14 rows=4 width=0) (actual time=121.325..183.903 rows=1001 loops=1)
                    Buffers: shared hit=76983, temp written=1518
                    I/O Timings: temp write=6.157
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8.89..16.95 rows=1 width=0) (actual time=121.324..183.782 rows=1001 loops=1)
                          Buffers: shared hit=76983, temp written=1518
                          I/O Timings: temp write=6.157
                          ->  Limit  (cost=8.89..16.93 rows=1 width=4) (actual time=121.322..183.655 rows=1001 loops=1)
                                Buffers: shared hit=76983, temp written=1518
                                I/O Timings: temp write=6.157
                                ->  Result  (cost=8.89..16.93 rows=1 width=4) (actual time=121.322..183.549 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      Buffers: shared hit=76983, temp written=1518
                                      I/O Timings: temp write=6.157
                                      ->  Nested Loop  (cost=8.89..16.93 rows=1 width=0) (actual time=121.306..183.391 rows=1001 loops=1)
                                            Buffers: shared hit=76983, temp written=1518
                                            I/O Timings: temp write=6.157
                                            ->  HashAggregate  (cost=8.45..8.46 rows=1 width=32) (actual time=121.257..126.239 rows=17293 loops=1)
                                                  Group Key: v.tenant_id, v.record_id
                                                  Batches: 5  Memory Usage: 8265kB  Disk Usage: 7520kB
                                                  Buffers: shared hit=7811, temp written=1518
                                                  I/O Timings: temp write=6.157
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..8.44 rows=1 width=32) (actual time=0.052..51.007 rows=192501 loops=1)
                                                        Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision))
                                                        Heap Fetches: 192501
                                                        Buffers: shared hit=7811
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=17293)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (id = v.record_id))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=69172
                    ->  Subquery Scan on "*SELECT* 2"  (cost=8.89..16.95 rows=1 width=0) (never executed)
                          ->  Limit  (cost=8.89..16.94 rows=1 width=4) (never executed)
                                ->  Result  (cost=8.89..16.94 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop  (cost=8.89..16.94 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=8.45..8.46 rows=1 width=32) (never executed)
                                                  Group Key: v_1.tenant_id, v_1.record_id
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..8.44 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision))
                                                        Heap Fetches: 0
                                            ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (id = v_1.record_id))
                                                  Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=8.89..16.95 rows=1 width=0) (never executed)
                          ->  Limit  (cost=8.89..16.94 rows=1 width=4) (never executed)
                                ->  Result  (cost=8.89..16.94 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop  (cost=8.89..16.94 rows=1 width=0) (never executed)
                                            ->  HashAggregate  (cost=8.45..8.46 rows=1 width=32) (never executed)
                                                  Group Key: v_2.tenant_id, v_2.record_id
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..8.44 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision))
                                                        Heap Fetches: 0
                                            ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (id = v_2.record_id))
                                                  Filter: ((assignee_id IS NOT NULL) AND ((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=9.17..17.27 rows=1 width=0) (never executed)
                          ->  Limit  (cost=9.17..17.26 rows=1 width=4) (never executed)
                                ->  Result  (cost=9.17..17.26 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop Semi Join  (cost=9.17..17.26 rows=1 width=0) (never executed)
                                            Join Filter: (v_3.record_id = acl_share.record_id)
                                            ->  Nested Loop  (cost=8.88..16.92 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=8.45..8.46 rows=1 width=32) (never executed)
                                                        Group Key: v_3.tenant_id, v_3.record_id
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..8.44 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (id = v_3.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using record_shares_pkey on record_shares acl_share  (cost=0.29..0.32 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (record_id = r_3.id))
                                                  Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.123 ms
Execution Time: 185.980 ms
```

## states: analyze pk=3 (48%), count statement

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=18301.43..18301.44 rows=1 width=8) (actual time=165.557..165.571 rows=1 loops=1)
  Buffers: shared hit=6718, temp written=625
  I/O Timings: temp write=8.265
  ->  Limit  (cost=15248.90..18288.92 rows=1001 width=4) (actual time=161.230..165.442 rows=1001 loops=1)
        Buffers: shared hit=6718, temp written=625
        I/O Timings: temp write=8.265
        ->  Result  (cost=15248.90..18392.18 rows=1035 width=4) (actual time=161.229..165.302 rows=1001 loops=1)
              Buffers: shared hit=6718, temp written=625
              I/O Timings: temp write=8.265
              ->  Append  (cost=15248.90..18381.83 rows=1035 width=0) (actual time=161.228..165.109 rows=1001 loops=1)
                    Buffers: shared hit=6718, temp written=625
                    I/O Timings: temp write=8.265
                    ->  Subquery Scan on "*SELECT* 1"  (cost=15248.90..16636.44 rows=1001 width=0) (actual time=161.226..164.962 rows=1001 loops=1)
                          Buffers: shared hit=6718, temp written=625
                          I/O Timings: temp write=8.265
                          ->  Limit  (cost=15248.90..16626.43 rows=1001 width=4) (actual time=161.225..164.789 rows=1001 loops=1)
                                Buffers: shared hit=6718, temp written=625
                                I/O Timings: temp write=8.265
                                ->  Result  (cost=15248.90..17187.89 rows=1409 width=4) (actual time=161.223..164.676 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      Buffers: shared hit=6718, temp written=625
                                      I/O Timings: temp write=8.265
                                      ->  Hash Semi Join  (cost=15248.90..17187.89 rows=1409 width=0) (actual time=161.215..164.512 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=6718, temp written=625
                                            I/O Timings: temp write=8.265
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.41 rows=5889 width=32) (actual time=0.034..1.492 rows=4088 loops=1)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=2172
                                            ->  Hash  (cost=11546.00..11546.00 rows=191467 width=32) (actual time=161.070..161.072 rows=192501 loops=1)
                                                  Buckets: 131072  Batches: 2  Memory Usage: 7017kB
                                                  Buffers: shared hit=4546, temp written=613
                                                  I/O Timings: temp write=8.145
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..11546.00 rows=191467 width=32) (actual time=0.012..92.720 rows=192501 loops=1)
                                                        Filter: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision))
                                                        Rows Removed by Filter: 207499
                                                        Buffers: shared hit=4546
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1538.51 rows=30 width=0) (never executed)
                          ->  Limit  (cost=0.86..1538.21 rows=30 width=4) (never executed)
                                ->  Result  (cost=0.86..1538.21 rows=30 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1538.21 rows=30 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..501.46 rows=124 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.35 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (record_id = r_1.id) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid))
                                                  Filter: (value = '3'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..156.59 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..156.56 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..156.56 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..156.56 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..55.06 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.84..45.12 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.84..45.11 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.84..45.11 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.84..45.11 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.64 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (record_id = r_3.id) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.797 ms
Execution Time: 167.508 ms
```

## states: vacuum pk=3 (48%), count statement

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=13850.64..13850.65 rows=1 width=8) (actual time=21.499..21.519 rows=1 loops=1)
  Buffers: shared hit=7253
  ->  Limit  (cost=0.98..13838.13 rows=1001 width=4) (actual time=0.236..21.228 rows=1001 loops=1)
        Buffers: shared hit=7253
        ->  Result  (cost=0.98..14308.12 rows=1035 width=4) (actual time=0.234..20.937 rows=1001 loops=1)
              Buffers: shared hit=7253
              ->  Append  (cost=0.98..14297.77 rows=1035 width=0) (actual time=0.233..20.585 rows=1001 loops=1)
                    Buffers: shared hit=7253
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.98..13088.45 rows=1001 width=0) (actual time=0.232..20.302 rows=1001 loops=1)
                          Buffers: shared hit=7253
                          ->  Limit  (cost=0.98..13078.44 rows=1001 width=4) (actual time=0.230..19.800 rows=1001 loops=1)
                                Buffers: shared hit=7253
                                ->  Result  (cost=0.98..18395.64 rows=1408 width=4) (actual time=0.229..19.586 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      Buffers: shared hit=7253
                                      ->  Nested Loop Semi Join  (cost=0.98..18395.64 rows=1408 width=0) (actual time=0.215..19.290 rows=1001 loops=1)
                                            Buffers: shared hit=7253
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.41 rows=5889 width=32) (actual time=0.043..1.469 rows=2044 loops=1)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1120
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..3.03 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=2044)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=6133
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1050.51 rows=30 width=0) (never executed)
                          ->  Limit  (cost=0.86..1050.21 rows=30 width=4) (never executed)
                                ->  Result  (cost=0.86..1050.21 rows=30 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1050.21 rows=30 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..501.46 rows=124 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.42 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..108.59 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..108.56 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..108.56 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..108.56 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..55.06 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.84..45.05 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.84..45.04 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.84..45.04 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.84..45.04 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'cdcad520-929e-4c0a-b4a0-9ca140d8b7de'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.57 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'bcccd137-8242-44f1-b212-12e345ff2d88'::uuid) AND (field_id = 'a2a36d11-28e5-4c76-9f6b-95594d781879'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.679 ms
Execution Time: 21.744 ms
```

## sweep: churn5 pk=3 (48%), count statement

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=18442.26..18442.27 rows=1 width=8) (actual time=100.506..100.513 rows=1 loops=1)
  Buffers: shared hit=6943, temp written=625
  I/O Timings: temp write=3.963
  ->  Limit  (cost=15521.23..18429.75 rows=1001 width=4) (actual time=97.445..100.425 rows=1001 loops=1)
        Buffers: shared hit=6943, temp written=625
        I/O Timings: temp write=3.963
        ->  Result  (cost=15521.23..18522.73 rows=1033 width=4) (actual time=97.444..100.327 rows=1001 loops=1)
              Buffers: shared hit=6943, temp written=625
              I/O Timings: temp write=3.963
              ->  Append  (cost=15521.23..18512.40 rows=1033 width=0) (actual time=97.444..100.198 rows=1001 loops=1)
                    Buffers: shared hit=6943, temp written=625
                    I/O Timings: temp write=3.963
                    ->  Subquery Scan on "*SELECT* 1"  (cost=15521.23..16950.17 rows=1001 width=0) (actual time=97.443..100.095 rows=1001 loops=1)
                          Buffers: shared hit=6943, temp written=625
                          I/O Timings: temp write=3.963
                          ->  Limit  (cost=15521.23..16940.16 rows=1001 width=4) (actual time=97.442..99.990 rows=1001 loops=1)
                                Buffers: shared hit=6943, temp written=625
                                I/O Timings: temp write=3.963
                                ->  Result  (cost=15521.23..17449.05 rows=1360 width=4) (actual time=97.441..99.909 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      Buffers: shared hit=6943, temp written=625
                                      I/O Timings: temp write=3.963
                                      ->  Hash Semi Join  (cost=15521.23..17449.05 rows=1360 width=0) (actual time=97.436..99.777 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=6943, temp written=625
                                            I/O Timings: temp write=3.963
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..494.51 rows=5613 width=32) (actual time=0.026..1.042 rows=4088 loops=1)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=2172
                                            ->  Hash  (cost=11771.00..11771.00 rows=193893 width=32) (actual time=97.289..97.290 rows=192501 loops=1)
                                                  Buckets: 131072  Batches: 2  Memory Usage: 7017kB
                                                  Buffers: shared hit=4771, temp written=613
                                                  I/O Timings: temp write=3.876
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..11771.00 rows=193893 width=32) (actual time=0.010..60.066 rows=192501 loops=1)
                                                        Filter: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid) AND (value = '3'::double precision))
                                                        Rows Removed by Filter: 207499
                                                        Buffers: shared hit=4771
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1364.04 rows=28 width=0) (never executed)
                          ->  Limit  (cost=0.86..1363.76 rows=28 width=4) (never executed)
                                ->  Result  (cost=0.86..1363.76 rows=28 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1363.76 rows=28 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..411.59 rows=114 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..8.34 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..147.90 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..147.87 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..147.87 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..147.87 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..46.36 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.84..45.13 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.84..45.12 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.84..45.12 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.84..45.12 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.65 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (record_id = r_3.id) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.415 ms
Execution Time: 101.655 ms
```

## sweep: rpc pk=3 (48%), count statement

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=7757.42..7757.43 rows=1 width=8) (actual time=13.319..13.330 rows=1 loops=1)
  Buffers: shared hit=9296
  ->  Limit  (cost=0.98..7744.90 rows=1001 width=4) (actual time=0.119..13.163 rows=1001 loops=1)
        Buffers: shared hit=9296
        ->  Result  (cost=0.98..7992.46 rows=1033 width=4) (actual time=0.118..13.008 rows=1001 loops=1)
              Buffers: shared hit=9296
              ->  Append  (cost=0.98..7982.13 rows=1033 width=0) (actual time=0.117..12.805 rows=1001 loops=1)
                    Buffers: shared hit=9296
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.98..7497.64 rows=1001 width=0) (actual time=0.117..12.639 rows=1001 loops=1)
                          Buffers: shared hit=9296
                          ->  Limit  (cost=0.98..7487.63 rows=1001 width=4) (actual time=0.116..12.452 rows=1001 loops=1)
                                Buffers: shared hit=9296
                                ->  Result  (cost=0.98..10165.17 rows=1359 width=4) (actual time=0.115..12.296 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      Buffers: shared hit=9296
                                      ->  Nested Loop Semi Join  (cost=0.98..10165.17 rows=1359 width=0) (actual time=0.106..12.094 rows=1001 loops=1)
                                            Buffers: shared hit=9296
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..239.31 rows=5613 width=32) (actual time=0.045..1.043 rows=2044 loops=1)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1120
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..1.76 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=2044)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (record_id = r.id) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid))
                                                  Filter: (value = '3'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=8176
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..419.20 rows=28 width=0) (never executed)
                          ->  Limit  (cost=0.86..418.92 rows=28 width=4) (never executed)
                                ->  Result  (cost=0.86..418.92 rows=28 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..418.92 rows=28 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..118.16 rows=114 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..2.63 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..45.50 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..45.47 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..45.47 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..45.47 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..13.56 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..2.65 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=12.29..14.63 rows=1 width=0) (never executed)
                          ->  Limit  (cost=12.29..14.62 rows=1 width=4) (never executed)
                                ->  Result  (cost=12.29..14.62 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid)
                                      ->  Nested Loop Semi Join  (cost=12.29..14.62 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=11.85..14.09 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=11.43..11.44 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=6.98..11.42 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=6.98..6.98 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..1.40 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..5.58 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..2.66 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01c7a34b-1d13-48c8-9f78-84ff58e1ff60'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.50 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '3c22096e-ef14-402e-bc79-3c60b66c7677'::uuid) AND (record_id = r_3.id) AND (field_id = '44fad517-6d34-4992-9b54-6a8bd484b530'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', random_page_cost = '1.1', jit = 'off'
Planning Time: 2.150 ms
Execution Time: 13.551 ms
```
