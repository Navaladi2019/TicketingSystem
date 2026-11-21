# TICK-1048 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app)

### S-A2-cancel candidate, unbudgeted count: a6, pk = 3 (broad 50% value, 6% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=21037.88..21037.89 rows=1 width=8) (actual time=58.914..64.272 rows=1 loops=1)
  Buffers: shared hit=12645
  ->  Limit  (cost=19934.30..21025.37 rows=1001 width=4) (actual time=57.499..64.212 rows=1001 loops=1)
        Buffers: shared hit=12645
        ->  Result  (cost=19934.30..21031.91 rows=1007 width=4) (actual time=57.499..64.150 rows=1001 loops=1)
              Buffers: shared hit=12645
              ->  Append  (cost=19934.30..21021.84 rows=1007 width=0) (actual time=57.498..64.066 rows=1001 loops=1)
                    Buffers: shared hit=12645
                    ->  Subquery Scan on "*SELECT* 1"  (cost=19934.30..20707.35 rows=1001 width=0) (actual time=57.497..63.996 rows=1001 loops=1)
                          Buffers: shared hit=12645
                          ->  Limit  (cost=19934.30..20697.34 rows=1001 width=4) (actual time=57.496..63.921 rows=1001 loops=1)
                                Buffers: shared hit=12645
                                ->  Gather  (cost=19934.30..20731.64 rows=1046 width=4) (actual time=57.495..63.834 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=12645
                                      ->  Result  (cost=18934.30..19627.04 rows=615 width=4) (actual time=55.864..57.010 rows=698 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                            Buffers: shared hit=12645
                                            ->  Parallel Hash Semi Join  (cost=18934.30..19627.04 rows=615 width=0) (actual time=55.859..56.952 rows=698 loops=2)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=12645
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..677.27 rows=3498 width=32) (actual time=0.050..0.362 rows=1456 loops=2)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1527
                                                  ->  Parallel Hash  (cost=18203.96..18203.96 rows=58382 width=32) (actual time=55.298..55.299 rows=77925 loops=2)
                                                        Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                        Buffers: shared hit=11089
                                                        ->  Parallel Seq Scan on rfv_num v  (cost=0.00..18203.96 rows=58382 width=32) (actual time=13.179..41.117 rows=77925 loops=2)
                                                              Filter: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision))
                                                              Rows Removed by Filter: 409958
                                                              Buffers: shared hit=11089
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..150.47 rows=3 width=0) (never executed)
                          ->  Limit  (cost=1.11..150.44 rows=3 width=4) (never executed)
                                ->  Result  (cost=1.11..150.44 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..150.44 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..45.66 rows=15 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.97 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..125.57 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..125.55 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..125.55 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..125.55 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..38.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.41 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.40 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.40 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.40 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.00 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_3.id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.793 ms
Execution Time: 64.358 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, three broad fields

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=10661.78..10661.79 rows=1 width=8) (actual time=64.440..69.832 rows=1 loops=1)
  Buffers: shared hit=61801
  ->  Limit  (cost=8666.49..10658.45 rows=266 width=4) (actual time=39.344..69.742 rows=1001 loops=1)
        Buffers: shared hit=61801
        ->  Result  (cost=8666.49..10658.45 rows=266 width=4) (actual time=39.343..69.666 rows=1001 loops=1)
              Buffers: shared hit=61801
              ->  Append  (cost=8666.49..10655.79 rows=266 width=0) (actual time=39.343..69.563 rows=1001 loops=1)
                    Buffers: shared hit=61801
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8666.49..10339.22 rows=263 width=0) (actual time=39.342..69.473 rows=1001 loops=1)
                          Buffers: shared hit=61801
                          ->  Limit  (cost=8666.49..10336.59 rows=263 width=4) (actual time=39.341..69.375 rows=1001 loops=1)
                                Buffers: shared hit=61801
                                ->  Gather  (cost=8666.49..10336.59 rows=263 width=4) (actual time=39.339..69.292 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=61801
                                      ->  Result  (cost=7666.48..9310.29 rows=155 width=4) (actual time=37.627..62.344 rows=824 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                            Buffers: shared hit=61801
                                            ->  Nested Loop Semi Join  (cost=7666.48..9310.29 rows=155 width=0) (actual time=37.623..62.254 rows=824 loops=2)
                                                  Buffers: shared hit=61801
                                                  ->  Nested Loop Semi Join  (cost=7666.05..8897.30 rows=441 width=64) (actual time=37.554..53.995 rows=2052 loops=2)
                                                        Buffers: shared hit=45388
                                                        ->  Parallel Hash Semi Join  (cost=7665.62..8361.29 rows=878 width=48) (actual time=37.529..42.362 rows=4104 loops=2)
                                                              Hash Cond: (r.id = v.record_id)
                                                              Buffers: shared hit=12555
                                                              ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..677.27 rows=3498 width=32) (actual time=0.028..1.680 rows=8133 loops=2)
                                                                    Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=8402
                                                              ->  Parallel Hash  (cost=6624.00..6624.00 rows=83286 width=32) (actual time=37.051..37.052 rows=99944 loops=2)
                                                                    Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                                    Buffers: shared hit=4124
                                                                    ->  Parallel Seq Scan on rfv_bool v  (cost=0.00..6624.00 rows=83286 width=32) (actual time=0.008..18.201 rows=99944 loops=2)
                                                                          Filter: (value AND (tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '1927c991-56a2-47ba-aa07-bcc7d72569b6'::uuid))
                                                                          Rows Removed by Filter: 100056
                                                                          Buffers: shared hit=4124
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.42..0.61 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=8208)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v.record_id) AND (field_id = '3e3460d1-89b5-4ef5-9f5a-7a51135b93e4'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=32833
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..0.93 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=4103)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v.record_id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                        Filter: (value = '3'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=16413
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.96..153.00 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..152.99 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..152.99 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..152.99 rows=1 width=0) (never executed)
                                            Join Filter: (r_1.id = v_3.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..152.37 rows=1 width=64) (never executed)
                                                  Join Filter: (r_1.id = v_5.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..150.31 rows=3 width=48) (never executed)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..45.66 rows=15 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..6.97 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_5  (cost=0.42..0.67 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v_4.record_id) AND (field_id = '3e3460d1-89b5-4ef5-9f5a-7a51135b93e4'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_3  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v_5.record_id) AND (field_id = '1927c991-56a2-47ba-aa07-bcc7d72569b6'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.96..127.45 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..127.44 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..127.44 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..127.44 rows=1 width=0) (never executed)
                                            Join Filter: (r_2.id = v_6.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..126.81 rows=1 width=64) (never executed)
                                                  Join Filter: (r_2.id = v_8.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..125.44 rows=2 width=48) (never executed)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..38.52 rows=12 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..7.24 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_8  (cost=0.42..0.67 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v_7.record_id) AND (field_id = '3e3460d1-89b5-4ef5-9f5a-7a51135b93e4'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_6  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v_8.record_id) AND (field_id = '1927c991-56a2-47ba-aa07-bcc7d72569b6'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 4"  (cost=25.63..34.79 rows=1 width=0) (never executed)
                          ->  Limit  (cost=25.63..34.78 rows=1 width=4) (never executed)
                                ->  Result  (cost=25.63..34.78 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=25.63..34.78 rows=1 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=25.19..33.83 rows=1 width=80) (never executed)
                                                  ->  Nested Loop Semi Join  (cost=24.77..33.22 rows=1 width=64) (never executed)
                                                        ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                              ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid))
                                                        ->  Index Scan using rfv_bool_pkey on rfv_bool v_9  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_3.id) AND (field_id = '1927c991-56a2-47ba-aa07-bcc7d72569b6'::uuid))
                                                              Filter: value
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_11  (cost=0.42..0.61 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v_9.record_id) AND (field_id = '3e3460d1-89b5-4ef5-9f5a-7a51135b93e4'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_10  (cost=0.42..0.93 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = v_9.record_id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.585 ms
Execution Time: 69.954 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk = 20 (partly correlated, 500 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=8574.94..8574.95 rows=1 width=8) (actual time=13.620..13.626 rows=1 loops=1)
  Buffers: shared hit=17947
  ->  Limit  (cost=7545.14..8574.05 rows=71 width=4) (actual time=8.663..13.595 rows=494 loops=1)
        Buffers: shared hit=17947
        ->  Result  (cost=7545.14..8574.05 rows=71 width=4) (actual time=8.662..13.565 rows=494 loops=1)
              Buffers: shared hit=17947
              ->  Append  (cost=7545.14..8573.34 rows=71 width=0) (actual time=8.662..13.525 rows=494 loops=1)
                    Buffers: shared hit=17947
                    ->  Subquery Scan on "*SELECT* 1"  (cost=7545.14..8263.56 rows=68 width=0) (actual time=8.662..13.447 rows=494 loops=1)
                          Buffers: shared hit=17933
                          ->  Limit  (cost=7545.14..8262.88 rows=68 width=4) (actual time=8.661..13.411 rows=494 loops=1)
                                Buffers: shared hit=17933
                                ->  Result  (cost=7545.14..8262.88 rows=68 width=4) (actual time=8.661..13.376 rows=494 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=17933
                                      ->  Hash Semi Join  (cost=7545.14..8262.88 rows=68 width=0) (actual time=8.658..13.338 rows=494 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=17933
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..701.75 rows=5946 width=32) (actual time=0.012..2.993 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=7431.36..7431.36 rows=9058 width=32) (actual time=8.632..8.632 rows=21487 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1599kB
                                                  Buffers: shared hit=4884
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..7431.36 rows=9058 width=32) (actual time=0.010..6.649 rows=21487 loops=1)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '20'::double precision))
                                                        Heap Fetches: 21487
                                                        Buffers: shared hit=4884
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..150.45 rows=1 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..150.44 rows=1 width=4) (actual time=0.020..0.021 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..150.44 rows=1 width=4) (actual time=0.020..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..150.44 rows=1 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..45.66 rows=15 width=32) (actual time=0.016..0.016 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.97 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '20'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..125.56 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..125.55 rows=1 width=4) (actual time=0.008..0.008 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..125.55 rows=1 width=4) (actual time=0.007..0.008 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..125.55 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..38.52 rows=12 width=32) (actual time=0.006..0.006 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '20'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.41 rows=1 width=0) (actual time=0.012..0.014 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.40 rows=1 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.40 rows=1 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.40 rows=1 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.010..0.011 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.003..0.003 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.00 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_3.id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value = '20'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.904 ms
Execution Time: 13.685 ms
```

### S-A2-cancel candidate, unbudgeted count: a50, pk = 3 (broad 50% value, 50% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=12611.88..12611.89 rows=1 width=8) (actual time=80.138..80.143 rows=1 loops=1)
  Buffers: shared hit=109383, temp read=492 written=1035
  I/O Timings: temp read=0.474 write=3.608
  ->  Limit  (cost=9517.75..12599.37 rows=1001 width=4) (actual time=79.106..80.086 rows=1001 loops=1)
        Buffers: shared hit=109383, temp read=492 written=1035
        I/O Timings: temp read=0.474 write=3.608
        ->  Result  (cost=9517.75..12614.76 rows=1006 width=4) (actual time=79.105..80.027 rows=1001 loops=1)
              Buffers: shared hit=109383, temp read=492 written=1035
              I/O Timings: temp read=0.474 write=3.608
              ->  Append  (cost=9517.75..12604.70 rows=1006 width=0) (actual time=79.104..79.948 rows=1001 loops=1)
                    Buffers: shared hit=109383, temp read=492 written=1035
                    I/O Timings: temp read=0.474 write=3.608
                    ->  Subquery Scan on "*SELECT* 1"  (cost=9517.75..12324.98 rows=1001 width=0) (actual time=79.104..79.893 rows=1001 loops=1)
                          Buffers: shared hit=109383, temp read=492 written=1035
                          I/O Timings: temp read=0.474 write=3.608
                          ->  Limit  (cost=9517.75..12314.97 rows=1001 width=4) (actual time=79.103..79.834 rows=1001 loops=1)
                                Buffers: shared hit=109383, temp read=492 written=1035
                                I/O Timings: temp read=0.474 write=3.608
                                ->  Result  (cost=9517.75..33784.61 rows=8684 width=4) (actual time=79.102..79.785 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=109383, temp read=492 written=1035
                                      I/O Timings: temp read=0.474 write=3.608
                                      ->  Merge Semi Join  (cost=9517.75..33784.61 rows=8684 width=0) (actual time=79.095..79.713 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=109383, temp read=492 written=1035
                                            I/O Timings: temp read=0.474 write=3.608
                                            ->  Sort  (cost=9515.95..9639.37 rows=49369 width=32) (actual time=79.061..79.207 rows=2026 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: external merge  Disk: 8256kB
                                                  Buffers: shared hit=109312, temp read=492 written=1035
                                                  I/O Timings: temp read=0.474 write=3.608
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..5667.31 rows=49369 width=32) (actual time=0.026..31.851 rows=200379 loops=1)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109312
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..23585.91 rows=140116 width=32) (actual time=0.028..0.238 rows=1126 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 1126
                                                  Buffers: shared hit=71
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..132.71 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..132.69 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..132.69 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..132.69 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..45.66 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..108.40 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..108.38 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..108.38 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..108.38 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..38.52 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.97 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.57 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.56 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.56 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.56 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) OR ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.16 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_3.id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.200 ms
Execution Time: 81.020 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk < 5 (every 3 and 4: 78% of the side rows)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=15541.48..15541.49 rows=1 width=8) (actual time=2.803..2.807 rows=1 loops=1)
  Buffers: shared hit=4629
  ->  Limit  (cost=0.99..15528.97 rows=1001 width=4) (actual time=0.022..2.731 rows=1001 loops=1)
        Buffers: shared hit=4629
        ->  Result  (cost=0.99..15730.63 rows=1014 width=4) (actual time=0.022..2.673 rows=1001 loops=1)
              Buffers: shared hit=4629
              ->  Append  (cost=0.99..15720.49 rows=1014 width=0) (actual time=0.022..2.592 rows=1001 loops=1)
                    Buffers: shared hit=4629
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..15369.27 rows=1001 width=0) (actual time=0.020..2.525 rows=1001 loops=1)
                          Buffers: shared hit=4629
                          ->  Limit  (cost=0.99..15359.26 rows=1001 width=4) (actual time=0.020..2.454 rows=1001 loops=1)
                                Buffers: shared hit=4629
                                ->  Result  (cost=0.99..41472.92 rows=2703 width=4) (actual time=0.020..2.394 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=4629
                                      ->  Nested Loop Semi Join  (cost=0.99..41472.92 rows=2703 width=0) (actual time=0.017..2.320 rows=1001 loops=1)
                                            Buffers: shared hit=4629
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..701.75 rows=5946 width=32) (actual time=0.012..0.159 rows=1022 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=541
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..6.85 rows=1 width=32) (actual time=0.002..0.002 rows=1 loops=1022)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r.id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value < '5'::double precision)
                                                  Rows Removed by Filter: 0
                                                  Buffers: shared hit=4088
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..172.64 rows=7 width=0) (never executed)
                          ->  Limit  (cost=0.99..172.57 rows=7 width=4) (never executed)
                                ->  Result  (cost=0.99..172.57 rows=7 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..172.57 rows=7 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..45.66 rows=15 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_1.id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..140.10 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.99..140.05 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.99..140.05 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..140.05 rows=5 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..38.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_2.id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.41 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.40 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.40 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.40 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.00 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_3.id) AND (field_id = '0dc7ced0-8298-4402-a66f-a595c43b3a97'::uuid))
                                                  Filter: (value < '5'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.724 ms
Execution Time: 2.856 ms
```

### S-A2-cancel candidate, unbudgeted count: a50, pw = 1 (176k invisible side rows, 100 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17637.81..17637.82 rows=1 width=8) (actual time=133.285..133.293 rows=1 loops=1)
  Buffers: shared hit=114257, temp read=1686 written=1686
  I/O Timings: temp read=1.666 write=5.890
  ->  Limit  (cost=14453.34..17625.30 rows=1001 width=4) (actual time=52.324..133.264 rows=100 loops=1)
        Buffers: shared hit=114257, temp read=1686 written=1686
        I/O Timings: temp read=1.666 write=5.890
        ->  Result  (cost=14453.34..17634.81 rows=1004 width=4) (actual time=52.323..133.248 rows=100 loops=1)
              Buffers: shared hit=114257, temp read=1686 written=1686
              I/O Timings: temp read=1.666 write=5.890
              ->  Append  (cost=14453.34..17624.77 rows=1004 width=0) (actual time=52.323..133.222 rows=100 loops=1)
                    Buffers: shared hit=114257, temp read=1686 written=1686
                    I/O Timings: temp read=1.666 write=5.890
                    ->  Subquery Scan on "*SELECT* 1"  (cost=14453.34..17345.08 rows=1001 width=0) (actual time=52.322..133.128 rows=100 loops=1)
                          Buffers: shared hit=114243, temp read=1686 written=1686
                          I/O Timings: temp read=1.666 write=5.890
                          ->  Limit  (cost=14453.34..17335.07 rows=1001 width=4) (actual time=52.320..133.103 rows=100 loops=1)
                                Buffers: shared hit=114243, temp read=1686 written=1686
                                I/O Timings: temp read=1.666 write=5.890
                                ->  Result  (cost=14453.34..20277.26 rows=2023 width=4) (actual time=52.320..133.089 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=114243, temp read=1686 written=1686
                                      I/O Timings: temp read=1.666 write=5.890
                                      ->  Hash Semi Join  (cost=14453.34..20277.26 rows=2023 width=0) (actual time=52.313..133.059 rows=100 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=114243, temp read=1686 written=1686
                                            I/O Timings: temp read=1.666 write=5.890
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5667.31 rows=49369 width=32) (actual time=0.028..28.285 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109312
                                            ->  Hash  (cost=14044.82..14044.82 rows=32637 width=32) (actual time=48.375..48.376 rows=175766 loops=1)
                                                  Buckets: 131072 (originally 32768)  Batches: 2 (originally 1)  Memory Usage: 7169kB
                                                  Buffers: shared hit=4931, temp written=559
                                                  I/O Timings: temp write=1.954
                                                  ->  Bitmap Heap Scan on rfv_num v  (cost=2384.67..14044.82 rows=32637 width=32) (actual time=7.510..23.568 rows=175766 loops=1)
                                                        Recheck Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision))
                                                        Heap Blocks: exact=1999
                                                        Buffers: shared hit=4931
                                                        ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..2376.51 rows=32637 width=0) (actual time=7.314..7.315 rows=175766 loops=1)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision))
                                                              Buffers: shared hit=2932
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..132.70 rows=1 width=0) (actual time=0.035..0.037 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..132.69 rows=1 width=4) (actual time=0.034..0.036 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..132.69 rows=1 width=4) (actual time=0.033..0.034 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..132.69 rows=1 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..45.66 rows=12 width=32) (actual time=0.023..0.023 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..108.39 rows=1 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..108.38 rows=1 width=4) (actual time=0.012..0.013 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..108.38 rows=1 width=4) (actual time=0.012..0.012 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..108.38 rows=1 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..38.52 rows=10 width=32) (actual time=0.010..0.010 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.97 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.57 rows=1 width=0) (actual time=0.017..0.019 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.56 rows=1 width=4) (actual time=0.017..0.018 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.56 rows=1 width=4) (actual time=0.017..0.018 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.56 rows=1 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.015..0.016 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.015..0.016 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.013..0.013 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) OR ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,3306f153-b7ba-489b-85e9-42c032eae116}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '38c6409c-b0c9-4f90-99cf-bf8dd77ab2b3'::uuid)) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.16 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_3.id) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.971 ms
Execution Time: 133.403 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pw = 1 (176k invisible side rows, 100 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=15492.14..15492.15 rows=1 width=8) (actual time=59.337..59.345 rows=1 loops=1)
  Buffers: shared hit=17994, temp read=695 written=695
  I/O Timings: temp read=0.651 write=2.319
  ->  Limit  (cost=14453.34..15489.05 rows=247 width=4) (actual time=44.492..59.329 rows=100 loops=1)
        Buffers: shared hit=17994, temp read=695 written=695
        I/O Timings: temp read=0.651 write=2.319
        ->  Result  (cost=14453.34..15489.05 rows=247 width=4) (actual time=44.492..59.320 rows=100 loops=1)
              Buffers: shared hit=17994, temp read=695 written=695
              I/O Timings: temp read=0.651 write=2.319
              ->  Append  (cost=14453.34..15486.58 rows=247 width=0) (actual time=44.491..59.309 rows=100 loops=1)
                    Buffers: shared hit=17994, temp read=695 written=695
                    I/O Timings: temp read=0.651 write=2.319
                    ->  Subquery Scan on "*SELECT* 1"  (cost=14453.34..15175.92 rows=244 width=0) (actual time=44.491..59.243 rows=100 loops=1)
                          Buffers: shared hit=17980, temp read=695 written=695
                          I/O Timings: temp read=0.651 write=2.319
                          ->  Limit  (cost=14453.34..15173.48 rows=244 width=4) (actual time=44.490..59.229 rows=100 loops=1)
                                Buffers: shared hit=17980, temp read=695 written=695
                                I/O Timings: temp read=0.651 write=2.319
                                ->  Result  (cost=14453.34..15173.48 rows=244 width=4) (actual time=44.489..59.220 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=17980, temp read=695 written=695
                                      I/O Timings: temp read=0.651 write=2.319
                                      ->  Hash Semi Join  (cost=14453.34..15173.48 rows=244 width=0) (actual time=44.486..59.204 rows=100 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=17980, temp read=695 written=695
                                            I/O Timings: temp read=0.651 write=2.319
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..701.75 rows=5946 width=32) (actual time=0.017..3.485 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13049
                                            ->  Hash  (cost=14044.82..14044.82 rows=32637 width=32) (actual time=44.412..44.413 rows=175766 loops=1)
                                                  Buckets: 131072 (originally 32768)  Batches: 2 (originally 1)  Memory Usage: 7169kB
                                                  Buffers: shared hit=4931, temp written=559
                                                  I/O Timings: temp write=1.872
                                                  ->  Bitmap Heap Scan on rfv_num v  (cost=2384.67..14044.82 rows=32637 width=32) (actual time=7.159..21.736 rows=175766 loops=1)
                                                        Recheck Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision))
                                                        Heap Blocks: exact=1999
                                                        Buffers: shared hit=4931
                                                        ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..2376.51 rows=32637 width=0) (actual time=6.967..6.967 rows=175766 loops=1)
                                                              Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision))
                                                              Buffers: shared hit=2932
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..150.45 rows=1 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..150.44 rows=1 width=4) (actual time=0.024..0.024 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..150.44 rows=1 width=4) (actual time=0.023..0.023 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..150.44 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..45.66 rows=15 width=32) (actual time=0.015..0.015 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.97 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..125.56 rows=1 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..125.55 rows=1 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..125.55 rows=1 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..125.55 rows=1 width=0) (actual time=0.008..0.009 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..38.52 rows=12 width=32) (actual time=0.008..0.008 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.41 rows=1 width=0) (actual time=0.015..0.017 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.40 rows=1 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.40 rows=1 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.40 rows=1 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.013..0.014 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb977aa9-99a7-4a04-b550-e0525098cf78}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'e56fa8d9-ad4f-442d-8161-fe49e8863f3f'::uuid)) AND (entity_definition_id = '01a20044-423a-7429-a645-e84ae51ac8ae'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.00 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'fdd4c469-9838-4ab7-9a02-c50e42a5ab4a'::uuid) AND (record_id = r_3.id) AND (field_id = '125d4916-0f14-4523-be06-b91f31330f7a'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.878 ms
Execution Time: 59.444 ms
```
