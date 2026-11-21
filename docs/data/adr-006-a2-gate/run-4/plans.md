# TICK-1048 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app)

### S-A2-cancel candidate, unbudgeted count: a6, pk = 3 (broad 50% value, 6% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=25814.37..25814.38 rows=1 width=8) (actual time=6.077..8.320 rows=1 loops=1)
  Buffers: shared hit=13315
  ->  Limit  (cost=1001.12..25805.36 rows=721 width=4) (actual time=0.196..8.251 rows=1001 loops=1)
        Buffers: shared hit=13315
        ->  Result  (cost=1001.12..25805.36 rows=721 width=4) (actual time=0.195..8.188 rows=1001 loops=1)
              Buffers: shared hit=13315
              ->  Append  (cost=1001.12..25798.15 rows=721 width=0) (actual time=0.195..8.108 rows=1001 loops=1)
                    Buffers: shared hit=13315
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1001.12..25425.18 rows=716 width=0) (actual time=0.194..8.039 rows=1001 loops=1)
                          Buffers: shared hit=13315
                          ->  Limit  (cost=1001.12..25418.02 rows=716 width=4) (actual time=0.193..7.962 rows=1001 loops=1)
                                Buffers: shared hit=13315
                                ->  Gather  (cost=1001.12..25418.02 rows=716 width=4) (actual time=0.193..7.898 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=13315
                                      ->  Result  (cost=1.11..24346.42 rows=421 width=4) (actual time=0.074..4.165 rows=643 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                            Buffers: shared hit=13315
                                            ->  Nested Loop Semi Join  (cost=1.11..24346.42 rows=421 width=0) (actual time=0.070..4.116 rows=643 loops=2)
                                                  Buffers: shared hit=13315
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..706.28 rows=3639 width=32) (actual time=0.019..0.257 rows=1329 loops=2)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1394
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..6.49 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=2658)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                        Heap Fetches: 1286
                                                        Buffers: shared hit=11921
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..184.14 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..184.12 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..184.12 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..184.12 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..62.70 rows=16 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..151.56 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..151.54 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..151.54 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..151.54 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..51.89 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.65 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.66 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.65 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.65 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.65 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_3.id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.780 ms
Execution Time: 8.414 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, three broad fields

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=10878.12..10878.13 rows=1 width=8) (actual time=63.865..69.767 rows=1 loops=1)
  Buffers: shared hit=60718 read=1
  I/O Timings: shared read=0.011
  ->  Limit  (cost=8666.49..10875.83 rows=183 width=4) (actual time=37.983..69.678 rows=1001 loops=1)
        Buffers: shared hit=60718 read=1
        I/O Timings: shared read=0.011
        ->  Result  (cost=8666.49..10875.83 rows=183 width=4) (actual time=37.982..69.601 rows=1001 loops=1)
              Buffers: shared hit=60718 read=1
              I/O Timings: shared read=0.011
              ->  Append  (cost=8666.49..10874.00 rows=183 width=0) (actual time=37.982..69.498 rows=1001 loops=1)
                    Buffers: shared hit=60718 read=1
                    I/O Timings: shared read=0.011
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8666.49..10498.22 rows=180 width=0) (actual time=37.981..69.409 rows=1001 loops=1)
                          Buffers: shared hit=60718 read=1
                          I/O Timings: shared read=0.011
                          ->  Limit  (cost=8666.49..10496.42 rows=180 width=4) (actual time=37.979..69.309 rows=1001 loops=1)
                                Buffers: shared hit=60718 read=1
                                I/O Timings: shared read=0.011
                                ->  Gather  (cost=8666.49..10496.42 rows=180 width=4) (actual time=37.979..69.231 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=60718 read=1
                                      I/O Timings: shared read=0.011
                                      ->  Result  (cost=7666.49..9478.42 rows=106 width=4) (actual time=36.138..61.631 rows=810 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                            Buffers: shared hit=60718 read=1
                                            I/O Timings: shared read=0.011
                                            ->  Nested Loop Semi Join  (cost=7666.49..9478.42 rows=106 width=0) (actual time=36.134..61.539 rows=810 loops=2)
                                                  Join Filter: (v.record_id = v_1.record_id)
                                                  Buffers: shared hit=60718 read=1
                                                  I/O Timings: shared read=0.011
                                                  ->  Nested Loop Semi Join  (cost=7666.05..8944.73 rows=457 width=64) (actual time=36.104..52.644 rows=2014 loops=2)
                                                        Buffers: shared hit=44610
                                                        ->  Parallel Hash Semi Join  (cost=7665.62..8391.00 rows=907 width=48) (actual time=36.079..40.997 rows=4026 loops=2)
                                                              Hash Cond: (r.id = v.record_id)
                                                              Buffers: shared hit=12401
                                                              ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..706.28 rows=3639 width=32) (actual time=0.061..1.731 rows=7965 loops=2)
                                                                    Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=8248
                                                              ->  Parallel Hash  (cost=6624.00..6624.00 rows=83286 width=32) (actual time=35.563..35.564 rows=99944 loops=2)
                                                                    Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                                    Buffers: shared hit=4124
                                                                    ->  Parallel Seq Scan on rfv_bool v  (cost=0.00..6624.00 rows=83286 width=32) (actual time=0.009..18.893 rows=99944 loops=2)
                                                                          Filter: (value AND (tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '185d52f3-6e4e-4573-ada0-6b9ea4165f18'::uuid))
                                                                          Rows Removed by Filter: 100056
                                                                          Buffers: shared hit=4124
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.42..0.61 rows=1 width=32) (actual time=0.003..0.003 rows=1 loops=8052)
                                                              Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v.record_id) AND (field_id = '4e78f30d-a941-412e-9acf-ed26f1004a29'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                              Rows Removed by Filter: 0
                                                              Buffers: shared hit=32209
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.43..1.16 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=4027)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v_2.record_id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                        Filter: (value = '3'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=16108 read=1
                                                        I/O Timings: shared read=0.011
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.96..186.20 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..186.19 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..186.19 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..186.19 rows=1 width=0) (never executed)
                                            Join Filter: (r_1.id = v_3.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..185.56 rows=1 width=64) (never executed)
                                                  Join Filter: (r_1.id = v_5.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..183.96 rows=2 width=48) (never executed)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..62.70 rows=16 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_5  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v_4.record_id) AND (field_id = '4e78f30d-a941-412e-9acf-ed26f1004a29'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_3  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v_5.record_id) AND (field_id = '185d52f3-6e4e-4573-ada0-6b9ea4165f18'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.96..153.65 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..153.64 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..153.64 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..153.64 rows=1 width=0) (never executed)
                                            Join Filter: (r_2.id = v_6.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..153.02 rows=1 width=64) (never executed)
                                                  Join Filter: (r_2.id = v_8.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..151.42 rows=2 width=48) (never executed)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..51.89 rows=13 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..7.65 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_8  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v_7.record_id) AND (field_id = '4e78f30d-a941-412e-9acf-ed26f1004a29'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_6  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v_8.record_id) AND (field_id = '185d52f3-6e4e-4573-ada0-6b9ea4165f18'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 4"  (cost=25.63..35.01 rows=1 width=0) (never executed)
                          ->  Limit  (cost=25.63..35.00 rows=1 width=4) (never executed)
                                ->  Result  (cost=25.63..35.00 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=25.63..35.00 rows=1 width=0) (never executed)
                                            Join Filter: (v_9.record_id = v_10.record_id)
                                            ->  Nested Loop Semi Join  (cost=25.19..33.82 rows=1 width=80) (never executed)
                                                  ->  Nested Loop Semi Join  (cost=24.77..33.21 rows=1 width=64) (never executed)
                                                        ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                              ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid))
                                                        ->  Index Scan using rfv_bool_pkey on rfv_bool v_9  (cost=0.42..0.62 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_3.id) AND (field_id = '185d52f3-6e4e-4573-ada0-6b9ea4165f18'::uuid))
                                                              Filter: value
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_11  (cost=0.42..0.61 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v_9.record_id) AND (field_id = '4e78f30d-a941-412e-9acf-ed26f1004a29'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_10  (cost=0.43..1.16 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = v_11.record_id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.607 ms
Execution Time: 69.879 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk = 20 (partly correlated, 500 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=7794.20..7794.21 rows=1 width=8) (actual time=13.321..13.326 rows=1 loops=1)
  Buffers: shared hit=17957
  ->  Limit  (cost=6675.03..7793.62 rows=46 width=4) (actual time=8.196..13.294 rows=494 loops=1)
        Buffers: shared hit=17957
        ->  Result  (cost=6675.03..7793.62 rows=46 width=4) (actual time=8.195..13.263 rows=494 loops=1)
              Buffers: shared hit=17957
              ->  Append  (cost=6675.03..7793.16 rows=46 width=0) (actual time=8.195..13.221 rows=494 loops=1)
                    Buffers: shared hit=17957
                    ->  Subquery Scan on "*SELECT* 1"  (cost=6675.03..7423.59 rows=43 width=0) (actual time=8.195..13.144 rows=494 loops=1)
                          Buffers: shared hit=17943
                          ->  Limit  (cost=6675.03..7423.16 rows=43 width=4) (actual time=8.193..13.108 rows=494 loops=1)
                                Buffers: shared hit=17943
                                ->  Result  (cost=6675.03..7423.16 rows=43 width=4) (actual time=8.193..13.079 rows=494 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=17943
                                      ->  Hash Semi Join  (cost=6675.03..7423.16 rows=43 width=0) (actual time=8.190..13.042 rows=494 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=17943
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..731.75 rows=6186 width=32) (actual time=0.017..3.088 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13059
                                            ->  Hash  (cost=6604.54..6604.54 rows=5594 width=32) (actual time=8.161..8.161 rows=21487 loops=1)
                                                  Buckets: 32768 (originally 8192)  Batches: 1 (originally 1)  Memory Usage: 1599kB
                                                  Buffers: shared hit=4884
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..6604.54 rows=5594 width=32) (actual time=0.012..6.325 rows=21487 loops=1)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '20'::double precision))
                                                        Heap Fetches: 21487
                                                        Buffers: shared hit=4884
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..184.13 rows=1 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..184.12 rows=1 width=4) (actual time=0.016..0.017 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..184.12 rows=1 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..184.12 rows=1 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..62.70 rows=16 width=32) (actual time=0.009..0.009 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '20'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..151.55 rows=1 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..151.54 rows=1 width=4) (actual time=0.009..0.009 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..151.54 rows=1 width=4) (actual time=0.008..0.009 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..151.54 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..51.89 rows=13 width=32) (actual time=0.007..0.007 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.65 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '20'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.66 rows=1 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.65 rows=1 width=4) (actual time=0.014..0.015 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.65 rows=1 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.65 rows=1 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.012..0.013 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_3.id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                  Filter: (value = '20'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.786 ms
Execution Time: 13.392 ms
```

### S-A2-cancel candidate, unbudgeted count: a50, pk = 3 (broad 50% value, 50% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=26359.07..26359.08 rows=1 width=8) (actual time=34.151..37.945 rows=1 loops=1)
  Buffers: shared hit=9309
  ->  Limit  (cost=24955.68..26346.56 rows=1001 width=4) (actual time=32.474..37.871 rows=1001 loops=1)
        Buffers: shared hit=9309
        ->  Result  (cost=24955.68..26350.73 rows=1004 width=4) (actual time=32.474..37.791 rows=1001 loops=1)
              Buffers: shared hit=9309
              ->  Append  (cost=24955.68..26340.69 rows=1004 width=0) (actual time=32.473..37.685 rows=1001 loops=1)
                    Buffers: shared hit=9309
                    ->  Subquery Scan on "*SELECT* 1"  (cost=24955.68..26018.21 rows=1001 width=0) (actual time=32.468..37.598 rows=1001 loops=1)
                          Buffers: shared hit=9309
                          ->  Limit  (cost=24955.68..26008.20 rows=1001 width=4) (actual time=32.467..37.506 rows=1001 loops=1)
                                Buffers: shared hit=9309
                                ->  Gather  (cost=24955.68..31161.45 rows=5902 width=4) (actual time=32.467..37.435 rows=1001 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=9309
                                      ->  Result  (cost=23955.68..29571.25 rows=2459 width=4) (actual time=30.251..31.687 rows=668 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                            Buffers: shared hit=9309
                                            ->  Parallel Hash Semi Join  (cost=23955.68..29571.25 rows=2459 width=0) (actual time=30.247..31.612 rows=668 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=9309
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5529.95 rows=21236 width=32) (actual time=0.049..0.721 rows=1360 loops=3)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2105
                                                  ->  Parallel Hash  (cost=23471.50..23471.50 rows=38689 width=32) (actual time=29.873..29.874 rows=51950 loops=3)
                                                        Buckets: 262144 (originally 131072)  Batches: 1 (originally 1)  Memory Usage: 0kB
                                                        Buffers: shared hit=7146
                                                        ->  Parallel Bitmap Heap Scan on rfv_num v  (cost=5720.44..23471.50 rows=38689 width=32) (actual time=6.121..16.822 rows=51950 loops=3)
                                                              Recheck Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision))
                                                              Heap Blocks: exact=1723
                                                              Buffers: shared hit=7146
                                                              ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..5697.23 rows=92854 width=0) (actual time=7.244..7.244 rows=155850 loops=1)
                                                                    Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision))
                                                                    Buffers: shared hit=2600
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..153.78 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..153.77 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..153.77 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..153.77 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..62.70 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..129.79 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..129.78 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..129.78 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..129.78 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..51.89 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.90 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.89 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) OR ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.48 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_3.id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.859 ms
Execution Time: 38.024 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk < 5 (every 3 and 4: 78% of the side rows)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=12351.92..12351.93 rows=1 width=8) (actual time=4.784..6.521 rows=1 loops=1)
  Buffers: shared hit=5350
  ->  Limit  (cost=1000.99..12339.40 rows=1001 width=4) (actual time=0.233..6.449 rows=1001 loops=1)
        Buffers: shared hit=5350
        ->  Result  (cost=1000.99..12475.33 rows=1013 width=4) (actual time=0.232..6.380 rows=1001 loops=1)
              Buffers: shared hit=5350
              ->  Append  (cost=1000.99..12465.20 rows=1013 width=0) (actual time=0.232..6.284 rows=1001 loops=1)
                    Buffers: shared hit=5350
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1000.99..12074.22 rows=1001 width=0) (actual time=0.232..6.207 rows=1001 loops=1)
                          Buffers: shared hit=5350
                          ->  Limit  (cost=1000.99..12064.21 rows=1001 width=4) (actual time=0.231..6.108 rows=1001 loops=1)
                                Buffers: shared hit=5350
                                ->  Gather  (cost=1000.99..27548.30 rows=2402 width=4) (actual time=0.231..6.031 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=5350
                                      ->  Result  (cost=0.99..26308.10 rows=1413 width=4) (actual time=0.046..2.806 rows=576 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                            Buffers: shared hit=5350
                                            ->  Nested Loop Semi Join  (cost=0.99..26308.10 rows=1413 width=0) (actual time=0.042..2.753 rows=576 loops=2)
                                                  Buffers: shared hit=5350
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..706.28 rows=3639 width=32) (actual time=0.019..0.131 rows=591 loops=2)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=621
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..7.26 rows=1 width=32) (actual time=0.004..0.004 rows=1 loops=1182)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r.id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                        Filter: (value < '5'::double precision)
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=4729
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..193.87 rows=6 width=0) (never executed)
                          ->  Limit  (cost=0.99..193.81 rows=6 width=4) (never executed)
                                ->  Result  (cost=0.99..193.81 rows=6 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..193.81 rows=6 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..62.70 rows=16 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_1.id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..158.38 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.99..158.33 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.99..158.33 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..158.33 rows=5 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..51.89 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_2.id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.66 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.65 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.65 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.65 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.24 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_3.id) AND (field_id = '312947a5-0d55-463b-816a-71dc37062223'::uuid))
                                                  Filter: (value < '5'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.945 ms
Execution Time: 6.595 ms
```

### S-A2-cancel candidate, unbudgeted count: a50, pw = 1 (704k invisible side rows, 100 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=8255.70..8255.71 rows=1 width=8) (actual time=458.037..458.043 rows=1 loops=1)
  Buffers: shared hit=910951
  ->  Limit  (cost=1.11..8243.18 rows=1001 width=4) (actual time=2.499..457.977 rows=100 loops=1)
        Buffers: shared hit=910951
        ->  Result  (cost=1.11..8325.52 rows=1011 width=4) (actual time=2.498..457.935 rows=100 loops=1)
              Buffers: shared hit=910951
              ->  Append  (cost=1.11..8315.41 rows=1011 width=0) (actual time=2.498..457.900 rows=100 loops=1)
                    Buffers: shared hit=910951
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1.11..7994.72 rows=1001 width=0) (actual time=2.497..457.817 rows=100 loops=1)
                          Buffers: shared hit=910937
                          ->  Limit  (cost=1.11..7984.71 rows=1001 width=4) (actual time=2.497..457.780 rows=100 loops=1)
                                Buffers: shared hit=910937
                                ->  Result  (cost=1.11..157830.56 rows=19789 width=4) (actual time=2.496..457.747 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=910937
                                      ->  Nested Loop Semi Join  (cost=1.11..157830.56 rows=19789 width=0) (actual time=2.493..457.700 rows=100 loops=1)
                                            Buffers: shared hit=910937
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5827.25 rows=50966 width=32) (actual time=0.043..32.088 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109320
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..3.00 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid) AND (value = '1'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 100
                                                  Buffers: shared hit=801617
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..152.78 rows=5 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..152.73 rows=5 width=4) (actual time=0.023..0.024 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..152.73 rows=5 width=4) (actual time=0.021..0.022 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..152.73 rows=5 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..62.70 rows=12 width=32) (actual time=0.011..0.011 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..128.97 rows=4 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..128.93 rows=4 width=4) (actual time=0.009..0.010 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..128.93 rows=4 width=4) (actual time=0.009..0.009 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..128.93 rows=4 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..51.89 rows=10 width=32) (actual time=0.007..0.007 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.91..33.89 rows=1 width=0) (actual time=0.017..0.018 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.91..33.88 rows=1 width=4) (actual time=0.016..0.017 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.91..33.88 rows=1 width=4) (actual time=0.015..0.017 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.91..33.88 rows=1 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.014..0.015 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) OR ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,35a1585b-68c6-46ae-aa56-810d4428570d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10fcc76c-58b8-40df-9b74-f6a960c1344b'::uuid)) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.55..1.47 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid) AND (value = '1'::double precision) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.962 ms
Execution Time: 458.112 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pw = 1 (704k invisible side rows, 100 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=11424.17..11424.18 rows=1 width=8) (actual time=35.407..38.763 rows=1 loops=1)
  Buffers: shared hit=109407
  ->  Limit  (cost=1001.12..11411.66 rows=1001 width=4) (actual time=0.269..38.741 rows=100 loops=1)
        Buffers: shared hit=109407
        ->  Result  (cost=1001.12..11536.46 rows=1013 width=4) (actual time=0.268..38.730 rows=100 loops=1)
              Buffers: shared hit=109407
              ->  Append  (cost=1001.12..11526.33 rows=1013 width=0) (actual time=0.268..38.713 rows=100 loops=1)
                    Buffers: shared hit=109407
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1001.12..11154.09 rows=1001 width=0) (actual time=0.267..38.646 rows=100 loops=1)
                          Buffers: shared hit=109393
                          ->  Limit  (cost=1001.12..11144.08 rows=1001 width=4) (actual time=0.266..38.629 rows=100 loops=1)
                                Buffers: shared hit=109393
                                ->  Gather  (cost=1001.12..25340.19 rows=2402 width=4) (actual time=0.266..38.619 rows=100 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=109393
                                      ->  Result  (cost=1.11..24099.99 rows=1413 width=4) (actual time=1.132..33.540 rows=50 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                            Buffers: shared hit=109393
                                            ->  Nested Loop Semi Join  (cost=1.11..24099.99 rows=1413 width=0) (actual time=1.128..33.525 rows=50 loops=2)
                                                  Buffers: shared hit=109393
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..706.28 rows=3639 width=32) (actual time=0.026..2.838 rows=12028 loops=2)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=13070
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..6.49 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=24055)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid) AND (value = '1'::double precision) AND (record_id = r.id))
                                                        Heap Fetches: 100
                                                        Buffers: shared hit=96323
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..182.94 rows=6 width=0) (actual time=0.020..0.022 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..182.88 rows=6 width=4) (actual time=0.019..0.021 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..182.88 rows=6 width=4) (actual time=0.018..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..182.88 rows=6 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..62.70 rows=16 width=32) (actual time=0.010..0.010 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..150.55 rows=5 width=0) (actual time=0.011..0.013 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..150.50 rows=5 width=4) (actual time=0.011..0.012 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..150.50 rows=5 width=4) (actual time=0.011..0.011 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..150.50 rows=5 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..51.89 rows=13 width=32) (actual time=0.009..0.009 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.65 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.69 rows=1 width=0) (actual time=0.014..0.018 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.68 rows=1 width=4) (actual time=0.014..0.018 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.68 rows=1 width=4) (actual time=0.014..0.017 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.68 rows=1 width=0) (actual time=0.013..0.015 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.012..0.014 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.009..0.010 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,e61dedcf-415a-4e68-bddb-ffa3112384ac}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'ff7c9a42-0634-446a-b5bb-7c4a0b447b21'::uuid)) AND (entity_definition_id = '01a20047-c3bc-74dd-ada9-069074f30636'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.27 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'b9f4acd3-3e61-477d-b10c-da4356830078'::uuid) AND (record_id = r_3.id) AND (field_id = 'd3dcfbe7-8a8d-4537-8d9c-225ac4e36d28'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.926 ms
Execution Time: 38.838 ms
```
